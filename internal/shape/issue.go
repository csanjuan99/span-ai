package shape

import (
	"context"
	"errors"

	"span/internal/jira"
)

// Backlog is the part of Jira that shaping uses: look at the backlog of a Kanban board and
// create a story in it.
type Backlog interface {
	Preview(ctx context.Context, project string, board int) (jira.Preview, error)
	CreateInBacklog(ctx context.Context, draft jira.Draft) (jira.Created, error)
}

// IssuePreview is what creating the story in Jira would touch, with nothing created.
type IssuePreview struct {
	Project         string     `json:"project"`
	Board           jira.Board `json:"board"`
	BacklogStatuses []string   `json:"backlogStatuses"`
	Summary         string     `json:"summary"`
	Description     string     `json:"description"`
}

func sealedSession(workspace, id string) (*Session, error) {
	s, err := Load(workspace, id)
	if err != nil {
		return nil, err
	}
	if s.Status != StatusSealed {
		return nil, fail("shape_not_sealed", "session %s is not sealed", id)
	}
	return s, nil
}

// PreviewIssue looks at the board and the statuses of its backlog for a sealed story. The
// project key is normalized here; a board of 0 means the project's only Kanban board.
func PreviewIssue(ctx context.Context, backlog Backlog, workspace, id, project string, board int) (IssuePreview, error) {
	if board < 0 {
		return IssuePreview{}, fail("invalid", "board must be a board id")
	}
	key, err := jira.ProjectKey(project)
	if err != nil {
		return IssuePreview{}, err
	}
	s, err := sealedSession(workspace, id)
	if err != nil {
		return IssuePreview{}, err
	}
	view, err := backlog.Preview(ctx, key, board)
	if err != nil {
		return IssuePreview{}, err
	}
	return IssuePreview{Project: key, Board: view.Board, BacklogStatuses: view.BacklogStatuses, Summary: StorySummary(s.Story), Description: s.Story}, nil
}

// CreateIssue creates the story of a sealed session in the backlog, once, and only when the
// caller confirms. The key is stored whenever Jira reports one, even when the issue did not
// reach the backlog. A session that already has a key is returned as it is.
func (e *Engine) CreateIssue(ctx context.Context, backlog func() (Backlog, error), workspace, id, project string, board int, confirm bool) (*Session, error) {
	s, err := sealedSession(workspace, id)
	if err != nil {
		return nil, err
	}
	if s.Jira != nil && s.Jira.Key != "" {
		return s, nil
	}
	if !confirm {
		return nil, fail("confirm_required", "creating the story in Jira needs confirm: true")
	}
	key, err := jira.ProjectKey(project)
	if err != nil {
		return nil, err
	}
	b, err := backlog()
	if err != nil {
		return nil, err
	}
	created, createErr := b.CreateInBacklog(ctx, jira.Draft{Project: key, Board: board, Summary: StorySummary(s.Story), Description: s.Story})
	if created.Key == "" {
		if createErr == nil {
			createErr = fail("invalid", "jira returned no issue key")
		}
		return nil, createErr
	}
	saved, saveErr := e.SetJira(workspace, id, JiraRef{Key: created.Key, URL: created.URL, Status: created.Status})
	if err := errors.Join(createErr, saveErr); err != nil {
		return nil, err
	}
	return saved, nil
}
