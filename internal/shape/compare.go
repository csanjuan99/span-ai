package shape

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// Limits of one comparison.
const (
	// CompareMin is the fewest sessions that can be compared.
	CompareMin = 2
	// CompareMax is the most sessions one comparison takes.
	CompareMax = 5
)

// The thresholds are estimates, not proofs: two statements are "the same" when most of
// their words are shared, and "in conflict" when they talk about the same thing but differ.
const (
	sameScore     = 0.8
	conflictScore = 0.45
	questionScore = 0.6
	// duplicatePercent is the share of items two sessions must have in common to be the same story.
	duplicatePercent = 60
	// duplicateMinItems keeps two one-line sessions from being called the same story.
	duplicateMinItems = 2
)

// Origins: how a statement came to be in a session.
const (
	originAssumed = "assumed by agent"
	originSealed  = "sealed, cannot change"
)

// SessionBrief is a session as the comparison names it.
type SessionBrief struct {
	ID     string `json:"id"`
	Ref    string `json:"ref"`
	Title  string `json:"title"`
	Status string `json:"status"`
	Sealed bool   `json:"sealed"`
	// Items counts the rules, examples and settled questions that were compared.
	Items int `json:"items"`
	// Unique counts those that exist in no other compared session and are not in conflict.
	Unique int `json:"unique"`
	// MergedInto is the session this one was merged into, when it was.
	MergedInto string `json:"mergedInto,omitempty"`
}

// Member is one statement of one session in a group.
type Member struct {
	Session string `json:"session"`
	Ref     string `json:"ref"`
	Item    string `json:"item"`
	Kind    string `json:"kind"`
	Text    string `json:"text"`
	// Origin says whether the person answered it, the agent assumed it, or the session is sealed.
	Origin string `json:"origin"`
	Sealed bool   `json:"sealed"`
}

// SameGroup is one statement found in two or more sessions.
type SameGroup struct {
	Kind    string   `json:"kind"`
	Text    string   `json:"text"`
	Members []Member `json:"members"`
}

// ConflictOption is one way the sessions answer the same thing, with who says it.
type ConflictOption struct {
	Text    string   `json:"text"`
	Members []Member `json:"members"`
}

// Conflict is something the sessions answer differently. Span never picks between two
// answers the person gave: the person decides which session's answer wins.
type Conflict struct {
	ID      string           `json:"id"`
	Title   string           `json:"title"`
	Options []ConflictOption `json:"options"`
}

// Duplicate is a pair of sessions that cover mostly the same items.
type Duplicate struct {
	A       string `json:"a"`
	B       string `json:"b"`
	ARef    string `json:"aRef"`
	BRef    string `json:"bRef"`
	Percent int    `json:"percent"`
	// AItems and BItems are the item ids that match, in the same order.
	AItems []string `json:"aItems"`
	BItems []string `json:"bItems"`
	// Dismissed is true after the person chose to keep the pair separate.
	Dismissed bool `json:"dismissed"`
}

// Comparison is what the compared sessions share and where they disagree.
type Comparison struct {
	Repo       string         `json:"repo"`
	Sessions   []SessionBrief `json:"sessions"`
	Same       []SameGroup    `json:"same"`
	Conflicts  []Conflict     `json:"conflicts"`
	Duplicates []Duplicate    `json:"duplicates"`
	Unique     int            `json:"unique"`
	// Estimate is always true: matching is by shared words, so the page says so.
	Estimate bool `json:"estimate"`
}

type cItem struct {
	sess   int
	kind   string
	id     string
	text   string
	value  string
	origin string
	toks   map[string]bool
	tokens int
}

type cState struct {
	sessions []*Session
	items    []cItem
	cmp      Comparison
	// conflicts keeps each conflict's members for Merge.
	conflicts [][]int
	// sameOf maps an item to the group number it belongs to, or -1.
	sameOf map[int]int
	groups [][]int
}

var stopwords = map[string]bool{
	"a": true, "an": true, "the": true, "of": true, "to": true, "and": true, "or": true, "is": true, "are": true,
	"in": true, "on": true, "for": true, "with": true, "can": true, "be": true, "it": true, "its": true, "by": true,
	"that": true, "this": true, "as": true, "at": true, "from": true, "their": true, "they": true, "his": true, "her": true,
}

