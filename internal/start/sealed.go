package start

import (
	"path/filepath"
	"strings"

	"span/internal/shape"
)

// SealedStory is one shaped session the desk can start from.
type SealedStory struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
	Repo   string `json:"repo,omitempty"`
}

// ListSealed returns sealed shape sessions for a workspace, newest first.
func ListSealed(workspace string) ([]SealedStory, error) {
	ws := strings.TrimSpace(workspace)
	if ws == "" {
		ws = "."
	}
	ws, err := filepath.Abs(ws)
	if err != nil {
		return nil, err
	}
	sessions, err := shape.List(ws)
	if err != nil {
		return nil, err
	}
	filtered, err := shape.Filter{Status: shape.StatusSealed}.Apply(sessions)
	if err != nil {
		return nil, err
	}
	out := make([]SealedStory, 0, len(filtered))
	for _, s := range filtered {
		out = append(out, SealedStory{
			ID: s.ID, Title: shape.StorySummary(s.Story),
			Status: s.Status, Repo: s.Repo,
		})
	}
	return out, nil
}
