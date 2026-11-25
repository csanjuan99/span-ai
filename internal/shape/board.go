package shape

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"
)

// Bulk actions: what the board does to several sessions at once.
const (
	// BulkAssume accepts the assumption of every open question that has one.
	BulkAssume = "assume"
	// BulkSeal seals sessions that are ready.
	BulkSeal = "seal"
	// BulkArchive hides sessions from the default list.
	BulkArchive = "archive"
	// BulkUnarchive brings archived sessions back.
	BulkUnarchive = "unarchive"
)

// maxBulk caps how many sessions one bulk call touches.
const maxBulk = 50

// BoardCard is one session on the board.
type BoardCard struct {
	ID     string `json:"id"`
	Ref    string `json:"ref"`
	Title  string `json:"title"`
	Status string `json:"status"`
	Repo   string `json:"repo"`
	Agent  string `json:"agent"`
	Model  string `json:"model"`
	// OpenQuestions wait for the person; Assumed were closed with the assumption.
	OpenQuestions int `json:"openQuestions"`
	Assumed       int `json:"assumed"`
	// Stories is how many stories a proposed split holds.
	Stories   int        `json:"stories"`
	JiraKey   string     `json:"jiraKey"`
	SealedAt  *time.Time `json:"sealedAt,omitempty"`
	UpdatedAt time.Time  `json:"updatedAt"`
	// Doing is the last thing that happened in the session, with who did it. It is read
	// from the history, so it is what the session did last and not a live view of the agent.
	Doing    string `json:"doing"`
	Archived bool   `json:"archived"`
}

// BoardColumn is the sessions of one status.
type BoardColumn struct {
	Key   string      `json:"key"`
	Name  string      `json:"name"`
	Count int         `json:"count"`
	Cards []BoardCard `json:"cards"`
	// Hidden is how many cards the limit left out.
	Hidden int `json:"hidden"`
}

// BoardView is the sessions of a workspace grouped by status.
type BoardView struct {
	// InFlight counts the sessions that are not sealed.
	InFlight int `json:"inFlight"`
	// NeedYou counts the open questions across the board.
	NeedYou int           `json:"needYou"`
	Columns []BoardColumn `json:"columns"`
	// Repos lists the repositories on the board, for its filter.
	Repos []string `json:"repos"`
}

var boardColumns = []struct{ key, name string }{
	{StatusInterviewing, "Interviewing"},
	{StatusReady, "Ready"},
	{StatusSplit, "Split"},
	{StatusSealed, "Sealed"},
}

func (s *Session) card() BoardCard {
	card := BoardCard{
		ID: s.ID, Ref: s.Ref(), Title: s.Title(), Status: s.Status, Repo: s.Repo, Agent: s.Agent, Model: s.Model,
		OpenQuestions: s.OpenQuestions(), Stories: len(s.Split), JiraKey: s.JiraKey(),
		SealedAt: s.SealedAt, UpdatedAt: s.Activity(), Archived: s.Archived,
	}
	for _, q := range s.Map.Questions {
		if q.Status == QuestionAssumed {
			card.Assumed++
		}
	}
	return card
}

// lastStep is the last thing that happened in a session, in a few words.
func lastStep(s *Session) string {
	events, err := deriveEvents(s)
	if err != nil || len(events) == 0 {
		return ""
	}
	last := events[len(events)-1]
	if last.Actor == personActor {
		return "you: " + strings.ToLower(last.Summary[:1]) + last.Summary[1:]
	}
	return last.Actor + ": " + strings.ToLower(last.Summary[:1]) + last.Summary[1:]
}

