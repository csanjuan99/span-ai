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
	"strings"
	"time"

	"span/internal/fsutil"
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
	// SourceIssue is the Jira key the idea was read from, if any.
	SourceIssue string
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
	if err := checkRepo(in.Workspace, in.Repo); err != nil {
		return nil, err
	}
	id, err := newID(now)
	if err != nil {
		return nil, err
	}
	return &Session{
		ID: id, Workspace: in.Workspace, Repo: in.Repo, Idea: in.Idea, Agent: in.Agent, Model: in.Model, SourceIssue: in.SourceIssue,
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
	s.UpdatedAt = time.Now().UTC()
	raw, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("encode session: %w", err)
	}
	if err := writeAtomic(path, append(raw, '\n')); err != nil {
		return err
	}
	return recordRevision(s)
}

// writeAtomic keeps session files readable by the group, as they always were.
func writeAtomic(path string, data []byte) error {
	return fsutil.WriteAtomicMode(path, data, 0o644)
}

// checkRepo refuses a repo that is not the workspace or a folder inside it, so the
// interview never runs in, or reads, another project. Links are followed first, so a
// link inside the workspace that points outside is refused too.
func checkRepo(workspace, repo string) error {
	if repo == "" {
		return nil
	}
	root := resolved(workspace)
	target := repo
	if !filepath.IsAbs(target) {
		target = filepath.Join(root, target)
	}
	rel, err := filepath.Rel(root, resolved(target))
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return fail("shape_repo_outside", "repo %q is not inside the workspace %s", repo, workspace)
	}
	return nil
}

// resolved is the path with its links followed, or the cleaned path when it does not exist.
func resolved(path string) string {
	if real, err := filepath.EvalSymlinks(path); err == nil {
		return real
	}
	return filepath.Clean(path)
}

// SetArchived hides or shows a session in the default list. A sealed session can be
// archived: the flag is not part of its story or digest.
func SetArchived(workspace, id string, archived bool) (*Session, error) {
	s, err := Load(workspace, id)
	if err != nil {
		return nil, err
	}
	s.Archived = archived
	if err := Save(s); err != nil {
		return nil, err
	}
	return s, nil
}

// Delete removes a session and its folder. A sealed session is a record of a decision and
// is refused with shape_sealed: archive it instead.
func Delete(workspace, id string) error {
	s, err := Load(workspace, id)
	if err != nil {
		return err
	}
	if s.Status == StatusSealed {
		return fail("shape_sealed", "session %s is sealed and cannot be deleted; archive it instead", id)
	}
	path, err := sessionPath(workspace, id)
	if err != nil {
		return err
	}
	if err := os.RemoveAll(filepath.Dir(path)); err != nil {
		return fmt.Errorf("delete session %s: %w", id, err)
	}
	return nil
}
