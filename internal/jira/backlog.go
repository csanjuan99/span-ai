package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const maxBoardPages = 20

// Board is a Jira Kanban board.
type Board struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// Preview is what creating a story would touch: the board and the statuses
// that count as its backlog. The caller adds the summary and description.
type Preview struct {
	Board           Board    `json:"board"`
	BacklogStatuses []string `json:"backlogStatuses"`
}

// Draft is a story to create in the backlog of a Kanban board.
// A zero Board discovers the single Kanban board of the project.
type Draft struct {
	Project     string
	Board       int
	Summary     string
	Description string
}

// Created is the issue that exists after CreateInBacklog.
type Created struct {
	Key    string `json:"key"`
	URL    string `json:"url"`
	Status string `json:"status"`
}

// backlogStatus is one status of the board's Backlog column.
type backlogStatus struct {
	ID   string
	Name string
}

// Preview resolves the board and its backlog statuses without writing anything.
func (c Client) Preview(ctx context.Context, project string, board int) (Preview, error) {
	key, err := ProjectKey(project)
	if err != nil {
		return Preview{}, err
	}
	resolved, statuses, err := c.backlog(ctx, key, board)
	if err != nil {
		return Preview{}, err
	}
	names := make([]string, 0, len(statuses))
	for _, status := range statuses {
		names = append(names, status.Name)
	}
	return Preview{Board: resolved, BacklogStatuses: names}, nil
}

// CreateInBacklog creates an unassigned Story labeled "span" and leaves it in
// the backlog of the Kanban board. The board and backlog are resolved before
// anything is created, so a refusal creates nothing. If the issue lands in
// another status it is moved through a transition into a backlog status; when
// none exists the error is jira_not_in_backlog and names the created key.
// Storing the key so a session never creates twice is the caller's job.
func (c Client) CreateInBacklog(ctx context.Context, draft Draft) (Created, error) {
	project, err := ProjectKey(draft.Project)
	if err != nil {
		return Created{}, err
	}
	summary := strings.TrimSpace(draft.Summary)
	if summary == "" {
		return Created{}, &Error{Code: "invalid", Msg: "summary is missing"}
	}
	_, statuses, err := c.backlog(ctx, project, draft.Board)
	if err != nil {
		return Created{}, err
	}
	raw, err := json.Marshal(map[string]any{"fields": map[string]any{
		"project":     map[string]string{"key": project},
		"issuetype":   map[string]string{"name": "Story"},
		"summary":     summary,
		"description": adfDoc(draft.Description),
		"labels":      []string{"span"},
	}})
	if err != nil {
		return Created{}, err
	}
	body, err := c.call(ctx, http.MethodPost, c.Account.BaseURL+"/rest/api/3/issue", raw)
	if err != nil {
		return Created{}, err
	}
	var made struct {
		Key string `json:"key"`
	}
	if err := json.Unmarshal(body, &made); err != nil || made.Key == "" {
		return Created{}, &Error{Code: "invalid", Msg: "jira did not return the new issue key"}
	}
	created := Created{Key: made.Key, URL: strings.TrimRight(c.Account.BaseURL, "/") + "/browse/" + made.Key}

	current, err := c.currentStatus(ctx, made.Key)
	if err != nil {
		return created, c.notInBacklog(made.Key, "its status could not be read: "+err.Error())
	}
	if target, ok := findStatus(statuses, current.ID); ok {
		created.Status = target.Name
		return created, nil
	}
	moves, err := c.transitions(ctx, made.Key)
	if err != nil {
		return created, c.notInBacklog(made.Key, "its transitions could not be read: "+err.Error())
	}
	for _, move := range moves {
		target, ok := findStatus(statuses, move.To.ID)
		if !ok {
			continue
		}
		if err := c.applyTransition(ctx, made.Key, move.ID); err != nil {
			return created, c.notInBacklog(made.Key, "the move to "+target.Name+" failed: "+err.Error())
		}
		created.Status = target.Name
		return created, nil
	}
	return created, c.notInBacklog(made.Key, "it is in "+current.Name+" and no transition reaches the backlog")
}

// notInBacklog reports an issue that exists but is not in the backlog.
func (c Client) notInBacklog(key, reason string) error {
	return &Error{Code: "jira_not_in_backlog", Msg: c.scrub(key + " was created but is not in the backlog: " + reason)}
}

func findStatus(statuses []backlogStatus, id string) (backlogStatus, bool) {
	for _, status := range statuses {
		if id != "" && status.ID == id {
			return status, true
		}
	}
	return backlogStatus{}, false
}

// currentStatus reads the id and name of the issue status.
func (c Client) currentStatus(ctx context.Context, key string) (backlogStatus, error) {
	body, err := c.get(ctx, c.Account.BaseURL+"/rest/api/3/issue/"+url.PathEscape(key)+"?fields=status")
	if err != nil {
		return backlogStatus{}, err
	}
	var payload struct {
		Fields struct {
			Status backlogStatus `json:"status"`
		} `json:"fields"`
	}
	if err := decodeJSON(body, &payload); err != nil {
		return backlogStatus{}, err
	}
	return payload.Fields.Status, nil
}

// UnmarshalJSON reads a Jira status object, keeping the id and the name.
func (s *backlogStatus) UnmarshalJSON(raw []byte) error {
	var item struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(raw, &item); err != nil {
		return err
	}
	s.ID, s.Name = item.ID, item.Name
	return nil
}

