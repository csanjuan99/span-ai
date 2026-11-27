package serve

import (
	"log"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"span/internal/engine"
	"span/internal/journal"
)

// control matches the requests that change a run on a person's say-so. It names the run
// and the action, so one wrapper journals all of them and a new control route only needs its
// name added here.
var control = regexp.MustCompile(`^/spans/([^/]+)/(answer|decide|story|agent|stop|rewind|pause|resume|approve|relaunch|instructions(?:/[^/]+)?|conflict/(?:choose|hint|abort))$`)

// keptFields are the form fields whose value is short and fixed in meaning. Anything else a
// person typed (a note, an answer, an instruction) is journaled as its length only.
var keptFields = map[string]bool{"stage": true, "environment": true, "cli": true, "agent": true, "model": true, "group": true, "option": true}

// statusWriter remembers the status the handler answered with.
type statusWriter struct {
	http.ResponseWriter
	status int
}

// WriteHeader remembers the status before passing it on.
func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// Write counts an implicit 200 when no header was written first.
func (w *statusWriter) Write(raw []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(raw)
}

// journalled records the control actions that succeeded. A refused or failed request changed
// nothing, so it leaves no line. A journal that cannot be written is logged and never fails
// the action itself: the person already did it, and refusing would hide that.
func journalled(root string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		match := control.FindStringSubmatch(r.URL.Path)
		if r.Method != http.MethodPost || match == nil || !safe(match[1]) {
			next.ServeHTTP(w, r)
			return
		}
		wrapped := &statusWriter{ResponseWriter: w}
		next.ServeHTTP(wrapped, r)
		if wrapped.status < 200 || wrapped.status >= 300 {
			return
		}
		detail := map[string]string{}
		for field, values := range r.PostForm {
			if len(values) == 0 {
				continue
			}
			if keptFields[field] {
				detail[field] = values[0]
			} else {
				detail[field+"Chars"] = strconv.Itoa(len([]rune(values[0])))
			}
		}
		if stage, ok := strings.CutPrefix(match[2], "instructions/"); ok {
			detail["stage"] = stage
			match[2] = "instructions"
		}
		if err := journal.Append(filepath.Join(root, match[1]), journal.Person, strings.ReplaceAll(match[2], "/", "."), detail); err != nil {
			log.Printf("journal %s: %v", match[1], err)
		}
	})
}

// eventsView is the journal for the page, with whether its chain still holds.
type eventsView struct {
	Events []journal.Event `json:"events"`
	// Intact is false when a line was removed or edited. BrokenAt is the first seq that no
	// longer follows from the one before it.
	Intact   bool `json:"intact"`
	BrokenAt int  `json:"brokenAt,omitempty"`
}

// readEvents reads a run's journal. A run with none has an empty, intact one.
func readEvents(root, id string) (eventsView, error) {
	dir := filepath.Join(root, id)
	if _, err := engine.Load(root, id); err != nil {
		return eventsView{}, err
	}
	events, err := journal.Read(dir)
	if err != nil {
		return eventsView{}, err
	}
	bad, err := journal.Verify(dir)
	if err != nil {
		return eventsView{}, err
	}
	if events == nil {
		events = []journal.Event{}
	}
	return eventsView{Events: events, Intact: bad == 0, BrokenAt: bad}, nil
}
