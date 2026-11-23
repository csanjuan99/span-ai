# Span

Usage: [USAGE.md](USAGE.md).

From this folder, run `./span`. The shell does not search the current directory, so `span` alone is not a command and nothing is added to `PATH`.

`./span serve` opens the quickstart page. The **Folder** field is the project. The Jira login is saved in that folder at `.span/jira.json`, not once for the user. `./span repos` lists the checkout you point at, or the checkouts one level inside a folder. A story can run in one repo or in several. Each selected repo gets its own run and its own `feat/`, `fix/`, or `hotfix/` branch.

Span runs one feature from a brief to a deploy. The same Go binary runs on a laptop or a server. Reports are the control surface.

The coding agent is whichever CLI you already have with Gentle AI: Cursor (`cursor-agent`), Claude (`claude`), Codex (`codex`), or Pi (`pi`). Choose one on the page, or set `agent:` in `span.yaml`. The agent must follow Organic Driven Development and leave `odd/tasks/<feature>.md` before any change request is opened. [harness/skills](harness/README.md) is listed by path in the build prompt. Skill bodies are not pasted in.

## Commands

```bash
go build -o span ./cmd/span
./span serve --addr 127.0.0.1:8787
./span repos --path /path/to/project
./span jira configure --project /path/to/project --url https://your-site.atlassian.net --email you@example.com --token '<token>'
./span jira status --project /path/to/project
./span quickstart --workspace /path/to/project --issue AET-1234 --file ./notes.md
./span run --brief brief.md --workspace /path/to/project
./span resume <id> --workspace /path/to/project
```

`./span quickstart` writes `span.yaml` when the project does not have one. It does not overwrite an existing file unless `--force` is set. `span.example.yaml` is a hand-written policy, not the file quickstart writes.

`SPAN_TOKEN` is required when `--addr` is not a loopback address. Clients send `Authorization: Bearer <token>`.

Run state lives in `<workspace>/.span/runs/<id>/`. The Jira token for that folder lives in `<workspace>/.span/jira.json`. Stage instructions can be added at `span/stages/<stage>.md` or, for one run, `.span/runs/<id>/instructions/<stage>.md`. Files attached on the form or with `--file` are copied into `.span/runs/<id>/handoffs/` before the first frame.

An agent that ran out of quota or credentials holds with the CLI named and what to do about it. Prove runs `prove.fast`, then `prove.acceptance`. A failing check returns to build with the command, its exit code, the last lines it printed, and the path to the full log, up to `repair.maxAttempts` (3 by default). The same failure twice holds as `no progress`; the same command passing and then failing on one commit holds as `inconclusive`. A killed process resumes from the cursor. A deploy that was started and cannot be confirmed is not run again.

## Checks

```bash
go test ./...
```
