package engine

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// A new worktree has the code and none of what is installed beside it. Span reads the
// lockfiles and installs it, so a run works without anything configured. prove.prepare
// replaces what is detected here.

type prepareStep struct {
	// dir is where the command runs.
	dir     string
	command string
}

// installer is one way to install, told apart by the file that locks it.
type installer struct {
	// lock is the file that selects the command, installed the folder it creates.
	lock, installed, command string
}

// ecosystems are tried in order within each group; the first match of a group wins.
var ecosystems = [][]installer{
	{
		{"pnpm-lock.yaml", "node_modules", "pnpm install --frozen-lockfile"},
		{"yarn.lock", "node_modules", "yarn install --frozen-lockfile"},
		{"bun.lock", "node_modules", "bun install --frozen-lockfile"},
		{"bun.lockb", "node_modules", "bun install --frozen-lockfile"},
		{"package-lock.json", "node_modules", "npm ci"},
		// Without a lockfile, install without writing one into the story's changes.
		{"package.json", "node_modules", "npm install --no-package-lock"},
	},
	{
		{"uv.lock", ".venv", "uv sync --frozen"},
		{"poetry.lock", ".venv", "poetry install"},
	},
	{{"composer.lock", "vendor", "composer install"}},
}

// detectPrepare lists what the tree at root needs installed, at the root and in its
// immediate folders. A folder with only a manifest, under a root of the same kind, is left
// to the root, as a workspace expects. redo installs even where the folder is there, for an
// install that did not finish.
func detectPrepare(root string, redo bool) []prepareStep {
	dirs := []string{root}
	for _, entry := range listDir(root) {
		name := entry.Name()
		if entry.IsDir() && !strings.HasPrefix(name, ".") && name != "node_modules" && name != "vendor" {
			dirs = append(dirs, filepath.Join(root, name))
		}
	}
	var steps []prepareStep
	atRoot := map[int]bool{}
	for _, dir := range dirs {
		for kind, group := range ecosystems {
			found, ok := firstLocked(dir, group)
			if !ok || (atRoot[kind] && found.lock == "package.json") {
				continue
			}
			if dir == root {
				atRoot[kind] = true
			}
			if redo || !present(filepath.Join(dir, found.installed)) {
				steps = append(steps, prepareStep{dir: dir, command: command(dir, found)})
			}
		}
	}
	return steps
}

// command is what installs in dir: the installer's own, unless the folder says its tool
// wants another.
func command(dir string, found installer) string {
	switch {
	case found.lock == "yarn.lock" && present(filepath.Join(dir, ".yarnrc.yml")):
		return "yarn install --immutable"
	case found.lock == "package.json":
		switch packageManager(dir) {
		case "pnpm":
			return "pnpm install --no-lockfile"
		case "yarn":
			return "yarn install --no-lockfile"
		case "bun":
			return "bun install --no-save"
		}
	}
	return found.command
}

// packageManager is the tool package.json names in packageManager, or "".
func packageManager(dir string) string {
	raw, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		log.Printf("read the package manager of %s: %v", dir, err)
		return ""
	}
	var manifest struct {
		PackageManager string `json:"packageManager"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		log.Printf("read the package manager of %s: %v", dir, err)
		return ""
	}
	name, _, _ := strings.Cut(manifest.PackageManager, "@")
	return name
}

// firstLocked is the first installer of the group whose file is in dir.
func firstLocked(dir string, group []installer) (installer, bool) {
	for _, candidate := range group {
		if present(filepath.Join(dir, candidate.lock)) {
			return candidate, true
		}
	}
	return installer{}, false
}

func present(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
