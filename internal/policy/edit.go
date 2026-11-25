package policy

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"span/internal/agent"
	"span/internal/prove"
)

// Changes is what a person edits from the settings page. A nil field is left alone; a set
// one replaces the value. Only these keys can be written: everything else in span.yaml is
// kept exactly as it is.
type Changes struct {
	Base       *string
	Open       *string
	Merge      *string
	Promote    *string
	Plan       *string
	Conflict   *string
	Focused    *[]string
	Fast       *[]string
	Acceptance *[]string
	QA         *string
	Repair     *int
	// Agents sets the CLI of the stages it names; an empty name clears the entry. A stage it
	// does not name is left alone.
	Agents map[string]string
	// DoneStatus is the Jira status a finished run moves its story to; empty clears it.
	DoneStatus *string
}

// DiffLine is one value a save would change: the key as a person reads it, and its
// effective value before and after. A list is shown joined with a comma.
type DiffLine struct {
	Key    string `json:"key"`
	Before string `json:"before"`
	After  string `json:"after"`
}

// EditPreview is what an edit changes. It is empty when the file already holds every value.
type EditPreview struct {
	Lines []DiffLine `json:"lines"`
}

// edit is one key to write: where it lives, how a person reads it, and its old and new value.
type edit struct {
	section string
	key     string
	label   string
	before  string
	after   string
	list    []string
	isList  bool
}

// PreviewEdit says what Edit would change and writes nothing. It refuses what Edit refuses.
func PreviewEdit(path string, changes Changes) (EditPreview, error) {
	_, _, preview, err := planEdit(path, changes)
	return preview, err
}

// Edit applies changes to span.yaml. It rewrites only the lines of the keys named, so
// comments, key order and every other key stay where they were. The result is validated
// before it reaches the disk and replaces the file through a temp file. A change that
// leaves every value as it is writes nothing, and a missing file is not created for it.
func Edit(path string, changes Changes) (EditPreview, error) {
	next, mode, preview, err := planEdit(path, changes)
	if err != nil || len(preview.Lines) == 0 {
		return preview, err
	}
	return preview, writeChecked(path, next, mode)
}

// planEdit computes the new body of the file and the lines that differ, without writing.
func planEdit(path string, changes Changes) (string, os.FileMode, EditPreview, error) {
	raw, mode, err := readForEdit(path)
	if err != nil {
		return "", 0, EditPreview{}, err
	}
	current, err := Parse([]byte(raw))
	if err != nil {
		return "", 0, EditPreview{}, err
	}
	wanted, err := editsOf(current, changes)
	if err != nil {
		return "", 0, EditPreview{}, err
	}
	body := raw
	preview := EditPreview{Lines: []DiffLine{}}
	for _, item := range wanted {
		if item.before == item.after {
			continue
		}
		body, err = applyEdit(body, item)
		if err != nil {
			return "", 0, EditPreview{}, err
		}
		preview.Lines = append(preview.Lines, DiffLine{Key: item.label, Before: item.before, After: item.after})
	}
	if len(preview.Lines) == 0 {
		return raw, mode, preview, nil
	}
	if _, err := Parse([]byte(body)); err != nil {
		return "", 0, EditPreview{}, err
	}
	return body, mode, preview, nil
}

