package serve

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"time"

	"span/internal/shape"
)

// boardPoll is how often a followed activity feed looks for new steps.
const boardPoll = time.Second

// boardRoutes are the routes of the sessions board: the sessions grouped by status, what
// the agents did lately, and the actions on several sessions at once. `span shape board`,
// `activity` and `bulk` call the same shape functions.
func boardRoutes(mux *http.ServeMux, c *shapeConfig) {
	mux.HandleFunc("GET /shapes/activity", c.activity)
	mux.HandleFunc("POST /shapes/bulk", c.bulk)
}

// queryLimit reads the optional non-negative limit of a query: 0 when it is absent. A bad
// value answers 400 naming what is being counted.
func queryLimit(w http.ResponseWriter, raw, unit string) (int, bool) {
	if raw == "" {
		return 0, true
	}
	limit, err := strconv.Atoi(raw)
	if err != nil || limit < 0 {
		writeErr(w, &requestError{"invalid", "limit must be a number of " + unit})
		return 0, false
	}
	return limit, true
}

// board answers GET /shapes?group=status: the sessions in a column per status, with the
// same filters as the list plus limit, the most cards a column shows.
func (c *shapeConfig) board(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	ws, ok := queryWorkspace(w, r)
	if !ok {
		return
	}
	limit, ok := queryLimit(w, query.Get("limit"), "cards")
	if !ok {
		return
	}
	view, err := shape.Board(ws, shape.Filter{
		Query: query.Get("q"), Repo: query.Get("repo"), Agent: query.Get("agent"), Archived: query.Get("archived"),
	}, limit)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

// activity answers what happened lately in the sessions that are still open. With
// follow=1 it streams the steps that are new as server-sent events until the client leaves.
func (c *shapeConfig) activity(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	ws, ok := queryWorkspace(w, r)
	if !ok {
		return
	}
	var since time.Time
	if raw := query.Get("since"); raw != "" {
		var err error
		if since, err = time.Parse(time.RFC3339Nano, raw); err != nil {
			writeErr(w, &requestError{"invalid", "since must be a time like 2026-11-25T09:30:00Z"})
			return
		}
	}
	limit, ok := queryLimit(w, query.Get("limit"), "steps")
	if !ok {
		return
	}
	who := query.Get("who")
	feed, err := shape.RecentActivity(ws, who, since, limit)
	if err != nil {
		writeErr(w, err)
		return
	}
	if query.Get("follow") != "1" {
		writeJSON(w, http.StatusOK, feed)
		return
	}
	c.follow(w, r, ws, who, feed)
}

// follow streams the first batch and then every step newer than the last one sent.
func (c *shapeConfig) follow(w http.ResponseWriter, r *http.Request, ws, who string, first []shape.Activity) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeErr(w, &requestError{"invalid", "this connection cannot stream"})
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store")
	send := func(batch []shape.Activity) error {
		// Oldest first, so a reader sees the steps in the order they happened.
		sort.SliceStable(batch, func(i, j int) bool { return batch[i].At.Before(batch[j].At) })
		for _, a := range batch {
			raw, err := json.Marshal(a)
			if err != nil {
				return err
			}
			if _, err := fmt.Fprintf(w, "event: step\ndata: %s\n\n", raw); err != nil {
				return err
			}
		}
		flusher.Flush()
		return nil
	}
	if err := streamActivity(r.Context(), send, flusher, w, ws, who, first); err != nil {
		// A client that left is not a failure; anything else is the reason the stream stopped.
		if !errors.Is(err, context.Canceled) {
			log.Printf("activity stream %s: %v", ws, err)
		}
	}
}

// streamActivity sends the first batch and then, on every poll, the steps newer than the
// last one sent, or a comment line that keeps the connection open. It returns why it
// stopped: the client leaving, or the write or read that failed.
func streamActivity(ctx context.Context, send func([]shape.Activity) error, flusher http.Flusher, w http.ResponseWriter, ws, who string, first []shape.Activity) error {
	if err := send(first); err != nil {
		return err
	}
	last := time.Time{}
	for _, a := range first {
		if a.At.After(last) {
			last = a.At
		}
	}
	tick := time.NewTicker(boardPoll)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tick.C:
			batch, err := shape.RecentActivity(ws, who, last, 0)
			if err != nil {
				return err
			}
			if len(batch) == 0 {
				if _, err := fmt.Fprint(w, ": waiting\n\n"); err != nil {
					return err
				}
				flusher.Flush()
				continue
			}
			if err := send(batch); err != nil {
				return err
			}
			for _, a := range batch {
				if a.At.After(last) {
					last = a.At
				}
			}
		}
	}
}

// bulkBody asks for one action on several sessions.
type bulkBody struct {
	Workspace string   `json:"workspace"`
	Action    string   `json:"action"`
	IDs       []string `json:"ids"`
	// DryRun says what the action would do and changes nothing.
	DryRun bool `json:"dryRun"`
}

// bulk runs one action on several sessions and answers what it did to each. The sessions
// are locked together, in a fixed order, so no other request changes one mid-way.
func (c *shapeConfig) bulk(w http.ResponseWriter, r *http.Request) {
	body, ws, ok := decodeInWorkspace(w, r, func(b bulkBody) string { return b.Workspace })
	if !ok {
		return
	}
	ids := append([]string{}, body.IDs...)
	sort.Strings(ids)
	last := ""
	for _, id := range ids {
		if id == last {
			continue
		}
		last = id
		defer c.locks.lock(sessionKey(ws, id))()
	}
	res, err := shape.Bulk(r.Context(), ws, body.Action, body.IDs, body.DryRun, func(s *shape.Session) *shape.Engine {
		return c.engine(ws, s)
	})
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}