// ProjectKey is the project key as Jira expects it: trimmed and upper case. A key that is
// empty or carries URL syntax is refused, so it is safe to put in a request path.
func ProjectKey(raw string) (string, error) {
	key := strings.ToUpper(strings.TrimSpace(raw))
	if key == "" || strings.ContainsAny(key, " /?&#") {
		return "", &Error{Code: "invalid", Msg: "project key is missing"}
	}
	return key, nil
}

// backlog resolves the board and the statuses of its Backlog column for a key that
// ProjectKey already accepted.
func (c Client) backlog(ctx context.Context, project string, boardID int) (Board, []backlogStatus, error) {
	board, err := c.board(ctx, project, boardID)
	if err != nil {
		return Board{}, nil, err
	}
	statuses, err := c.backlogStatuses(ctx, board)
	if err != nil {
		return Board{}, nil, err
	}
	return board, statuses, nil
}

// board returns the explicit board after checking it is a Kanban board, or the
// only Kanban board of the project.
func (c Client) board(ctx context.Context, project string, id int) (Board, error) {
	if id > 0 {
		body, err := c.get(ctx, c.Account.BaseURL+"/rest/agile/1.0/board/"+strconv.Itoa(id))
		if err != nil {
			return Board{}, boardError(err, id)
		}
		var found struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
			Type string `json:"type"`
		}
		if err := decodeJSON(body, &found); err != nil {
			return Board{}, err
		}
		if found.Type != "" && !strings.EqualFold(found.Type, "kanban") {
			return Board{}, &Error{Code: "jira_board_not_found", Msg: fmt.Sprintf("board %d is a %s board, not a kanban board", id, found.Type)}
		}
		return Board{ID: id, Name: found.Name}, nil
	}
	var boards []Board
	for page, start := 0, 0; page < maxBoardPages; page++ {
		endpoint := c.Account.BaseURL + "/rest/agile/1.0/board?type=kanban&projectKeyOrId=" + url.QueryEscape(project) + "&startAt=" + strconv.Itoa(start)
		body, err := c.get(ctx, endpoint)
		if err != nil {
			return Board{}, boardError(err, 0)
		}
		var list struct {
			Values []Board `json:"values"`
			IsLast bool    `json:"isLast"`
		}
		if err := decodeJSON(body, &list); err != nil {
			return Board{}, err
		}
		boards = append(boards, list.Values...)
		if list.IsLast || len(list.Values) == 0 {
			break
		}
		start += len(list.Values)
	}
	switch len(boards) {
	case 0:
		return Board{}, &Error{Code: "jira_board_not_found", Msg: "project " + project + " has no kanban board"}
	case 1:
		return boards[0], nil
	}
	lines := make([]string, 0, len(boards))
	for _, board := range boards {
		lines = append(lines, fmt.Sprintf("%d %s", board.ID, board.Name))
	}
	return Board{}, &Error{Code: "jira_board_ambiguous", Msg: c.scrub("project " + project + " has several kanban boards, choose one: " + strings.Join(lines, "; "))}
}

// boardError turns a missing board into the board error code.
func boardError(err error, id int) error {
	if coded, ok := err.(*Error); ok && coded.Code == "not_found" {
		if id > 0 {
			return &Error{Code: "jira_board_not_found", Msg: fmt.Sprintf("board %d was not found", id)}
		}
		return &Error{Code: "jira_board_not_found", Msg: "kanban boards were not found"}
	}
	return err
}

// backlogStatuses reads the board configuration. The column named "Backlog"
// (any case) holds the backlog statuses; an empty or missing column means the
// backlog is not enabled. The configuration lists status ids only, so each id
// is resolved to its name with GET /rest/api/3/status/{id}.
func (c Client) backlogStatuses(ctx context.Context, board Board) ([]backlogStatus, error) {
	body, err := c.get(ctx, c.Account.BaseURL+"/rest/agile/1.0/board/"+strconv.Itoa(board.ID)+"/configuration")
	if err != nil {
		return nil, boardError(err, board.ID)
	}
	var config struct {
		ColumnConfig struct {
			Columns []struct {
				Name     string `json:"name"`
				Statuses []struct {
					ID string `json:"id"`
				} `json:"statuses"`
			} `json:"columns"`
		} `json:"columnConfig"`
	}
	if err := decodeJSON(body, &config); err != nil {
		return nil, err
	}
	var ids []string
	for _, column := range config.ColumnConfig.Columns {
		if strings.EqualFold(strings.TrimSpace(column.Name), "backlog") {
			for _, status := range column.Statuses {
				if status.ID != "" {
					ids = append(ids, status.ID)
				}
			}
			break
		}
	}
	if len(ids) == 0 {
		return nil, &Error{Code: "jira_backlog_disabled", Msg: fmt.Sprintf("board %d (%s) has no backlog: its Backlog column has no statuses", board.ID, board.Name)}
	}
	statuses := make([]backlogStatus, 0, len(ids))
	for _, id := range ids {
		raw, err := c.get(ctx, c.Account.BaseURL+"/rest/api/3/status/"+url.PathEscape(id))
		if err != nil {
			return nil, err
		}
		var named backlogStatus
		if err := decodeJSON(raw, &named); err != nil {
			return nil, err
		}
		if named.ID == "" {
			named.ID = id
		}
		statuses = append(statuses, named)
	}
	return statuses, nil
}