// Board groups the sessions of a workspace by status. A limit above zero shows at most that
// many cards per column and says how many it hid.
func Board(workspace string, f Filter, limit int) (BoardView, error) {
	all, err := List(workspace)
	if all == nil {
		return BoardView{}, err
	}
	if err != nil {
		// The unreadable sessions are named in the error; the board shows the rest.
		log.Printf("shape board: %v", err)
	}
	kept, ferr := f.Apply(all)
	if ferr != nil {
		return BoardView{}, ferr
	}
	if limit < 0 {
		return BoardView{}, fail("invalid", "limit must not be negative")
	}
	byStatus := map[string][]BoardCard{}
	repos := map[string]bool{}
	view := BoardView{Columns: []BoardColumn{}, Repos: []string{}}
	for _, s := range all {
		if !s.Archived {
			repos[s.Repo] = true
		}
	}
	for _, s := range kept {
		card := s.card()
		if s.Status != StatusSealed {
			view.InFlight++
			view.NeedYou += card.OpenQuestions
			card.Doing = lastStep(s)
		}
		byStatus[s.Status] = append(byStatus[s.Status], card)
	}
	for _, col := range boardColumns {
		cards := byStatus[col.key]
		sort.SliceStable(cards, func(i, j int) bool { return cards[i].UpdatedAt.After(cards[j].UpdatedAt) })
		column := BoardColumn{Key: col.key, Name: col.name, Count: len(cards), Cards: []BoardCard{}}
		shown := cards
		if limit > 0 && len(shown) > limit {
			shown = shown[:limit]
			column.Hidden = len(cards) - limit
		}
		column.Cards = append(column.Cards, shown...)
		view.Columns = append(view.Columns, column)
	}
	for repo := range repos {
		view.Repos = append(view.Repos, repo)
	}
	sort.Strings(view.Repos)
	return view, nil
}

// Activity is one thing that happened in a session, for the live feed.
type Activity struct {
	At      time.Time `json:"at"`
	Session string    `json:"session"`
	Ref     string    `json:"ref"`
	Actor   string    `json:"actor"`
	Kind    string    `json:"kind"`
	Text    string    `json:"text"`
}

// RecentActivity lists what happened across the sessions that are not sealed or archived,
// newest first. who is "agents" (the default), "you" or "everyone"; since drops older
// steps, so a client asks only for what is new.
func RecentActivity(workspace, who string, since time.Time, limit int) ([]Activity, error) {
	switch who {
	case "", "agents", "you", "everyone":
	default:
		return nil, fail("invalid", "who must be agents, you or everyone, not %q", who)
	}
	if limit <= 0 {
		limit = 30
	}
	all, err := List(workspace)
	if all == nil {
		return nil, err
	}
	if err != nil {
		log.Printf("shape activity: %v", err)
	}
	out := []Activity{}
	for _, s := range all {
		if s.Archived || s.Status == StatusSealed {
			continue
		}
		events, derr := deriveEvents(s)
		if derr != nil {
			// One session whose history cannot be read must not hide the others' activity.
			log.Printf("shape activity: session %s: %v", s.ID, derr)
			continue
		}
		for _, e := range events {
			if !since.IsZero() && !e.At.After(since) {
				continue
			}
			if (who == "" || who == "agents") && e.Actor == personActor {
				continue
			}
			if who == "you" && e.Actor != personActor {
				continue
			}
			out = append(out, Activity{At: e.At, Session: s.ID, Ref: s.Ref(), Actor: e.Actor, Kind: e.Kind, Text: e.Summary})
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].At.After(out[j].At) })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// BulkItem is what a bulk action did to one session.
type BulkItem struct {
	ID  string `json:"id"`
	Ref string `json:"ref"`
	// OK says the session was changed, or would be in a dry run.
	OK bool `json:"ok"`
	// Skipped says there was nothing to do: no open question to assume, already archived.
	Skipped bool `json:"skipped,omitempty"`
	// Changed lists what changed, one line each: the questions assumed, the assumptions sealed in.
	Changed []string `json:"changed,omitempty"`
	Code    string   `json:"code,omitempty"`
	Message string   `json:"message,omitempty"`
}

// BulkResult is what a bulk action did, per session. One session failing never stops the
// others.
type BulkResult struct {
	Action  string     `json:"action"`
	DryRun  bool       `json:"dryRun"`
	Items   []BulkItem `json:"items"`
	Done    int        `json:"done"`
	Skipped int        `json:"skipped"`
	Failed  int        `json:"failed"`
}

// EngineFor builds the engine a session runs its agent on.
type EngineFor func(*Session) *Engine

