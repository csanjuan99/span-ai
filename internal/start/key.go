package start

import (
	"path/filepath"
	"strings"
	"time"

	"span/internal/shape"
)

// RunKey names the Jira key or span id the next run would use.
type RunKey struct {
	Key     string `json:"key"`
	Source  string `json:"source"`
	ShapeID string `json:"shapeId,omitempty"`
}

// NextRunKey returns the issue key for a sealed shape, or a preview span id when only text is given.
func NextRunKey(workspace, shapeID, spansRoot string) (RunKey, error) {
	ws := strings.TrimSpace(workspace)
	if ws == "" {
		ws = "."
	}
	ws, err := filepath.Abs(ws)
	if err != nil {
		return RunKey{}, err
	}
	id := strings.TrimSpace(shapeID)
	if id != "" {
		s, err := shape.Load(ws, id)
		if err != nil {
			return RunKey{}, err
		}
		if s.Jira != nil && s.Jira.Key != "" {
			return RunKey{Key: s.Jira.Key, Source: "jira", ShapeID: id}, nil
		}
		return RunKey{Key: s.ID, Source: "shape", ShapeID: id}, nil
	}
	return RunKey{Key: previewSpanID(time.Now().UTC()), Source: "preview"}, nil
}

func previewSpanID(now time.Time) string {
	return now.Format("2006-01-02-1504") + "-??"
}
