package main

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"span/internal/serve"
)

const settingsUsage = `usage: ./span settings show [--project .] [--json]
       ./span settings set [--project .] [--preview] [--json] key=value...
       keys: base open merge promote plan conflict focused fast acceptance qa repair doneStatus agents.<stage>
       ./span settings agent [--project .] [--set auto|any|<cli>] [--json]
       ./span settings model --cli <cli> [--project .] [--set <model>] [--json]   (no --set lists the models)
       ./span settings folder <path> [--json]
       (each does what GET/POST /config, /agent, /agent/models and /onboarding/folder do; exit 0 done, 1 refused, 2 usage)`

func settingsCmd(args []string) int { return settingsMain(args, os.Stdout, os.Stderr) }

// settingsMain runs the settings verbs with the same code as the settings routes of span serve.
func settingsMain(args []string, out, errw io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(errw, settingsUsage)
		return 2
	}
	verb, rest := args[0], args[1:]
	project, set, cli := ".", "", ""
	asJSON, preview, hasSet := false, false, false
	var pos []string
	for i := 0; i < len(rest); i++ {
		next := func() (string, bool) {
			if i+1 >= len(rest) {
				return "", false
			}
			i++
			return rest[i], true
		}
		var ok bool
		switch rest[i] {
		case "--json":
			asJSON = true
		case "--preview":
			preview = true
		case "--project":
			project, ok = next()
		case "--set":
			set, ok = next()
			hasSet = true
		case "--cli":
			cli, ok = next()
		default:
			if strings.HasPrefix(rest[i], "--") {
				fmt.Fprintln(errw, settingsUsage)
				return 2
			}
			pos = append(pos, rest[i])
			continue
		}
		if !ok && rest[i] != "--json" && rest[i] != "--preview" {
			fmt.Fprintln(errw, settingsUsage)
			return 2
		}
	}
	show := func(value any, err error) int {
		if err != nil {
			return failWith(errw, asJSON, err)
		}
		return emit(out, errw, value)
	}
	ctx := context.Background()
	switch verb {
	case "show":
		if len(pos) != 0 {
			break
		}
		return show(serve.ConfigOf(project), nil)
	case "set":
		form := url.Values{}
		for _, pair := range pos {
			key, value, found := strings.Cut(pair, "=")
			if !found || key == "" {
				fmt.Fprintln(errw, settingsUsage)
				return 2
			}
			form.Set(key, value)
		}
		if len(form) == 0 {
			break
		}
		return show(serve.EditConfig(project, form, !preview))
	case "agent":
		if len(pos) != 0 {
			break
		}
		if !hasSet {
			return show(serve.AgentOf(project), nil)
		}
		return show(serve.ChooseAgent(project, set))
	case "model":
		if len(pos) != 0 || cli == "" {
			break
		}
		if !hasSet {
			return show(serve.ModelsOf(ctx, cli))
		}
		return show(serve.ChooseModel(project, cli, strings.TrimSpace(set)))
	case "folder":
		if len(pos) != 1 {
			break
		}
		return show(serve.ProbeFolderOf(ctx, pos[0]), nil)
	}
	fmt.Fprintln(errw, settingsUsage)
	return 2
}
