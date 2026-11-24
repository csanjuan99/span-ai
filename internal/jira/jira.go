// Package jira reads stories from Jira Cloud with an API token.
// The token stays in that project's .span directory and is never committed.
package jira

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"
)

// Account is the Jira site and the credential that can read it.
// An empty email means a bearer token (Jira Data Center). Otherwise the
// email and API token are sent as HTTP basic auth, which is Jira Cloud.
type Account struct {
	BaseURL string `json:"baseUrl"`
	Email   string `json:"email"`
	Token   string `json:"token"`
}

// Public is the account without the token.
type Public struct {
	Configured bool   `json:"configured"`
	BaseURL    string `json:"baseUrl,omitempty"`
	Email      string `json:"email,omitempty"`
	Auth       string `json:"auth,omitempty"`
}

// Story is one Jira issue in the shape Span can execute.
type Story struct {
	Key        string `json:"key"`
	Summary    string `json:"summary"`
	Type       string `json:"type"`
	Status     string `json:"status"`
	URL        string `json:"url"`
	Brief      string `json:"brief"`
	Acceptance string `json:"acceptance,omitempty"`
}

// Error is a failure the caller can handle.
type Error struct {
	Code string
	Msg  string
}

// Error formats the code and message.
func (e *Error) Error() string {
	if e.Msg == "" {
		return e.Code
	}
	return e.Code + ": " + e.Msg
}

// SpanCode returns the stable code callers match on.
func (e *Error) SpanCode() string { return e.Code }

// Path is the account file for one project: <project>/.span/jira.json.
func Path(project string) (string, error) {
	abs, err := filepath.Abs(strings.TrimSpace(project))
	if err != nil {
		return "", err
	}
	info, err := os.Stat(abs)
	if err != nil || !info.IsDir() {
		return "", &Error{Code: "invalid", Msg: "project path is missing"}
	}
	return filepath.Join(abs, ".span", "jira.json"), nil
}

// Save writes the account for this project so only the owner can read it.
// A blank token keeps the token already stored for this project.
func Save(project string, next Account) error {
	next.BaseURL = strings.TrimRight(strings.TrimSpace(next.BaseURL), "/")
	next.Email = strings.TrimSpace(next.Email)
	next.Token = strings.TrimSpace(next.Token)
	if _, err := url.ParseRequestURI(next.BaseURL); err != nil || !strings.Contains(next.BaseURL, "://") {
		return &Error{Code: "invalid", Msg: "jira site url is missing"}
	}
	if next.Token == "" {
		current, err := Load(project)
		if err != nil || current.Token == "" {
			return &Error{Code: "invalid", Msg: "api token is missing"}
		}
		next.Token = current.Token
	}
	raw, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return err
	}
	path, err := Path(project)
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("*\n"), 0o644); err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o600)
}

// Load reads the account saved for this project.
func Load(project string) (Account, error) {
	path, err := Path(project)
	if err != nil {
		return Account{}, err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Account{}, &Error{Code: "not_configured", Msg: "jira account is not configured"}
		}
		return Account{}, err
	}
	var stored Account
	if err := json.Unmarshal(raw, &stored); err != nil {
		return Account{}, &Error{Code: "invalid", Msg: "jira config cannot be read"}
	}
	stored.BaseURL = strings.TrimRight(stored.BaseURL, "/")
	if stored.BaseURL == "" || stored.Token == "" {
		return Account{}, &Error{Code: "not_configured", Msg: "jira account is not configured"}
	}
	return stored, nil
}

// Logout removes the saved account for this project.
func Logout(project string) Public {
	if path, err := Path(project); err == nil {
		// Only a missing file counts as signed out. Any other failure leaves
		// the account on disk, and the reload below reports it as configured.
		if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			account, loadErr := Load(project)
			return View(account, loadErr == nil)
		}
	}
	account, err := Load(project)
	return View(account, err == nil)
}

// View returns the account fields a caller may display.
func View(account Account, configured bool) Public {
	auth := ""
	if configured {
		auth = "basic"
		if account.Email == "" {
			auth = "bearer"
		}
	}
	return Public{Configured: configured, BaseURL: account.BaseURL, Email: account.Email, Auth: auth}
}

// Client calls one Jira site.
type Client struct {
	Account Account
	HTTP    *http.Client
}

// Issue reads one story, including its description and acceptance criteria.
func (c Client) Issue(ctx context.Context, key string) (Story, error) {
	key = strings.ToUpper(strings.TrimSpace(key))
	if !keyPattern(key) {
		return Story{}, &Error{Code: "invalid", Msg: "issue key is missing"}
	}
	endpoint := c.Account.BaseURL + "/rest/api/3/issue/" + url.PathEscape(key) + "?fields=*navigable&expand=names"
	body, err := c.get(ctx, endpoint)
	if err != nil {
		return Story{}, err
	}
	var payload map[string]any
	if err := decodeJSON(body, &payload); err != nil {
		return Story{}, err
	}
	fields, _ := payload["fields"].(map[string]any)
	names, _ := payload["names"].(map[string]string)
	if names == nil {
		names = stringMap(payload["names"])
	}
	summary, _ := fields["summary"].(string)
	acceptance := acceptance(fields, names)
	description := fieldText(fields["description"])
	found, _ := payload["key"].(string)
	if found == "" {
		found = key
	}
	story := Story{
		Key:        found,
		Summary:    summary,
		Type:       nestedName(fields, "issuetype"),
		Status:     nestedName(fields, "status"),
		URL:        strings.TrimRight(c.Account.BaseURL, "/") + "/browse/" + found,
		Acceptance: acceptance,
	}
	story.Brief = brief(story, description)
	return story, nil
}