// editsOf turns the changes into one edit per key, in the order the page shows them, and
// refuses a value the file would not accept.
func editsOf(current File, c Changes) ([]edit, error) {
	var out []edit
	scalar := func(section, key string, before string, value *string) {
		if value == nil {
			return
		}
		label := key
		if section != "" {
			label = section + "." + key
		}
		out = append(out, edit{section: section, key: key, label: label, before: before, after: strings.TrimSpace(*value)})
	}
	scalar("", "base", current.Base, c.Base)
	scalar("stages", "open", current.OpenMode(), c.Open)
	scalar("stages", "merge", current.MergeMode(), c.Merge)
	scalar("stages", "promote", orAuto(current.Stages.Promote), c.Promote)
	scalar("stages", "plan", current.PlanMode(), c.Plan)
	scalar("stages", "conflict", current.ConflictMode(), c.Conflict)
	lists := []struct {
		key    string
		before []string
		value  *[]string
	}{
		{"focused", current.Prove.Focused, c.Focused},
		{"fast", current.Prove.Fast, c.Fast},
		{"acceptance", current.Prove.Acceptance, c.Acceptance},
	}
	for _, item := range lists {
		if item.value == nil {
			continue
		}
		clean, err := cleanCommands("prove."+item.key, *item.value)
		if err != nil {
			return nil, err
		}
		out = append(out, edit{
			section: "prove", key: item.key, label: "prove." + item.key,
			before: strings.Join(item.before, ", "), after: strings.Join(clean, ", "),
			list: clean, isList: true,
		})
	}
	scalar("qa", "command", current.QA.Command, c.QA)
	for stage := range c.Agents {
		if !isAgentStage(stage) {
			return nil, fmt.Errorf("agents: %q is not one of %s", stage, strings.Join(AgentStageNames, ", "))
		}
	}
	for _, stage := range AgentStageNames {
		if value, ok := c.Agents[stage]; ok {
			scalar("agents", stage, strings.TrimSpace(current.Agents[stage]), &value)
		}
	}
	scalar("jira", "doneStatus", current.DoneStatus(), c.DoneStatus)
	if c.Repair != nil {
		if *c.Repair < 1 || *c.Repair > MaxRepairAttempts {
			return nil, fmt.Errorf("repair.maxAttempts is %d, allowed 1 to %d", *c.Repair, MaxRepairAttempts)
		}
		out = append(out, edit{
			section: "repair", key: "maxAttempts", label: "repair.maxAttempts",
			before: fmt.Sprint(current.RepairCap()), after: fmt.Sprint(*c.Repair),
		})
	}
	for _, item := range out {
		if err := checkValue(item); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// checkValue refuses a scalar the file would not accept, so the page learns why before
// anything is written. The whole body is validated again after the rewrite.
func checkValue(item edit) error {
	if item.isList {
		return nil
	}
	switch item.label {
	case "stages.open", "stages.merge", "stages.promote", "stages.conflict":
		if item.after != "auto" && item.after != "hold" {
			return fmt.Errorf("%s %q is not auto or hold", item.label, item.after)
		}
	case "stages.plan":
		if item.after != "auto" && item.after != "hold" && item.after != "conditional" {
			return fmt.Errorf("stages.plan %q is not auto, conditional or hold", item.after)
		}
	case "agents.frame", "agents.build", "agents.inspect":
		if item.after != "" && !agent.Known(item.after) {
			return fmt.Errorf("%s %q is not one of auto, any, or %s", item.label, item.after, agent.IDs())
		}
	case "qa.command":
		if item.after != "" {
			if _, err := cleanCommands("qa.command", []string{item.after}); err != nil {
				return err
			}
		}
	}
	if strings.ContainsAny(item.after, "\r\n") {
		return fmt.Errorf("%s must be one line", item.label)
	}
	return nil
}

// cleanCommands trims a list of checks and refuses a line that is empty, spans lines, or is
// prose: shell punctuation is never executed, so a check written with it would silently
// never run.
func cleanCommands(label string, commands []string) ([]string, error) {
	out := make([]string, 0, len(commands))
	for _, command := range commands {
		clean := strings.TrimSpace(command)
		switch {
		case clean == "":
			return nil, fmt.Errorf("%s: a check cannot be empty", label)
		case strings.ContainsAny(clean, "\r\n"):
			return nil, fmt.Errorf("%s: %q must be one line", label, clean)
		case prove.Prose(clean):
			return nil, fmt.Errorf("%s: %q uses shell punctuation, and checks run as plain commands", label, clean)
		}
		out = append(out, clean)
	}
	return out, nil
}

// applyEdit writes one key into the body of the file.
func applyEdit(body string, item edit) (string, error) {
	lines := splitBody(body)
	var err error
	switch {
	case item.section == "":
		lines = setTopScalar(lines, item.key, item.after)
	case item.isList:
		lines, err = setList(lines, item.section, item.key, item.list)
	default:
		lines, err = setScalar(lines, item.section, item.key, item.after)
	}
	if err != nil {
		return "", err
	}
	if len(lines) == 0 {
		return "", nil
	}
	return strings.Join(lines, "\n") + "\n", nil
}

func splitBody(body string) []string {
	if body == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(body, "\n"), "\n")
}

// scalarText is a value as it is written after "key: ", quoted only when YAML needs it.
func scalarText(value string) string {
	if _, err := strconv.Atoi(value); err == nil {
		return value
	}
	out, err := yaml.Marshal(value)
	if err != nil {
		return value
	}
	return strings.TrimSuffix(string(out), "\n")
}

func indentOf(line string) int {
	return len(line) - len(strings.TrimLeft(line, " \t"))
}

func skippable(line string) bool {
	trimmed := strings.TrimSpace(line)
	return trimmed == "" || strings.HasPrefix(trimmed, "#")
}

// setTopScalar writes or clears a top-level key such as base. A new key goes after agent:
// or feature: when the file has one, else at the end.
func setTopScalar(lines []string, key, value string) []string {
	for i, line := range lines {
		if !strings.HasPrefix(line, key+":") {
			continue
		}
		if value == "" {
			return append(lines[:i], lines[i+1:]...)
		}
		lines[i] = key + ": " + scalarText(value)
		return lines
	}
	if value == "" {
		return lines
	}
	entry := key + ": " + scalarText(value)
	for _, anchor := range []string{"agent:", "feature:"} {
		for i, line := range lines {
			if strings.HasPrefix(line, anchor) {
				return append(lines[:i+1], append([]string{entry}, lines[i+1:]...)...)
			}
		}
	}
	return append(lines, entry)
}

// section is where a top-level block sits: its header, the indent of its children, and the
// line after its last child. Comments and blank lines after the last child belong to what
// follows, so a new key lands directly under its siblings.
type section struct {
	header int
	indent int
	end    int
}

// findSection locates a block. It reports false when the file has none, and an error when
// the block is written inline ("stages: {open: auto}"), which this editor will not guess at.
func findSection(lines []string, name string) (section, bool, error) {
	for i, line := range lines {
		if !strings.HasPrefix(line, name+":") {
			continue
		}
		rest := strings.TrimSpace(strings.TrimPrefix(line, name+":"))
		if rest != "" && !strings.HasPrefix(rest, "#") {
			return section{}, false, fmt.Errorf("%s: is written inline; rewrite it as a block to change it from here", name)
		}
		found := section{header: i, indent: 2, end: i + 1}
		first := true
		for j := i + 1; j < len(lines); j++ {
			if skippable(lines[j]) {
				continue
			}
			if indentOf(lines[j]) == 0 && !strings.HasPrefix(strings.TrimSpace(lines[j]), "- ") {
				break
			}
			if first {
				found.indent = indentOf(lines[j])
				first = false
			}
			found.end = j + 1
		}
		return found, true, nil
	}
	return section{}, false, nil
}

// findKey is the line of a key directly under the section header, or -1.
func findKey(lines []string, in section, key string) int {
	for i := in.header + 1; i < in.end; i++ {
		if indentOf(lines[i]) == in.indent && strings.HasPrefix(strings.TrimSpace(lines[i]), key+":") {
			return i
		}
	}
	return -1
}

// insertAt puts lines in at index, keeping everything else.
func insertAt(lines []string, index int, add ...string) []string {
	out := make([]string, 0, len(lines)+len(add))
	out = append(out, lines[:index]...)
	out = append(out, add...)
	return append(out, lines[index:]...)
}

// setScalar writes a key under a section, creating the section when the file has none. An
// empty value removes the key, and the section with it when nothing else is left in it.
func setScalar(lines []string, name, key, value string) ([]string, error) {
	in, ok, err := findSection(lines, name)
	if err != nil {
		return nil, err
	}
	if !ok {
		if value == "" {
			return lines, nil
		}
		return append(lines, name+":", "  "+key+": "+scalarText(value)), nil
	}
	at := findKey(lines, in, key)
	pad := strings.Repeat(" ", in.indent)
	switch {
	case at >= 0 && value == "":
		lines = append(lines[:at], lines[at+1:]...)
		left, found, err := findSection(lines, name)
		if err != nil {
			return nil, err
		}
		if found && left.end == left.header+1 {
			lines = append(lines[:left.header], lines[left.header+1:]...)
		}
		return lines, nil
	case at >= 0:
		lines[at] = pad + key + ": " + scalarText(value)
	case value != "":
		lines = insertAt(lines, in.end, pad+key+": "+scalarText(value))
	}
	return lines, nil
}

// setList writes a list of commands under a section. The key's old items, block or flow, are
// replaced; an empty list is written as [].
func setList(lines []string, name, key string, items []string) ([]string, error) {
	in, ok, err := findSection(lines, name)
	if err != nil {
		return nil, err
	}
	if !ok {
		if len(items) == 0 {
			return lines, nil
		}
		return append(append(lines, name+":"), listLines("  ", key, items)...), nil
	}
	at := findKey(lines, in, key)
	pad := strings.Repeat(" ", in.indent)
	if at < 0 {
		if len(items) == 0 {
			return lines, nil
		}
		return insertAt(lines, in.end, listLines(pad, key, items)...), nil
	}
	stop := at + 1
	for j := at + 1; j < in.end; j++ {
		if skippable(lines[j]) {
			continue
		}
		deeper := indentOf(lines[j]) > in.indent
		dash := indentOf(lines[j]) == in.indent && strings.HasPrefix(strings.TrimSpace(lines[j]), "- ")
		if !deeper && !dash {
			break
		}
		stop = j + 1
	}
	return append(lines[:at], append(listLines(pad, key, items), lines[stop:]...)...), nil
}

func listLines(pad, key string, items []string) []string {
	if len(items) == 0 {
		return []string{pad + key + ": []"}
	}
	out := []string{pad + key + ":"}
	for _, item := range items {
		out = append(out, pad+"  - "+scalarText(item))
	}
	return out
}
