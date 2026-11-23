package prove

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// maxDiffFiles bounds the diff a focused band is derived from. A change that large is
// better proved by the whole suite than by a command line that names half the repository.
const maxDiffFiles = 200

// DiffFocus derives a focused band from the files this branch changed, for a story whose
// criteria and policy named none: go test runs the changed packages and every package
// that imports them, test-only imports included, and vitest or jest run the tests related
// to the changed files. nil means no safe narrowing exists and the wide band keeps
// deciding. Anything it cannot reason about, a go.mod, a config file, a lockfile, an
// asset, a runner it does not know, gives up instead of guessing. Inputs are not changed.
func DiffFocus(ctx context.Context, check Checker, dir string, commands, changed []string) []string {
	if len(changed) == 0 || len(changed) > maxDiffFiles {
		return nil
	}
	for _, command := range commands {
		clean := strings.TrimSpace(command)
		if clean == "" || Prose(clean) {
			continue
		}
		fields := strings.Fields(clean)
		if len(fields) >= 2 && fields[0] == "go" && fields[1] == "test" {
			return goFocus(ctx, check, dir, fields[2:], changed)
		}
		if prefix, rest, ok := jsRunner(fields, "vitest"); ok {
			return vitestFocus(prefix, rest, changed)
		}
		if prefix, rest, ok := jsRunner(fields, "jest"); ok {
			return jestFocus(prefix, rest, changed)
		}
	}
	return nil
}

// goValueFlags are the go test flags whose value may be the next word.
var goValueFlags = map[string]bool{
	"-run": true, "-tags": true, "-timeout": true, "-count": true, "-p": true, "-parallel": true,
	"-cpu": true, "-bench": true, "-benchtime": true, "-coverprofile": true, "-covermode": true,
	"-ldflags": true, "-vet": true, "-skip": true, "-shuffle": true, "-coverpkg": true,
}

// goRefusedFlags change what go test builds or where, in a way go list would have to repeat
// exactly: arguments for the test binary, another module file, another directory.
var goRefusedFlags = map[string]bool{"-args": true, "-mod": true, "-modfile": true, "-C": true}

// goListed is the part of a go list -json record the closure needs.
type goListed struct {
	ImportPath   string
	Dir          string
	Imports      []string
	TestImports  []string
	XTestImports []string
}

func goFocus(ctx context.Context, check Checker, dir string, args, changed []string) []string {
	var flags []string
	tags := ""
	patterns := 0
	for i := 0; i < len(args); i++ {
		word := args[i]
		switch {
		case word == "./...":
			patterns++
		case strings.HasPrefix(word, "-"):
			name, value, hasValue := strings.Cut(word, "=")
			if goRefusedFlags[strings.Replace(name, "--", "-", 1)] {
				return nil
			}
			flags = append(flags, word)
			if goValueFlags[name] && !hasValue {
				if i+1 >= len(args) {
					return nil
				}
				i++
				value = args[i]
				flags = append(flags, value)
			}
			if name == "-tags" {
				tags = value
			}
		default:
			// Another package pattern, or a value for a flag this table does not know.
			return nil
		}
	}
	if patterns != 1 {
		return nil
	}
	dirs := map[string]bool{}
	var docs []string
	for _, file := range changed {
		file = filepath.ToSlash(strings.TrimSpace(file))
		switch {
		case strings.HasSuffix(file, ".md"):
			docs = append(docs, path.Dir(file))
			continue
		case !strings.HasSuffix(file, ".go"), hasSegment(file, "testdata"), hasSegment(file, "vendor"):
			return nil
		}
		dirs[path.Dir(file)] = true
	}
	if len(dirs) == 0 {
		return nil
	}
	list := "go list -json=ImportPath,Dir,Imports,TestImports,XTestImports"
	if tags != "" {
		list += " -tags " + tags
	}
	out, err := check.Run(ctx, list+" ./...", dir)
	if err != nil || out.ExitCode != 0 || out.Truncated {
		return nil
	}
	pkgs, err := decodeListing(out.Text)
	if err != nil || len(pkgs) == 0 {
		return nil
	}
	byDir := map[string]string{}
	importers := map[string][]string{}
	known := map[string]bool{}
	for _, pkg := range pkgs {
		known[pkg.ImportPath] = true
	}
	for _, pkg := range pkgs {
		rel, ok := relDir(dir, pkg.Dir)
		if !ok {
			return nil
		}
		byDir[rel] = pkg.ImportPath
		for _, list := range [][]string{pkg.Imports, pkg.TestImports, pkg.XTestImports} {
			for _, imported := range list {
				if known[imported] && imported != pkg.ImportPath {
					importers[imported] = append(importers[imported], pkg.ImportPath)
				}
			}
		}
	}
	for _, doc := range docs {
		// A test can embed or read a Markdown file in its own tree; only one outside every
		// package directory is known not to matter.
		for pkgDir := range byDir {
			if pkgDir == "." || doc == pkgDir || strings.HasPrefix(doc, pkgDir+"/") {
				return nil
			}
		}
	}
	affected := map[string]bool{}
	var queue []string
	for changedDir := range dirs {
		pkg, ok := byDir[changedDir]
		if !ok {
			// Not a package this command tests: a build-excluded directory or a nested module.
			return nil
		}
		if !affected[pkg] {
			affected[pkg] = true
			queue = append(queue, pkg)
		}
	}
	for len(queue) > 0 {
		next := queue[0]
		queue = queue[1:]
		for _, importer := range importers[next] {
			if !affected[importer] {
				affected[importer] = true
				queue = append(queue, importer)
			}
		}
	}
	if len(affected) == 0 || len(affected) >= len(known) {
		return nil
	}
	names := make([]string, 0, len(affected))
	for name := range affected {
		names = append(names, name)
	}
	sort.Strings(names)
	words := append([]string{"go", "test"}, flags...)
	return []string{strings.Join(append(words, names...), " ")}
}