// Search lists up to 20 issues. An empty query lists the current user's recent issues.
func (c Client) Search(ctx context.Context, jql string) ([]Story, error) {
	jql = strings.TrimSpace(jql)
	if jql == "" {
		jql = "assignee = currentUser() ORDER BY updated DESC"
	}
	raw, err := json.Marshal(map[string]any{
		"jql":        jql,
		"maxResults": 20,
		"fields":     []string{"summary", "status", "issuetype"},
	})
	if err != nil {
		return nil, err
	}
	body, err := c.call(ctx, http.MethodPost, c.Account.BaseURL+"/rest/api/3/search/jql", raw)
	if err != nil {
		return nil, err
	}
	var payload struct {
		Issues []struct {
			Key    string         `json:"key"`
			Fields map[string]any `json:"fields"`
		} `json:"issues"`
	}
	if err := decodeJSON(body, &payload); err != nil {
		return nil, err
	}
	out := make([]Story, 0, len(payload.Issues))
	for _, issue := range payload.Issues {
		out = append(out, Story{
			Key:     issue.Key,
			Summary: textOf(issue.Fields["summary"]),
			Type:    nestedName(issue.Fields, "issuetype"),
			Status:  nestedName(issue.Fields, "status"),
			URL:     strings.TrimRight(c.Account.BaseURL, "/") + "/browse/" + issue.Key,
		})
	}
	return out, nil
}

func (c Client) get(ctx context.Context, endpoint string) ([]byte, error) {
	return c.call(ctx, http.MethodGet, endpoint, nil)
}

