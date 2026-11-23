// Package transcript appends agent stdout and keeps a secret out of the file.
package transcript

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const flushEvery = 500 * time.Millisecond

// Log is the durable agent stdout for one run.
type Log struct {
	mu      sync.Mutex
	file    *os.File
	secrets []string
	pending string
	partial string
	stage   string
	batch   []string
	publish func(stage string, lines []string)
	stop    chan struct{}
	stopped chan struct{}
	closed  bool
}

// Open creates or appends path and flushes new lines about every 500ms.
func Open(path string, secrets []string, publish func(stage string, lines []string)) (*Log, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	log := &Log{
		file:    file,
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
	prev, lines := l.take()
	l.stage = stage
	l.mu.Unlock()
	l.send(prev, lines)
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
	stage, lines := l.take()
	l.mu.Unlock()
	l.send(stage, lines)
	select {
	case <-l.stop:
	default:
		close(l.stop)
	}
	<-l.stopped
	return l.file.Close()
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
			stage, lines := l.take()
			l.mu.Unlock()
			l.send(stage, lines)
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
	_, _ = l.file.WriteString(strings.Join(lines, "\n") + "\n")
	l.batch = append(l.batch, lines...)
}

func (l *Log) take() (string, []string) {
	lines := l.batch
	l.batch = nil
	return l.stage, lines
}

func (l *Log) send(stage string, lines []string) {
	if len(lines) == 0 || l.publish == nil {
		return
	}
	l.publish(stage, lines)
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