// Bulk applies one action to several sessions. With dryRun nothing changes and the result
// says what would. A session that cannot take the action reports why, and the others still
// do. Sealing never touches Jira or the repository.
func Bulk(ctx context.Context, workspace, action string, ids []string, dryRun bool, engineFor EngineFor) (BulkResult, error) {
	switch action {
	case BulkAssume, BulkSeal, BulkArchive, BulkUnarchive:
	default:
		return BulkResult{}, fail("invalid", "action must be assume, seal, archive or unarchive, not %q", action)
	}
	ids = uniqueIDs(ids)
	if len(ids) == 0 {
		return BulkResult{}, fail("invalid", "no sessions selected")
	}
	if len(ids) > maxBulk {
		return BulkResult{}, fail("invalid", "at most %d sessions at once, got %d", maxBulk, len(ids))
	}
	res := BulkResult{Action: action, DryRun: dryRun, Items: []BulkItem{}}
	for _, id := range ids {
		item := bulkOne(ctx, workspace, action, id, dryRun, engineFor)
		switch {
		case item.Code != "":
			res.Failed++
		case item.Skipped:
			res.Skipped++
		case item.OK:
			res.Done++
		}
		res.Items = append(res.Items, item)
	}
	return res, nil
}

func uniqueIDs(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, id := range in {
		id = strings.TrimSpace(id)
		if id != "" && !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

func failedItem(s *Session, id string, err error) BulkItem {
	item := BulkItem{ID: id, Code: "error", Message: err.Error()}
	if s != nil {
		item.Ref = s.Ref()
	}
	var coded *Error
	if errors.As(err, &coded) {
		item.Code = coded.Code
		if coded.Msg != "" {
			item.Message = coded.Msg
		}
	}
	return item
}

func bulkOne(ctx context.Context, workspace, action, id string, dryRun bool, engineFor EngineFor) BulkItem {
	s, err := Load(workspace, id)
	if err != nil {
		return failedItem(nil, id, err)
	}
	item := BulkItem{ID: id, Ref: s.Ref()}
	switch action {
	case BulkAssume:
		return bulkAssume(ctx, s, item, dryRun, engineFor)
	case BulkSeal:
		return bulkSeal(s, item, dryRun, engineFor)
	}
	want := action == BulkArchive
	if s.Archived == want {
		item.Skipped, item.OK = true, true
		item.Message = map[bool]string{true: "already archived", false: "is not archived"}[want]
		return item
	}
	if dryRun {
		item.OK, item.Changed = true, []string{action + " " + s.Ref()}
		return item
	}
	if _, err := SetArchived(workspace, id, want); err != nil {
		return failedItem(s, id, err)
	}
	item.OK, item.Changed = true, []string{action + "d " + s.Ref()}
	return item
}

func bulkAssume(ctx context.Context, s *Session, item BulkItem, dryRun bool, engineFor EngineFor) BulkItem {
	if s.Status == StatusSealed {
		return failedItem(s, s.ID, fail("shape_sealed", "session %s is sealed: a sealed story cannot change", s.ID))
	}
	var ids []string
	for _, q := range s.Map.Questions {
		if q.Status == QuestionOpen && strings.TrimSpace(q.Assume) != "" {
			ids = append(ids, q.ID)
			item.Changed = append(item.Changed, fmt.Sprintf("%s becomes assumed: %s", q.ID, q.Assume))
		}
	}
	if len(ids) == 0 {
		item.Skipped, item.OK, item.Changed = true, true, nil
		item.Message = "no open question has an assumption"
		return item
	}
	if dryRun {
		item.OK = true
		return item
	}
	if _, err := engineFor(s).Answer(ctx, s.Workspace, s.ID, nil, ids); err != nil {
		return failedItem(s, s.ID, err)
	}
	item.OK = true
	return item
}

func bulkSeal(s *Session, item BulkItem, dryRun bool, engineFor EngineFor) BulkItem {
	if s.Status != StatusReady {
		return failedItem(s, s.ID, fail("shape_not_ready", "status is %s, not ready", s.Status))
	}
	for _, q := range s.Map.Questions {
		if q.Status == QuestionAssumed {
			item.Changed = append(item.Changed, fmt.Sprintf("sealed with the assumption for %s: %s", q.ID, q.Answer))
		}
	}
	if dryRun {
		item.OK = true
		return item
	}
	if _, err := engineFor(s).Seal(s.Workspace, s.ID); err != nil {
		return failedItem(s, s.ID, err)
	}
	item.OK = true
	return item
}
