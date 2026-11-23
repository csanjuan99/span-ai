// Package transcript appends agent stdout and keeps a secret out of the file.
package transcript

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const flushEvery = 500 * time.Millisecond

// Batch is one flush of the log: the lines and the byte range they fill in the file,
// [Start, End). Each line takes len(line)+1 bytes, its text and a newline.
type Batch struct {
	Stage string
	Lines []string
	Start int64
	End   int64
}

// Log is the durable agent stdout for one run.
type Log struct {
	mu      sync.Mutex
	file    *os.File
	out     io.Writer
	secrets []string
	pending string
	partial string
	stage   string
	lines   []string
	// size is the file size after the last write; start and end bound the lines not yet
	// taken. They move under mu with the bytes, so a batch's range is exact.
	size  int64
	start int64
	end   int64
	// err is the first write failure. Write never returns it, because failing the
	// agent's stdout would stall the agent; Close reports it.
	err     error
	publish func(Batch)
	// sendMu is taken before mu is released, so batches are published in take order.
	// publish must never call back into the log.
	sendMu sync.Mutex
	// afterTake and beforePublish let tests force interleavings; both are nil otherwise.
	afterTake     func(lines []string)
	beforePublish func()
	stop          chan struct{}
	stopped       chan struct{}
	closed        bool
}

// Open creates or appends path and flushes new lines about every 500ms. publish gets the
// batches in the order they were taken, each with its exact byte range in the file.
func Open(path string, secrets []string, publish func(Batch)) (*Log, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	var size int64
	if info, err := file.Stat(); err == nil {
		size = info.Size()
	}
	log := &Log{
		file:    file,
		out:     file,
		size:    size,
		start:   size,
		end:     size,
		secrets: secrets,
		publish: publish,
		stop:    make(chan struct{}),
		stopped: make(chan struct{}),
	}
	go log.loop()
	return log, nil
}

// SetStage flushes the previous stage and tags the next lines.
func (l *Log) SetStage(stage string) {
	l.mu.Lock()
	prev := l.take()
	l.stage = stage
	l.handOff(prev)
}

// Write redacts secrets before they reach the file.
func (l *Log) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return len(p), nil
	}
	l.pending += string(p)
	l.drain(false)
	return len(p), nil
}

// Close flushes the remainder and stops the ticker.
func (l *Log) Close() error {
	l.mu.Lock()
	if !l.closed {
		l.drain(true)
		l.closed = true
	}
	last := l.take()
	l.handOff(last)
	select {
	case <-l.stop:
	default:
		close(l.stop)
	}
	<-l.stopped
	closeErr := l.file.Close()
	l.mu.Lock()
	writeErr := l.err
	l.mu.Unlock()
	return errors.Join(writeErr, closeErr)
}

func (l *Log) loop() {
	defer close(l.stopped)
	tick := time.NewTicker(flushEvery)
	defer tick.Stop()
	for {
		select {
		case <-l.stop:
			return
		case <-tick.C:
			l.mu.Lock()
			l.handOff(l.take())
		}
	}
}

func (l *Log) drain(finish bool) {
	text := scrub(l.pending, l.secrets)
	emit := text
	if !finish {
		var hold string
		emit, hold = holdSecret(text, l.secrets)
		l.pending = hold
	} else {
		l.pending = ""
	}
	if emit == "" {
		return
	}
	combined := l.partial + emit
	parts := strings.Split(combined, "\n")
	l.partial = parts[len(parts)-1]
	lines := parts[:len(parts)-1]
	if finish && l.partial != "" {
		lines = append(lines, l.partial)
		l.partial = ""
	}
	if len(lines) == 0 {
		return
	}
	n, err := io.WriteString(l.out, strings.Join(lines, "\n")+"\n")
	l.keep(lines, n)
	if err != nil && l.err == nil {
		l.err = err
	}
}

// keep records what one write put in the file: n bytes, of which only the lines that
// were written whole join the batch. A torn tail still counts toward size, so the next
// batch starts where the file really ends.
func (l *Log) keep(lines []string, n int) {
	if len(l.lines) == 0 {
		l.start = l.size
		l.end = l.size
	}
	left := n
	for _, line := range lines {
		width := len(line) + 1
		if width > left {
			break
		}
		left -= width
		l.lines = append(l.lines, line)
		l.end += int64(width)
	}
	l.size += int64(n)
}

// take empties the batch; the caller holds mu.
func (l *Log) take() Batch {
	batch := Batch{Stage: l.stage, Lines: l.lines, Start: l.start, End: l.end}
	l.lines = nil
	l.start = l.size
	l.end = l.size
	if len(batch.Lines) > 0 && l.afterTake != nil {
		l.afterTake(batch.Lines)
	}
	return batch
}

// handOff publishes a taken batch. It is called with mu held and releases it, taking
// sendMu first so a later take cannot be published before this one.
func (l *Log) handOff(batch Batch) {
	if len(batch.Lines) == 0 || l.publish == nil {
		l.mu.Unlock()
		return
	}
	hook := l.beforePublish
	l.sendMu.Lock()
	l.mu.Unlock()
	defer l.sendMu.Unlock()
	if hook != nil {
		hook()
	}
	l.publish(batch)
}

func scrub(text string, secrets []string) string {
	for _, secret := range secrets {
		if secret == "" {
			continue
		}
		text = strings.ReplaceAll(text, secret, "[redacted]")
	}
	return text
}

func holdSecret(text string, secrets []string) (string, string) {
	hold := 0
	for _, secret := range secrets {
		if secret == "" {
			continue
		}
		n := len(secret) - 1
		if n > len(text) {
			n = len(text)
		}
		for ; n > 0; n-- {
			if strings.HasSuffix(text, secret[:n]) && n > hold {
				hold = n
			}
		}
	}
	if hold == 0 {
		return text, ""
	}
	return text[:len(text)-hold], text[len(text)-hold:]
}

// Tail returns the last n lines. A missing file is empty.
func Tail(path string, n int) ([]string, error) {
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	const maxBytes = 1 << 20
	if len(raw) > maxBytes {
		raw = raw[len(raw)-maxBytes:]
		if i := strings.IndexByte(string(raw), '\n'); i >= 0 {
			raw = raw[i+1:]
		}
	}
	text := strings.TrimRight(string(raw), "\n")
	if text == "" {
		return nil, nil
	}
	lines := strings.Split(text, "\n")
	if n > 0 && len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return lines, nil
}
