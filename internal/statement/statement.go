// Package statement is the on-disk record of one stage.
// The digest covers every field except Digest, so a later stage can point at
// the exact bytes of the previous one.
package statement

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	Advance = "advance"
	Hold    = "hold"
	Stop    = "stop"
)

// Command is one declared check, not a shell script.
type Command struct {
	Command  string `json:"command"`
	ExitCode int    `json:"exitCode"`
	Error    string `json:"error,omitempty"`
	Band     string `json:"band,omitempty"`
}

// Statement is the control record for one stage in one iteration.
type Statement struct {
	SpanID             string    `json:"spanID"`
	Stage              string    `json:"stage"`
	Iteration          int       `json:"iteration"`
	Verdict            string    `json:"verdict"`
	ParentDigest       string    `json:"parentDigest,omitempty"`
	BriefDigest        string    `json:"briefDigest,omitempty"`
	FeatureDoc         string    `json:"featureDoc,omitempty"`
	GitSHA             string    `json:"gitSHA,omitempty"`
	ChangeURL          string    `json:"changeURL,omitempty"`
	Environment        string    `json:"environment,omitempty"`
	Commands           []Command `json:"commands,omitempty"`
	Reason             string    `json:"reason"`
	InstructionsDigest string    `json:"instructionsDigest,omitempty"`
	ManifestDigest     string    `json:"manifestDigest,omitempty"`
	HandoffDigests     []string  `json:"handoffDigests,omitempty"`
	Attributable       bool      `json:"attributable,omitempty"`
	Time               time.Time `json:"time"`
	Digest             string    `json:"digest,omitempty"`
}

// Seal fills Digest from the canonical JSON of every other field.
func (s *Statement) Seal() error {
	s.Digest = ""
	raw, err := json.Marshal(s)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(raw)
	s.Digest = hex.EncodeToString(sum[:])
	return nil
}

// Write seals the statement and stores JSON plus a short markdown report.
func Write(dir string, s *Statement) error {
	if err := s.Seal(); err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	if err := writeAtomic(filepath.Join(dir, s.Stage+".json"), raw); err != nil {
		return err
	}
	return writeAtomic(filepath.Join(dir, s.Stage+".md"), []byte(markdown(s)))
}

// Read loads a stage statement. ok is false when the file is absent.
func Read(dir, stage string) (Statement, bool, error) {
	raw, err := os.ReadFile(filepath.Join(dir, stage+".json"))
	if os.IsNotExist(err) {
		return Statement{}, false, nil
	}
	if err != nil {
		return Statement{}, false, err
	}
	var s Statement
	if err := json.Unmarshal(raw, &s); err != nil {
		return Statement{}, false, err
	}
	return s, true, nil
}

// Archive renames the stage files so the same stage can run again.
func Archive(dir, stage, suffix string) error {
	for _, ext := range []string{".json", ".md"} {
		from := filepath.Join(dir, stage+ext)
		if _, err := os.Stat(from); os.IsNotExist(err) {
			continue
		}
		to := filepath.Join(dir, stage+"-"+suffix+ext)
		if err := os.Rename(from, to); err != nil {
			return err
		}
	}
	return nil
}

func markdown(s *Statement) string {
	body := fmt.Sprintf("# %s\n\nVerdict: %s\nIteration: %d\nParent: %s\nReason: %s\n",
		s.Stage, s.Verdict, s.Iteration, s.ParentDigest, s.Reason)
	if s.GitSHA != "" {
		body += "Git: " + s.GitSHA + "\n"
	}
	if s.ChangeURL != "" {
		body += "Change: " + s.ChangeURL + "\n"
	}
	if s.Environment != "" {
		body += "Environment: " + s.Environment + "\n"
	}
	if len(s.Commands) > 0 {
		body += "\n## Commands\n\n"
		for _, c := range s.Commands {
			line := fmt.Sprintf("- `%s` exit %d", c.Command, c.ExitCode)
			if c.Error != "" {
				line += " (" + c.Error + ")"
			}
			body += line + "\n"
		}
	}
	body += "\nDigest: " + s.Digest + "\n"
	return body
}

func writeAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	f, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer func() { _ = os.Remove(tmp) }()
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