func decodeListing(text string) ([]goListed, error) {
	decoder := json.NewDecoder(strings.NewReader(text))
	var pkgs []goListed
	for {
		var pkg goListed
		err := decoder.Decode(&pkg)
		if errors.Is(err, io.EOF) {
			return pkgs, nil
		}
		if err != nil {
			return nil, err
		}
		if pkg.ImportPath == "" || pkg.Dir == "" {
			return nil, errors.New("go list printed a package without a path")
		}
		pkgs = append(pkgs, pkg)
	}
}

// relDir is a package directory relative to the workspace, in the slash form git prints.
// A temporary directory can be reached through a symlink, so both spellings are tried.
func relDir(root, dir string) (string, bool) {
	try := func(base, target string) (string, bool) {
		rel, err := filepath.Rel(base, target)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return "", false
		}
		return filepath.ToSlash(rel), true
	}
	if rel, ok := try(root, dir); ok {
		return rel, true
	}
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", false
	}
	if rel, ok := try(realRoot, dir); ok {
		return rel, true
	}
	realDir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return "", false
	}
	return try(realRoot, realDir)
}

func hasSegment(file, segment string) bool {
	for _, part := range strings.Split(file, "/") {
		if part == segment {
			return true
		}
	}
	return false
}

// jsLaunchers are the ways a project starts a JavaScript runner directly. A package.json
// script (npm test, pnpm test) is not one: what it runs is not known.
var jsLaunchers = [][]string{{}, {"npx"}, {"pnpm"}, {"pnpm", "exec"}, {"yarn"}}

// jsRunner splits a command into its launcher and runner words, and what follows them.
func jsRunner(fields []string, runner string) ([]string, []string, bool) {
	for _, launcher := range jsLaunchers {
		n := len(launcher)
		if len(fields) <= n || fields[n] != runner {
			continue
		}
		match := true
		for i, word := range launcher {
			if fields[i] != word {
				match = false
				break
			}
		}
		if match {
			return fields[:n+1], fields[n+1:], true
		}
	}
	return nil, nil, false
}

var jsExtensions = map[string]bool{
	".ts": true, ".tsx": true, ".js": true, ".jsx": true, ".mjs": true, ".cjs": true,
	".vue": true, ".svelte": true,
}

// jsFiles are the changed source files a related-tests run can follow, sorted. nil when a
// change touches anything the runner cannot trace through imports: a manifest, a lockfile,
// a config or setup file, a mock directory, a stylesheet, an asset.
func jsFiles(changed []string) []string {
	var files []string
	for _, file := range changed {
		file = filepath.ToSlash(strings.TrimSpace(file))
		base := strings.ToLower(path.Base(file))
		switch {
		case strings.HasSuffix(file, ".md"):
			continue
		case !jsExtensions[path.Ext(file)], strings.Contains(base, ".config."), strings.Contains(base, "setup"),
			strings.HasPrefix(base, "vitest.workspace."), strings.HasPrefix(base, "vitest.config"), strings.HasPrefix(base, "vite.config"),
			hasSegment(file, "__mocks__"), hasSegment(file, "node_modules"):
			return nil
		}
		files = append(files, file)
	}
	sort.Strings(files)
	return files
}

// jsFlags keeps every flag of the original command. A positional word is a filter, a
// subcommand, or a flag value this code cannot tell apart, so it gives up on it.
func jsFlags(rest []string, drop string) ([]string, bool) {
	var flags []string
	for _, word := range rest {
		switch {
		case word == drop:
			continue
		case strings.HasPrefix(word, "-"):
			flags = append(flags, word)
		default:
			return nil, false
		}
	}
	return flags, true
}

func vitestFocus(prefix, rest, changed []string) []string {
	files := jsFiles(changed)
	if len(files) == 0 {
		return nil
	}
	if len(rest) > 0 && rest[0] == "run" {
		rest = rest[1:]
	}
	flags, ok := jsFlags(rest, "--run")
	if !ok {
		return nil
	}
	words := append(append([]string{}, prefix...), "related", "--run")
	words = append(append(words, flags...), files...)
	return []string{strings.Join(words, " ")}
}

func jestFocus(prefix, rest, changed []string) []string {
	files := jsFiles(changed)
	if len(files) == 0 {
		return nil
	}
	for _, word := range rest {
		if strings.HasPrefix(word, "--findRelatedTests") {
			return nil
		}
	}
	flags, ok := jsFlags(rest, "")
	if !ok {
		return nil
	}
	words := append(append([]string{}, prefix...), flags...)
	words = append(append(words, "--findRelatedTests"), files...)
	return []string{strings.Join(words, " ")}
}