func (c Client) call(ctx context.Context, method, endpoint string, payload []byte) ([]byte, error) {
	if c.Account.BaseURL == "" || c.Account.Token == "" {
		return nil, &Error{Code: "not_configured", Msg: "jira account is not configured"}
	}
	var reader io.Reader
	if payload != nil {
		reader = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
	if err != nil {
		return nil, &Error{Code: "invalid", Msg: "jira url is invalid"}
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", c.Account.authorization())
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	client := c.HTTP
	if client == nil {
		client = &http.Client{
			Timeout: 20 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 3 {
					return fmt.Errorf("too many redirects")
				}
				if req.URL.Host != via[0].URL.Host {
					req.Header.Del("Authorization")
				}
				return nil
			},
		}
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, &Error{Code: "unreachable", Msg: "jira did not respond"}
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return nil, &Error{Code: "unreachable", Msg: "jira did not respond"}
	}
	switch res.StatusCode {
	case http.StatusOK, http.StatusCreated, http.StatusNoContent:
		return body, nil
	case http.StatusUnauthorized, http.StatusForbidden:
		return nil, &Error{Code: "unauthorized", Msg: "jira rejected the account"}
	case http.StatusNotFound:
		return nil, &Error{Code: "not_found", Msg: "jira issue was not found"}
	default:
		msg := strings.TrimSpace(string(body))
		if len(msg) > 180 {
			msg = msg[:180]
		}
		if msg == "" {
			msg = res.Status
		}
		return nil, &Error{Code: "invalid", Msg: c.scrub(msg)}
	}
}

// Comment posts one plain-text note. Each line becomes an ADF paragraph.
func (c Client) Comment(ctx context.Context, key, text string) error {
	key = strings.ToUpper(strings.TrimSpace(key))
	if !keyPattern(key) {
		return &Error{Code: "invalid", Msg: "issue key is missing"}
	}
	raw, err := json.Marshal(map[string]any{"body": adfDoc(text)})
	if err != nil {
		return err
	}
	endpoint := c.Account.BaseURL + "/rest/api/3/issue/" + url.PathEscape(key) + "/comment"
	_, err = c.call(ctx, http.MethodPost, endpoint, raw)
	return err
}

// Transition moves the issue to a status by name. The id comes from the transitions available now.
func (c Client) Transition(ctx context.Context, key, statusName string) error {
	key = strings.ToUpper(strings.TrimSpace(key))
	statusName = strings.TrimSpace(statusName)
	if !keyPattern(key) {
		return &Error{Code: "invalid", Msg: "issue key is missing"}
	}
	if statusName == "" {
		return &Error{Code: "invalid", Msg: "status name is missing"}
	}
	available, err := c.transitions(ctx, key)
	if err != nil {
		return err
	}
	var id string
	names := make([]string, 0, len(available))
	for _, item := range available {
		label := item.To.Name
		if label == "" {
			label = item.Name
		}
		if label != "" {
			names = append(names, label)
		}
		if strings.EqualFold(item.To.Name, statusName) {
			id = item.ID
			continue
		}
		if id == "" && strings.EqualFold(item.Name, statusName) {
			id = item.ID
		}
	}
	if id == "" {
		return &Error{Code: "invalid", Msg: c.scrub("status " + statusName + " is not available: " + strings.Join(names, ", "))}
	}
	return c.applyTransition(ctx, key, id)
}

// transition is one move Jira offers for an issue right now.
type transition struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	To   struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"to"`
}

// transitions lists the moves available for the issue in its current status.
func (c Client) transitions(ctx context.Context, key string) ([]transition, error) {
	endpoint := c.Account.BaseURL + "/rest/api/3/issue/" + url.PathEscape(key) + "/transitions"
	raw, err := c.get(ctx, endpoint)
	if err != nil {
		return nil, err
	}
	var payload struct {
		Transitions []transition `json:"transitions"`
	}
	if err := decodeJSON(raw, &payload); err != nil {
		return nil, err
	}
	return payload.Transitions, nil
}

// applyTransition performs the transition with this id on the issue.
func (c Client) applyTransition(ctx context.Context, key, id string) error {
	endpoint := c.Account.BaseURL + "/rest/api/3/issue/" + url.PathEscape(key) + "/transitions"
	body, err := json.Marshal(map[string]any{"transition": map[string]string{"id": id}})
	if err != nil {
		return err
	}
	_, err = c.call(ctx, http.MethodPost, endpoint, body)
	return err
}

func (c Client) scrub(msg string) string {
	if c.Account.Token != "" && strings.Contains(msg, c.Account.Token) {
		msg = strings.ReplaceAll(msg, c.Account.Token, "[redacted]")
	}
	return msg
}

func adfDoc(text string) map[string]any {
	lines := strings.Split(text, "\n")
	content := make([]any, 0, len(lines))
	for _, line := range lines {
		para := map[string]any{"type": "paragraph"}
		if strings.TrimSpace(line) != "" {
			para["content"] = []any{map[string]any{"type": "text", "text": line}}
		}
		content = append(content, para)
	}
	if len(content) == 0 {
		content = append(content, map[string]any{"type": "paragraph"})
	}
	return map[string]any{"type": "doc", "version": 1, "content": content}
}

func (a Account) authorization() string {
	if a.Email == "" {
		return "Bearer " + a.Token
	}
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(a.Email+":"+a.Token))
}

func brief(story Story, description string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s %s\n", story.Key, story.Summary)
	if story.Type != "" || story.Status != "" {
		fmt.Fprintf(&b, "Type: %s\nStatus: %s\n", story.Type, story.Status)
	}
	if story.URL != "" {
		fmt.Fprintf(&b, "URL: %s\n", story.URL)
	}
	if description != "" {
		fmt.Fprintf(&b, "\n%s\n", description)
	}
	if story.Acceptance != "" {
		fmt.Fprintf(&b, "\nAcceptance criteria:\n%s\n", story.Acceptance)
	}
	return strings.TrimSpace(b.String()) + "\n"
}

func acceptance(fields map[string]any, names map[string]string) string {
	for id, name := range names {
		if strings.Contains(strings.ToLower(name), "acceptance criteria") {
			return fieldText(fields[id])
		}
	}
	return ""
}

func fieldText(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case map[string]any:
		return strings.TrimSpace(adf(typed))
	default:
		return ""
	}
}

func adf(node map[string]any) string {
	if text, ok := node["text"].(string); ok {
		return text
	}
	content, _ := node["content"].([]any)
	var parts []string
	for _, item := range content {
		child, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if text := adf(child); text != "" {
			parts = append(parts, text)
		}
	}
	kind, _ := node["type"].(string)
	joiner := ""
	switch kind {
	case "paragraph", "heading", "listItem", "bulletList", "orderedList", "blockquote", "codeBlock":
		joiner = "\n"
	default:
		joiner = ""
	}
	return strings.TrimSpace(strings.Join(parts, joiner))
}

func nestedName(fields map[string]any, key string) string {
	if fields == nil {
		return ""
	}
	node, _ := fields[key].(map[string]any)
	return textOf(node["name"])
}

func textOf(value any) string {
	text, _ := value.(string)
	return text
}

func stringMap(value any) map[string]string {
	raw, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	out := make(map[string]string, len(raw))
	for key, item := range raw {
		if text, ok := item.(string); ok {
			out[key] = text
		}
	}
	return out
}

func keyPattern(key string) bool {
	if len(key) < 3 || !strings.Contains(key, "-") {
		return false
	}
	project, number, ok := strings.Cut(key, "-")
	if !ok || project == "" || number == "" {
		return false
	}
	for _, r := range project {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	if !unicode.IsLetter(rune(project[0])) {
		return false
	}
	for _, r := range number {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// decodeJSON parses a Jira response body, hiding the body from the error.
func decodeJSON(raw []byte, v any) error {
	if err := json.Unmarshal(raw, v); err != nil {
		return &Error{Code: "invalid", Msg: "jira response is not json"}
	}
	return nil
}
