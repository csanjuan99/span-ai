package agent

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"strings"
	"sync"
	"time"

	"span/internal/proc"
)

// Model is one choice for a CLI: the value passed to its model flag and a label to read.
type Model struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// ModelList is what the desk offers for one CLI. Free says any valid model name may be
// typed too, because the CLI cannot list its models.
type ModelList struct {
	Models []Model `json:"models"`
	Free   bool    `json:"free"`
	// Source is "cli" (listed by the CLI), "fixed" (aliases Span knows) or "none".
	Source string `json:"source"`
}

const (
	listTimeout = 15 * time.Second
	listLimit   = 1 << 20
	listTTL     = time.Minute
	// A failed list is remembered for less: a login fixes it, and waiting a minute for the
	// next try would be worse than a second spawn.
	failTTL = 30 * time.Second
)

// fixedModels are the aliases a CLI accepts but cannot list. Claude resolves each alias to its
// newest model, so the list does not go stale.
var fixedModels = map[string][]Model{
	"claude": {
		{ID: "opus", Label: "Opus"},
		{ID: "sonnet", Label: "Sonnet"},
		{ID: "haiku", Label: "Haiku"},
	},
}

// listArgs are the arguments that make a CLI print its models. A CLI not here cannot list.
var listArgs = map[string][]string{
	"cursor": {"--list-models"},
	"pi":     {"--list-models"},
}

// runList runs a CLI's list command and returns its output. Tests replace it.
var runList = func(ctx context.Context, binary string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, listTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, args...)
	proc.Tame(cmd)
	var out bytes.Buffer
	cmd.Stdout = &limitWriter{buf: &out, left: listLimit}
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return out.String(), nil
}

type limitWriter struct {
	buf  *bytes.Buffer
	left int
}

func (w *limitWriter) Write(p []byte) (int, error) {
	n := len(p)
	if n > w.left {
		p = p[:w.left]
	}
	w.left -= len(p)
	w.buf.Write(p)
	return n, nil
}

var (
	listMu    sync.Mutex
	listCache = map[string]cachedList{}
	// listLocks make concurrent requests for one CLI wait for the first instead of each
	// starting a process.
	listLocks = map[string]*sync.Mutex{}
)

type cachedList struct {
	at   time.Time
	list ModelList
	err  string
}

func lockFor(id string) *sync.Mutex {
	listMu.Lock()
	defer listMu.Unlock()
	if listLocks[id] == nil {
		listLocks[id] = &sync.Mutex{}
	}
	return listLocks[id]
}

// Models is what the desk can offer for the CLI with this id. A CLI that lists is asked, at
// most once a minute (half that after a failure), and concurrent requests share one answer. A
// failure is returned so the page can say the list is unavailable and still let the person
// type a name.
func Models(ctx context.Context, id string) (ModelList, error) {
	spec, ok := Find(id)
	if !ok {
		return ModelList{Source: "none"}, nil
	}
	if fixed, ok := fixedModels[spec.ID]; ok {
		return ModelList{Models: fixed, Free: true, Source: "fixed"}, nil
	}
	args, ok := listArgs[spec.ID]
	if !ok {
		return ModelList{Free: true, Source: "none"}, nil
	}
	lock := lockFor(spec.ID)
	lock.Lock()
	defer lock.Unlock()
	listMu.Lock()
	cached, hit := listCache[spec.ID]
	listMu.Unlock()
	if hit {
		if cached.err == "" && time.Since(cached.at) < listTTL {
			return cached.list, nil
		}
		if cached.err != "" && time.Since(cached.at) < failTTL {
			return cached.list, errors.New(cached.err)
		}
	}
	out, err := runList(ctx, spec.Binary, args...)
	if err != nil {
		failed := ModelList{Free: true, Source: "none"}
		listMu.Lock()
		listCache[spec.ID] = cachedList{at: time.Now(), list: failed, err: err.Error()}
		listMu.Unlock()
		return failed, err
	}
	list := ModelList{Models: ParseModels(spec.ID, out), Free: true, Source: "cli"}
	listMu.Lock()
	listCache[spec.ID] = cachedList{at: time.Now(), list: list}
	listMu.Unlock()
	return list, nil
}

// ParseModels reads the output of a CLI's list command. Every name goes through ValidModel:
// a line that is not a safe model name is dropped, never offered.
func ParseModels(id, out string) []Model {
	var models []Model
	seen := map[string]bool{}
	add := func(value, label string) {
		if ValidModel(value) && !seen[value] {
			seen[value] = true
			models = append(models, Model{ID: value, Label: label})
		}
	}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		switch id {
		case "cursor":
			// "gpt-5.3-codex - Codex 5.3"; the header and blank lines have no " - ".
			if value, label, ok := strings.Cut(line, " - "); ok && !strings.ContainsAny(value, " \t") {
				add(value, strings.TrimSpace(label))
			}
		case "pi":
			// "provider  model  context ..." with a header row, passed on as provider/model.
			fields := strings.Fields(line)
			if len(fields) >= 2 && fields[0] != "provider" {
				add(fields[0]+"/"+fields[1], fields[0]+" "+fields[1])
			}
		}
	}
	return models
}
