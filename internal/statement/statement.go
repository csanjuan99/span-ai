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
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"span/internal/fsutil"
)

const (
	Advance = "advance"
	Hold    = "hold"
	Stop    = "stop"
)

// TailBytes is how much of a command's output travels with a statement. The
// failure is at the end, so the tail is what a reader needs and what the next
// build prompt receives. The whole output stays in the log file.
const TailBytes = 4096

// TestCounts is what a check's own output said about the tests it ran.
type TestCounts struct {
	Passed  int `json:"passed"`
	Failed  int `json:"failed,omitempty"`
	Skipped int `json:"skipped,omitempty"`
}

// Command is one declared check, not a shell script.
type Command struct {
	Command  string `json:"command"`
	ExitCode int    `json:"exitCode"`
	Error    string `json:"error,omitempty"`
	Band     string `json:"band,omitempty"`
	Output   string `json:"output,omitempty"`
	LogPath  string `json:"logPath,omitempty"`
	// Lines is how many lines the command printed in all, so a reader of the tail knows how
	// much of the log it is looking at ("5 of 142 lines").
	Lines int `json:"lines,omitempty"`
	// DurationMS is how long the command ran, so the QA tab can show a time per gate.
	DurationMS  int64  `json:"durationMs,omitempty"`
	Fingerprint string `json:"fingerprint,omitempty"`
	// Preexisting marks a failure the base commit already had. It is shown, not counted.
	Preexisting bool `json:"preexisting,omitempty"`
	// Tests is set only when the output printed a summary Span can read. A check that prints
	// none (a build, a linter) leaves it nil: no count is better than a guessed one.
	Tests *TestCounts `json:"tests,omitempty"`
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
	Fingerprint        string    `json:"fingerprint,omitempty"`
	Inconclusive       bool      `json:"inconclusive,omitempty"`
	Cap                int       `json:"cap,omitempty"`
	Weight             string    `json:"weight,omitempty"`
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
	if err := fsutil.WriteAtomic(filepath.Join(dir, s.Stage+".json"), raw); err != nil {
		return err
	}
	return fsutil.WriteAtomic(filepath.Join(dir, s.Stage+".md"), []byte(markdown(s)))
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
			if c.Band != "" {
				line += " (" + c.Band + ")"
			}
			if c.Error != "" {
				line += " " + c.Error
			}
			if c.Preexisting {
				line += " already failing on the base commit, not counted"
			}
			body += line + "\n"
			if c.LogPath != "" {
				body += "  Log: " + c.LogPath + "\n"
			}
			if c.Output != "" {
				body += "\n```\n" + strings.TrimRight(c.Output, "\n") + "\n```\n"
			}
		}
	}
	if s.Inconclusive {
		body += "\nInconclusive: the same command passed and failed on this commit.\n"
	}
	body += "\nDigest: " + s.Digest + "\n"
	return body
}

// Tail is the last TailBytes of output, marked when the start was dropped.
// A caller stores this on the statement; the whole output goes to the log file.
func Tail(output string) string {
	if len(output) <= TailBytes {
		return output
	}
	cut := len(output) - TailBytes
	for cut < len(output) && !utf8.RuneStart(output[cut]) {
		cut++
	}
	return "…" + output[cut:]
}

// Signature is the stable identity of a failure. Two runs that failed the same
// way share it, whatever order the parts arrived in. It exists so a repair loop
// can tell progress from a repeat without a model.
func Signature(parts ...string) string {
	seen := make(map[string]bool, len(parts))
	kept := make([]string, 0, len(parts))
	for _, part := range parts {
		clean := strings.Join(strings.Fields(strings.ToLower(part)), " ")
		if clean == "" || seen[clean] {
			continue
		}
		seen[clean] = true
		kept = append(kept, clean)
	}
	sort.Strings(kept)
	sum := sha256.Sum256([]byte(strings.Join(kept, "\n")))
	return hex.EncodeToString(sum[:])
}

// CountLines is how many lines output holds; a last line without a newline still counts.
func CountLines(output string) int {
	if output == "" {
		return 0
	}
	lines := strings.Count(output, "\n")
	if !strings.HasSuffix(output, "\n") {
		lines++
	}
	return lines
}
