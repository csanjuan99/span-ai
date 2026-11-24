package shape

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"time"
)

var idPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,80}$`)

// StartInput is what a person gives to begin a session. Repo is optional and is a folder
// inside Workspace, or an absolute path.
type StartInput struct {
	Workspace string
	Repo      string
	Idea      string
	Agent     string
	Model     string
}

// Dir is the folder holding the sessions of a workspace.
func Dir(workspace string) string {
	return filepath.Join(workspace, ".span", "shapes")
}

func notFound(id string) error {
	return fail("shape_not_found", "no shaping session %q", id)
}

func validID(id string) bool {
	return idPattern.MatchString(id) && id != ".." && id != "."
}

func sessionPath(workspace, id string) (string, error) {
	if !validID(id) {
		return "", notFound(id)
	}
	return filepath.Join(Dir(workspace), id, "session.json"), nil
}

func newID(now time.Time) (string, error) {
	var b [3]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("session id: %w", err)
	}
	return now.UTC().Format("20060102-150405.000000000") + "-" + hex.EncodeToString(b[:]), nil
}

func newSession(in StartInput, now time.Time) (*Session, error) {
	id, err := newID(now)
	if err != nil {
		return nil, err
	}
	return &Session{
		ID: id, Workspace: in.Workspace, Repo: in.Repo, Idea: in.Idea, Agent: in.Agent, Model: in.Model,
		Status: StatusInterviewing,
		Map:    Map{Rules: []Rule{}, Examples: []Example{}, Questions: []Question{}},
		Split:  []Split{}, Proposals: []Proposal{}, Turns: []Turn{},
	}, nil
}

// Create stores a new empty session with a unique id.
func Create(workspace string, in StartInput) (*Session, error) {
	in.Workspace = workspace
	s, err := newSession(in, time.Now())
	if err != nil {
		return nil, err
	}
	if err := Save(s); err != nil {
		return nil, err
	}
	return s, nil
}

// Load reads one session. An id that is not a plain name, or has no session, is
// shape_not_found.
func Load(workspace, id string) (*Session, error) {
	path, err := sessionPath(workspace, id)
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, notFound(id)
	}
	if err != nil {
		return nil, fmt.Errorf("read session: %w", err)
	}
	var s Session
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, fmt.Errorf("session %s: %w", id, err)
	}
	return &s, nil
}

// List returns every session of the workspace, newest first. Ids start with their
// creation time, so the order needs no extra field. A session that cannot be read is
// reported in the returned error (naming its id) while the others are still returned.
func List(workspace string) ([]*Session, error) {
	entries, err := os.ReadDir(Dir(workspace))
	if errors.Is(err, fs.ErrNotExist) {
		return []*Session{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	out := []*Session{}
	var loadErrs []error
	for _, e := range entries {
		if !e.IsDir() || !validID(e.Name()) {
			continue
		}
		s, err := Load(workspace, e.Name())
		if err != nil {
			loadErrs = append(loadErrs, fmt.Errorf("session %s: %w", e.Name(), err))
			continue
		}
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID > out[j].ID })
	return out, errors.Join(loadErrs...)
}

// Save writes the session atomically: a temporary file in the same folder, then a rename.
func Save(s *Session) error {
	path, err := sessionPath(s.Workspace, s.ID)
	if err != nil {
		return err
	}
	raw, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("encode session: %w", err)
	}
	return writeAtomic(path, append(raw, '\n'))
}

func writeAtomic(path string, data []byte) (err error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create %s: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("temp file: %w", err)
	}
	name := tmp.Name()
	defer func() {
		if err == nil {
			return
		}
		if rmErr := os.Remove(name); rmErr != nil && !errors.Is(rmErr, fs.ErrNotExist) {
			err = errors.Join(err, rmErr)
		}
	}()
	if _, werr := tmp.Write(data); werr != nil {
		return errors.Join(fmt.Errorf("write %s: %w", path, werr), tmp.Close())
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close %s: %w", path, err)
	}
	if err := os.Chmod(name, 0o644); err != nil {
		return fmt.Errorf("chmod %s: %w", path, err)
	}
	if err := os.Rename(name, path); err != nil {
		return fmt.Errorf("replace %s: %w", path, err)
	}
	return nil
}