func tokens(text string) map[string]bool {
	out := map[string]bool{}
	for _, w := range strings.FieldsFunc(strings.ToLower(text), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsNumber(r) }) {
		if stopwords[w] {
			continue
		}
		if len(w) > 3 && strings.HasSuffix(w, "s") {
			w = strings.TrimSuffix(w, "s")
		}
		out[w] = true
	}
	return out
}

func jaccard(a, b map[string]bool) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	shared := 0
	for w := range a {
		if b[w] {
			shared++
		}
	}
	return float64(shared) / float64(len(a)+len(b)-shared)
}

func normalized(text string) string {
	words := make([]string, 0)
	for w := range tokens(text) {
		words = append(words, w)
	}
	sort.Strings(words)
	return strings.Join(words, " ")
}

// resolveSessions loads the sessions by id or by their SES- reference, in id order, and
// refuses what a comparison cannot take.
func resolveSessions(workspace string, refs []string) ([]*Session, error) {
	if len(refs) < CompareMin || len(refs) > CompareMax {
		return nil, fail("invalid", "compare takes %d to %d sessions, got %d", CompareMin, CompareMax, len(refs))
	}
	all, err := List(workspace)
	if err != nil && all == nil {
		return nil, err
	}
	seen := map[string]bool{}
	var out []*Session
	for _, ref := range refs {
		var found *Session
		for _, s := range all {
			if s.ID == ref || strings.EqualFold(s.Ref(), ref) {
				found = s
				break
			}
		}
		if found == nil {
			return nil, notFound(ref)
		}
		if seen[found.ID] {
			return nil, fail("invalid", "session %s is listed twice", ref)
		}
		seen[found.ID] = true
		out = append(out, found)
	}
	repo := out[0].Repo
	for _, s := range out[1:] {
		if s.Repo != repo {
			return nil, fail("invalid", "sessions must share one repo: %s is in %q and %s is in %q", out[0].Ref(), repo, s.Ref(), s.Repo)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func originOf(s *Session, source Source) string {
	if s.Status == StatusSealed {
		return originSealed
	}
	if source.Kind != SourceAnswer {
		switch source.Kind {
		case SourceCode:
			return "from the code"
		case SourceContext:
			return "from the product context"
		}
		return "from the idea"
	}
	for _, q := range s.Map.Questions {
		if q.AnswerID == source.Ref && q.Status == QuestionAssumed {
			return originAssumed
		}
	}
	return "answered by you (" + source.Ref + ")"
}

func collect(sessions []*Session) []cItem {
	var out []cItem
	add := func(i int, kind, id, text, value, origin string) {
		toks := tokens(text)
		out = append(out, cItem{sess: i, kind: kind, id: id, text: text, value: value, origin: origin, toks: toks, tokens: len(toks)})
	}
	for i, s := range sessions {
		for _, r := range s.Map.Rules {
			add(i, ItemRule, r.ID, r.Text, r.Text, originOf(s, r.Source))
		}
		for _, e := range s.Map.Examples {
			add(i, ItemExample, e.ID, e.Text, e.Text, originOf(s, e.Source))
		}
		for _, q := range s.Map.Questions {
			switch q.Status {
			case QuestionAnswered:
				origin := "answered by you"
				if q.AnswerID != "" {
					origin += " (" + q.AnswerID + ")"
				}
				if s.Status == StatusSealed {
					origin = originSealed
				}
				add(i, ItemQuestion, q.ID, q.Text, q.Answer, origin)
			case QuestionAssumed:
				origin := originAssumed
				if s.Status == StatusSealed {
					origin = originSealed
				}
				add(i, ItemQuestion, q.ID, q.Text, q.Answer, origin)
			}
		}
	}
	return out
}

type dsu []int

func newDSU(n int) dsu {
	d := make(dsu, n)
	for i := range d {
		d[i] = i
	}
	return d
}

func (d dsu) find(i int) int {
	for d[i] != i {
		d[i] = d[d[i]]
		i = d[i]
	}
	return i
}

func (d dsu) union(a, b int) { d[d.find(a)] = d.find(b) }

// clusters returns the groups of two or more items joined by edge, in item order.
func clusters(n int, edge func(i, j int) bool) [][]int {
	d := newDSU(n)
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			if edge(i, j) {
				d.union(i, j)
			}
		}
	}
	byRoot := map[int][]int{}
	var order []int
	for i := 0; i < n; i++ {
		root := d.find(i)
		if _, ok := byRoot[root]; !ok {
			order = append(order, root)
		}
		byRoot[root] = append(byRoot[root], i)
	}
	var out [][]int
	for _, root := range order {
		if len(byRoot[root]) > 1 {
			out = append(out, byRoot[root])
		}
	}
	return out
}

func (st *cState) member(i int) Member {
	it, s := st.items[i], st.sessions[st.items[i].sess]
	text := it.text
	if it.kind == ItemQuestion {
		text = it.value
	}
	return Member{Session: s.ID, Ref: s.Ref(), Item: it.id, Kind: it.kind, Text: text, Origin: it.origin, Sealed: s.Status == StatusSealed}
}

// comparable says whether two kinds of item can be measured against each other: a rule and an
// example can state the same fact, a question only matches a question.
func comparable(a, b string) bool {
	if a == ItemQuestion || b == ItemQuestion {
		return a == b
	}
	return true
}

func differentSessions(st *cState, ids []int) bool {
	first := st.items[ids[0]].sess
	for _, i := range ids[1:] {
		if st.items[i].sess != first {
			return true
		}
	}
	return false
}

// Compare finds what two to five sessions of one repo share and where they disagree.
// refs are session ids or SES- references. Nothing is written.
func Compare(workspace string, refs []string) (Comparison, error) {
	st, err := compareState(workspace, refs)
	if err != nil {
		return Comparison{}, err
	}
	return st.cmp, nil
}

func compareState(workspace string, refs []string) (*cState, error) {
	sessions, err := resolveSessions(workspace, refs)
	if err != nil {
		return nil, err
	}
	st := &cState{sessions: sessions, items: collect(sessions), sameOf: map[int]int{}}
	n := len(st.items)
	score := func(i, j int) (float64, bool) {
		a, b := st.items[i], st.items[j]
		if a.sess == b.sess || !comparable(a.kind, b.kind) {
			return 0, false
		}
		return jaccard(a.toks, b.toks), true
	}
	st.groups = clusters(n, func(i, j int) bool {
		sc, ok := score(i, j)
		return ok && st.items[i].kind == st.items[j].kind && st.items[i].kind != ItemQuestion && sc >= sameScore
	})
	inConflict := map[int]bool{}
	conflictEdge := func(i, j int) bool {
		sc, ok := score(i, j)
		if !ok {
			return false
		}
		a, b := st.items[i], st.items[j]
		if a.kind == ItemQuestion {
			return sc >= questionScore && normalized(a.value) != normalized(b.value)
		}
		return sc >= conflictScore && sc < sameScore
	}
	for _, ids := range clusters(n, conflictEdge) {
		if !differentSessions(st, ids) {
			continue
		}
		options := map[string][]int{}
		var keys []string
		for _, i := range ids {
			key := normalized(st.items[i].value)
			if _, ok := options[key]; !ok {
				keys = append(keys, key)
			}
			options[key] = append(options[key], i)
		}
		if len(keys) < 2 {
			continue
		}
		st.conflicts = append(st.conflicts, ids)
		c := Conflict{ID: "C" + strconv.Itoa(len(st.conflicts)), Title: conflictTitle(st, ids)}
		for _, key := range keys {
			opt := ConflictOption{Text: st.items[options[key][0]].value}
			for _, i := range options[key] {
				opt.Members = append(opt.Members, st.member(i))
				inConflict[i] = true
			}
			c.Options = append(c.Options, opt)
		}
		st.cmp.Conflicts = append(st.cmp.Conflicts, c)
	}
	st.fillSame()
	if err := st.fillDuplicates(workspace); err != nil {
		return nil, err
	}
	st.fillBriefs(inConflict)
	st.cmp.Repo = sessions[0].Repo
	st.cmp.Estimate = true
	if st.cmp.Same == nil {
		st.cmp.Same = []SameGroup{}
	}
	if st.cmp.Conflicts == nil {
		st.cmp.Conflicts = []Conflict{}
	}
	if st.cmp.Duplicates == nil {
		st.cmp.Duplicates = []Duplicate{}
	}
	return st, nil
}

// conflictTitle names what the conflict is about: the question when one is involved, else
// the shortest statement.
func conflictTitle(st *cState, ids []int) string {
	best := ""
	for _, i := range ids {
		it := st.items[i]
		text := it.text
		if it.kind == ItemQuestion {
			return text
		}
		if best == "" || len(text) < len(best) {
			best = text
		}
	}
	return best
}

func (st *cState) fillSame() {
	for gi, ids := range st.groups {
		if !differentSessions(st, ids) {
			continue
		}
		g := SameGroup{Kind: st.items[ids[0]].kind, Text: st.items[ids[0]].text}
		for _, i := range ids {
			g.Members = append(g.Members, st.member(i))
			st.sameOf[i] = gi
		}
		st.cmp.Same = append(st.cmp.Same, g)
	}
}

// fillDuplicates pairs sessions whose items mostly match one to one.
func (st *cState) fillDuplicates(workspace string) error {
	dismissed, err := dismissedPairs(workspace)
	if err != nil {
		return err
	}
	for a := 0; a < len(st.sessions); a++ {
		for b := a + 1; b < len(st.sessions); b++ {
			var aItems, bItems []string
			used := map[int]bool{}
			countA, countB := 0, 0
			for _, x := range st.items {
				if x.kind != ItemQuestion && x.sess == a {
					countA++
				}
				if x.kind != ItemQuestion && x.sess == b {
					countB++
				}
				if x.sess != a || x.kind == ItemQuestion {
					continue
				}
				for j, y := range st.items {
					if y.sess != b || y.kind != x.kind || used[j] || jaccard(x.toks, y.toks) < sameScore {
						continue
					}
					used[j] = true
					aItems, bItems = append(aItems, x.id), append(bItems, y.id)
					break
				}
			}
			smaller := min(countA, countB)
			if smaller == 0 || len(aItems) < duplicateMinItems {
				continue
			}
			percent := len(aItems) * 100 / smaller
			if percent < duplicatePercent {
				continue
			}
			sa, sb := st.sessions[a], st.sessions[b]
			st.cmp.Duplicates = append(st.cmp.Duplicates, Duplicate{
				A: sa.ID, B: sb.ID, ARef: sa.Ref(), BRef: sb.Ref(), Percent: percent, AItems: aItems, BItems: bItems,
				Dismissed: dismissed[pairKey(sa.ID, sb.ID)],
			})
		}
	}
	return nil
}

func (st *cState) fillBriefs(inConflict map[int]bool) {
	for i, s := range st.sessions {
		brief := SessionBrief{ID: s.ID, Ref: s.Ref(), Title: s.Title(), Status: s.Status, Sealed: s.Status == StatusSealed, MergedInto: s.MergedInto}
		for k, it := range st.items {
			if it.sess != i {
				continue
			}
			brief.Items++
			if _, same := st.sameOf[k]; !same && !inConflict[k] {
				brief.Unique++
			}
		}
		st.cmp.Unique += brief.Unique
		st.cmp.Sessions = append(st.cmp.Sessions, brief)
	}
}

func pairKey(a, b string) string {
	if a > b {
		a, b = b, a
	}
	return a + "|" + b
}

const dismissedFile = "compare-dismissed.json"

func dismissedPath(workspace string) string { return filepath.Join(Dir(workspace), dismissedFile) }

// dismissedPairs reads the pairs the person chose to keep separate. A workspace that never
// dismissed a pair has no file, which is not an error; a file that cannot be read or parsed
// is, because silently dropping it would offer every dismissed merge again.
func dismissedPairs(workspace string) (map[string]bool, error) {
	out := map[string]bool{}
	raw, err := os.ReadFile(dismissedPath(workspace))
	if errors.Is(err, fs.ErrNotExist) {
		return out, nil
	}
	if err != nil {
		return nil, err
	}
	var file struct {
		Pairs []string `json:"pairs"`
	}
	if err := json.Unmarshal(raw, &file); err != nil {
		return nil, fmt.Errorf("%s: %w", dismissedPath(workspace), err)
	}
	for _, p := range file.Pairs {
		out[p] = true
	}
	return out, nil
}

func writeDismissed(workspace string, pairs map[string]bool) error {
	keys := make([]string, 0, len(pairs))
	for k := range pairs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	raw, err := json.MarshalIndent(map[string]any{"pairs": keys, "updatedAt": time.Now().UTC()}, "", "  ")
	if err != nil {
		return fmt.Errorf("encode dismissed pairs: %w", err)
	}
	if err := os.MkdirAll(Dir(workspace), 0o755); err != nil {
		return fmt.Errorf("write dismissed pairs: %w", err)
	}
	return writeAtomic(dismissedPath(workspace), append(raw, '\n'))
}

// Dismiss records that the person chose to keep these sessions separate: every pair among
// them is marked dismissed in later comparisons. Nothing about the sessions changes.
func Dismiss(workspace string, refs []string) (Comparison, error) {
	sessions, err := resolveSessions(workspace, refs)
	if err != nil {
		return Comparison{}, err
	}
	pairs, err := dismissedPairs(workspace)
	if err != nil {
		return Comparison{}, err
	}
	for i := 0; i < len(sessions); i++ {
		for j := i + 1; j < len(sessions); j++ {
			pairs[pairKey(sessions[i].ID, sessions[j].ID)] = true
		}
	}
	if err := writeDismissed(workspace, pairs); err != nil {
		return Comparison{}, err
	}
	return Compare(workspace, refs)
}

// MergeInput names what to merge. Into receives the unique items of From. Against lists
// sessions that were part of the comparison the person decided on, so conflict ids match;
// they are read, never changed. Resolve maps a conflict id to the session whose answer wins.
type MergeInput struct {
	Into    string
	From    []string
	Against []string
	Resolve map[string]string
}

// MergeResult says what a merge did.
type MergeResult struct {
	Into *Session `json:"into"`
	// Moved counts the items copied into the target; Answers counts the settled questions among them.
	Moved   int `json:"moved"`
	Answers int `json:"answers"`
	// Replaced counts the statements of the target rewritten by a conflict decision.
	Replaced int `json:"replaced"`
	// Merged lists the sessions that now link to the target.
	Merged []string `json:"merged"`
}

func (st *cState) indexOf(ref string) int {
	for i, s := range st.sessions {
		if s.ID == ref || strings.EqualFold(s.Ref(), ref) {
			return i
		}
	}
	return -1
}

// Merge moves the items that only the From sessions have into Into. Items found in more than
// one session are kept once. Every conflict needs a decision first. A sealed session cannot
// change, so it can be compared but never merged from or into.
func Merge(workspace string, in MergeInput) (MergeResult, error) {
	if len(in.From) == 0 {
		return MergeResult{}, fail("invalid", "merge needs at least one session to merge from")
	}
	refs := append(append([]string{in.Into}, in.From...), in.Against...)
	st, err := compareState(workspace, refs)
	if err != nil {
		return MergeResult{}, err
	}
	into := st.indexOf(in.Into)
	if into < 0 {
		return MergeResult{}, notFound(in.Into)
	}
	merging := map[int]bool{into: true}
	for _, ref := range in.From {
		i := st.indexOf(ref)
		if i < 0 {
			return MergeResult{}, notFound(ref)
		}
		if i == into {
			return MergeResult{}, fail("invalid", "session %s cannot be merged into itself", ref)
		}
		merging[i] = true
	}
	for i := range merging {
		if st.sessions[i].MergedInto != "" {
			return MergeResult{}, fail("invalid", "session %s was already merged into %s", st.sessions[i].Ref(), st.sessions[i].MergedInto)
		}
		if st.sessions[i].Status == StatusSealed {
			return MergeResult{}, fail("shape_sealed", "session %s is sealed; a sealed story cannot change, so it can be compared but not merged", st.sessions[i].Ref())
		}
	}
	winners, err := st.decisions(in.Resolve, merging)
	if err != nil {
		return MergeResult{}, err
	}
	return st.apply(workspace, into, merging, winners)
}

// decisions turns the resolve map into the winning session of every conflict that touches
// the merge, and refuses a merge that leaves one undecided.
func (st *cState) decisions(resolve map[string]string, merging map[int]bool) (map[int]int, error) {
	winners := map[int]int{}
	var open []string
	for ci, ids := range st.conflicts {
		id := st.cmp.Conflicts[ci].ID
		touches := false
		for _, i := range ids {
			if merging[st.items[i].sess] {
				touches = true
			}
		}
		if !touches {
			continue
		}
		pick, ok := resolve[id]
		if !ok {
			open = append(open, id)
			continue
		}
		w := st.indexOf(pick)
		hasWinner := false
		for _, i := range ids {
			if st.items[i].sess == w {
				hasWinner = true
			}
		}
		if w < 0 || !hasWinner {
			return nil, fail("invalid", "%s: %q is not one of the sessions that answer it", id, pick)
		}
		winners[ci] = w
	}
	if len(open) > 0 {
		return nil, fail("shape_conflicts_open", "decide %s first: the merge does not choose between two answers", strings.Join(open, ", "))
	}
	return winners, nil
}

func nextNumber(prefix string, ids []string) int {
	top := 0
	for _, id := range ids {
		if n, err := strconv.Atoi(strings.TrimPrefix(id, prefix)); err == nil && strings.HasPrefix(id, prefix) && n > top {
			top = n
		}
	}
	return top + 1
}

// apply writes the merge: the target changes first, then each merged session is marked.
func (st *cState) apply(workspace string, into int, merging map[int]bool, winners map[int]int) (MergeResult, error) {
	target := st.sessions[into]
	skip, rep, replaced := st.plan(into, merging, winners, target)
	var ruleIDs, exampleIDs, questionIDs, answerIDs []string
	for _, r := range target.Map.Rules {
		ruleIDs = append(ruleIDs, r.ID)
	}
	for _, e := range target.Map.Examples {
		exampleIDs = append(exampleIDs, e.ID)
	}
	for _, q := range target.Map.Questions {
		questionIDs = append(questionIDs, q.ID)
		if q.AnswerID != "" {
			answerIDs = append(answerIDs, q.AnswerID)
		}
	}
	result := MergeResult{Replaced: replaced, Merged: []string{}}
	// newID is the id an item has in the target: its own when it stays, a fresh one when it moves.
	newID := map[int]string{}
	moved := func(k int) bool {
		it := st.items[k]
		return it.sess != into && merging[it.sess] && !skip[k]
	}
	for k := range st.items {
		if st.items[k].sess == into {
			newID[k] = st.items[k].id
		}
	}
	for k, it := range st.items {
		if it.kind != ItemRule || !moved(k) {
			continue
		}
		rule := mergeRule(st.sessions[it.sess], it.id)
		rule.ID = "R" + strconv.Itoa(nextNumber("R", ruleIDs))
		ruleIDs = append(ruleIDs, rule.ID)
		newID[k] = rule.ID
		target.Map.Rules = append(target.Map.Rules, rule)
		result.Moved++
	}
	ruleOf := func(from *Session, ref string) string {
		for k, it := range st.items {
			if it.kind == ItemRule && st.sessions[it.sess] == from && it.id == ref {
				if id, ok := newID[k]; ok {
					return id
				}
				if r, ok := rep[k]; ok {
					return newID[r]
				}
			}
		}
		return ""
	}
	for k, it := range st.items {
		if it.kind != ItemExample || !moved(k) {
			continue
		}
		from := st.sessions[it.sess]
		ex := mergeExample(from, it.id)
		ex.ID = "E" + strconv.Itoa(nextNumber("E", exampleIDs))
		exampleIDs = append(exampleIDs, ex.ID)
		ex.Rule = ruleOf(from, ex.Rule)
		target.Map.Examples = append(target.Map.Examples, ex)
		result.Moved++
	}
	for k, it := range st.items {
		if it.kind != ItemQuestion || !moved(k) {
			continue
		}
		q := mergeQuestion(st.sessions[it.sess], it.id)
		q.ID = "Q" + strconv.Itoa(nextNumber("Q", questionIDs))
		questionIDs = append(questionIDs, q.ID)
		if q.AnswerID != "" {
			q.AnswerID = "A" + strconv.Itoa(nextNumber("A", answerIDs))
			answerIDs = append(answerIDs, q.AnswerID)
			result.Answers++
		}
		target.Map.Questions = append(target.Map.Questions, q)
		result.Moved++
	}
	// Open questions that only a merged session has move too, so nothing the person has to answer is lost.
	for src, from := range st.sessions {
		if src == into || !merging[src] {
			continue
		}
		for _, q := range from.Map.Questions {
			if q.Status != QuestionOpen || hasQuestion(target, q.Text) {
				continue
			}
			q.ID = "Q" + strconv.Itoa(nextNumber("Q", questionIDs))
			questionIDs = append(questionIDs, q.ID)
			target.Map.Questions = append(target.Map.Questions, q)
			result.Moved++
		}
	}
	if target.OpenQuestions() > 0 {
		target.Status = StatusInterviewing
	}
	if err := saveWithLabel(target, "Merged "+strings.Join(refsOf(st, merging, into), ", "), ActorPerson); err != nil {
		return MergeResult{}, err
	}
	for src, from := range st.sessions {
		if src == into || !merging[src] {
			continue
		}
		from.MergedInto = target.ID
		if err := saveWithLabel(from, "Merged into "+target.Ref(), ActorPerson); err != nil {
			return MergeResult{}, err
		}
		result.Merged = append(result.Merged, from.Ref())
	}
	result.Into = target
	return result, nil
}

func hasQuestion(s *Session, text string) bool {
	want := tokens(text)
	for _, q := range s.Map.Questions {
		if jaccard(want, tokens(q.Text)) >= sameScore {
			return true
		}
	}
	return false
}

// plan decides which items of the merged sessions stay where they are. skip marks an item that
// does not move because the target (or an item that moves instead) already says it; rep maps a
// skipped item to the item that stands for it. A decided conflict also rewrites the target's
// statement when another session's answer wins; replaced counts those rewrites.
func (st *cState) plan(into int, merging map[int]bool, winners map[int]int, target *Session) (skip map[int]bool, rep map[int]int, replaced int) {
	skip, rep = map[int]bool{}, map[int]int{}
	settle := func(ids []int, keep func(i int) bool) {
		standIn := -1
		for _, i := range ids {
			if st.items[i].sess == into {
				standIn = i
			}
		}
		for _, i := range ids {
			it := st.items[i]
			if it.sess == into || !merging[it.sess] {
				continue
			}
			switch {
			case standIn >= 0:
				skip[i], rep[i] = true, standIn
			case keep(i) && standIn < 0:
				standIn = i
			default:
				skip[i] = true
				if standIn >= 0 {
					rep[i] = standIn
				}
			}
		}
	}
	for _, ids := range st.groups {
		settle(ids, func(int) bool { return true })
	}
	for ci, ids := range st.conflicts {
		w, decided := winners[ci]
		if !decided {
			continue
		}
		winner := -1
		for _, i := range ids {
			if st.items[i].sess == w {
				winner = i
				break
			}
		}
		for _, i := range ids {
			if st.items[i].sess == into && w != into && replaceItem(target, st.items[i], st.items[winner]) {
				replaced++
			}
		}
		settle(ids, func(i int) bool { return st.items[i].sess == w })
	}
	return skip, rep, replaced
}

func refsOf(st *cState, merging map[int]bool, into int) []string {
	var out []string
	for i, s := range st.sessions {
		if merging[i] && i != into {
			out = append(out, s.Ref())
		}
	}
	return out
}

// saveWithLabel saves a session and names the revision that results.
func saveWithLabel(s *Session, label, actor string) error {
	path, err := revisionsPath(s.Workspace, s.ID)
	if err != nil {
		return err
	}
	revisionHints.Store(path, revisionHint{label: label, actor: actor})
	defer revisionHints.Delete(path)
	return Save(s)
}

// replaceItem rewrites the target's statement with the winning one and says whether it changed.
func replaceItem(target *Session, old, winner cItem) bool {
	switch old.kind {
	case ItemRule:
		for i := range target.Map.Rules {
			if target.Map.Rules[i].ID == old.id && target.Map.Rules[i].Text != winner.text {
				target.Map.Rules[i].Text = winner.text
				return true
			}
		}
	case ItemExample:
		for i := range target.Map.Examples {
			if target.Map.Examples[i].ID == old.id && target.Map.Examples[i].Text != winner.text {
				target.Map.Examples[i].Text = winner.text
				return true
			}
		}
	case ItemQuestion:
		for i := range target.Map.Questions {
			q := &target.Map.Questions[i]
			if q.ID == old.id && q.Answer != winner.value {
				q.Answer, q.Status = winner.value, QuestionAnswered
				return true
			}
		}
	}
	return false
}

func mergeRule(s *Session, id string) Rule {
	for _, r := range s.Map.Rules {
		if r.ID == id {
			return r
		}
	}
	return Rule{}
}

func mergeExample(s *Session, id string) Example {
	for _, e := range s.Map.Examples {
		if e.ID == id {
			return e
		}
	}
	return Example{}
}

func mergeQuestion(s *Session, id string) Question {
	for _, q := range s.Map.Questions {
		if q.ID == id {
			return q
		}
	}
	return Question{}
}
