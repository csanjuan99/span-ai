package policy

import (
	"fmt"
	"path/filepath"
	"strings"
)

// The three weights a story can earn. The weight decides how much machinery the run
// pays for: a one-line fix and a module with twenty criteria and a design handoff should
// not travel the same path.
const (
	Light    = "light"
	Standard = "standard"
	Deep     = "deep"
)

// Weight is the declared weight, or the derived one when the file says nothing.
func (f File) Weight(criteria int, handoffs []string) string {
	if declared := strings.ToLower(strings.TrimSpace(f.Run.Weight)); declared != "" {
		return declared
	}
	return Derive(criteria, handoffs)
}

// Derive reads the story, not the mood: a design handoff or many criteria means the
// change will be read closely, so the run must earn more checks.
func Derive(criteria int, handoffs []string) string {
	for _, name := range handoffs {
		switch strings.ToLower(filepath.Ext(name)) {
		case ".html", ".htm", ".png", ".jpg", ".jpeg", ".gif", ".webp", ".pdf", ".fig", ".svg":
			return Deep
		}
	}
	switch {
	case criteria >= 16:
		return Deep
	case criteria >= 1 && criteria <= 4:
		return Light
	case criteria == 0:
		return Light
	default:
		return Standard
	}
}

// RepairCapFor is the ceiling for that weight. A fix gets two repairs; a deep story,
// where each attempt is expensive and each criterion is a real thing to verify, gets six.
func (f File) RepairCapFor(weight string) int {
	if f.Repair.MaxAttempts > 0 {
		return f.Repair.MaxAttempts
	}
	switch weight {
	case Light:
		return 2
	case Deep:
		return 6
	default:
		return DefaultRepairAttempts
	}
}

// WideOn reports whether the whole suite is worth running for this weight. A fix does
// not need it: its focused band is the answer.
func WideOn(weight string) bool { return weight != Light }

// ValidWeight reports whether a declared weight is one Span knows.
func ValidWeight(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", Light, Standard, Deep:
		return true
	}
	return false
}

// WeightError is the message for a weight Span does not know.
func WeightError(value string) error {
	return fmt.Errorf("run.weight %q is not one of light, standard, or deep", value)
}
