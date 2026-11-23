# Using Span

Span takes a brief through to deploy. The same binary runs on a laptop or a server. Each stage writes a statement: `advance` continues, `hold` pauses, `stop` ends the span.

## Quickstart

Build the page, then the binary. `./span quickstart` waits until the run finishes and prints text. The page returns as soon as the run is opened and follows it.

```bash
cd /path/to/span
cd web && npm install && npm run build && cd ..
go build -o span ./cmd/span

./span quickstart --workspace /path/to/project --story "Add saved cards"
./span serve --addr 127.0.0.1:8787
```

Open `http://127.0.0.1:8787`. Find the repos, save the Jira account, load a story, and start the run. **Configure and run** answers with the span id and opens that span. Those actions are the same operations as `./span repos`, `./span jira`, and `./span quickstart`. Restart `./span serve` after rebuilding the binary.

What auto-configuration writes, only if `span.yaml` is missing:

- `agent: any`
- `open: auto` and `merge: hold`
- one prove command when the repository root shows it: `go test ./...`, `npm test`, `cargo test`, or `python -m pytest`
- the harness path and the skill names under `harness/skills`

`--force` rewrites `span.yaml`. An existing file is left in place.

The project must already be a git repository with the base branch on `origin` (or locally, when there is no `origin` remote): `dev` for `feat/` and `fix/`, `master` for `hotfix/`, unless the run or `span.yaml` names another one. See [Where a branch starts and where its PR goes](#where-a-branch-starts-and-where-its-pr-goes).

## Where Span writes

Span reads the folder you give it.

- A git checkout is that repo. Running with `--workspace` pointed at `salute-infra` writes there.
- A folder of checkouts, such as `salute/` with `salute-apollo`, `salute-infra`, and `salute-hermes`, lists those repos. Pick one or several. Each selected repo gets its own run and its own branch, and the agent is told to write only in that repo.
- If the folder is itself a checkout and also contains other checkouts, both are listed and you choose.

Span looks one directory down. Hidden directories, `node_modules`, and `vendor` are skipped.

```bash
./span repos --path /path/to/salute
./span quickstart --workspace /path/to/salute --repo salute-infra --repo salute-apollo --issue AET-1478
```

`--repo` repeats once per checkout. With a single repo, omit it. The page at `./span serve` uses the same choice: find repos, then tick the ones this story touches.

| Message | What to fix |
| --- | --- |
| `no_repo` | The path is missing, or it is not a checkout and it has no checkout one level down |
| `selection_required` | The folder lists more than one checkout. Pass `--repo` once per name, or tick them on the page |
| `unknown_repo` | That name is not one of the checkouts in the folder |
| `invalid: files need a story` | A file was attached and neither the story text nor an issue key was set |

## Jira Cloud

Span reads a Jira Cloud story and uses it as the brief. It comments and moves the issue only when the story is missing information. It does not edit the issue fields.

You need three values from the site you already use in the browser:

| Value | Where it comes from |
| --- | --- |
| Site | The origin in the address bar, `https://your-site.atlassian.net` |
| Email | The Atlassian account that can open the story |
| API token | Created once at <https://id.atlassian.com/manage-profile/security/api-tokens> |

Create the token from that page with **Create API token**, copy it, and paste it into the form or the command below. Jira Cloud does not redirect back with the token. An OAuth app could, and Span does not use one. The normal path is the form or `./span jira configure`.

The written steps live in this file, under this heading. The page at `http://127.0.0.1:8787` shows the same link above the Jira fields. With an email, Span sends HTTP basic auth. With the email left empty, it sends the token as a bearer token. The account needs permission to browse the project, add comments, and transition issues. Jira Cloud often answers 404 when the account cannot see the issue, and Span reports that as `not_found`. Keep the token out of the repo and out of shell history.

### Save the account

The login belongs to one folder. A parent folder and a checkout inside it are different projects. Save the account in the same folder you pass as `--workspace` or type in the form. `--project` defaults to `.`, the folder you are in when you run `./span`.

```bash
./span jira configure \
  --project /path/to/salute \
  --url https://your-site.atlassian.net \
  --email you@example.com \
  --token '<paste the token>'
./span jira status --project /path/to/salute
```

`status` prints `configured true`, the project folder, the site, and the email. It does not print the token.

On the page, the **Folder** field is that project. A saved account shows **Logged in to Jira as** the email and the site **for this folder**, and hides the token fields. Changing the folder loads that folder's login. **Close session** removes only that folder's login. The command is `./span jira logout --project /path/to/salute`.

On `http://127.0.0.1:8787` the first save is the **Jira session** form: site, email, API token, **Save session**. The token field is cleared after a successful save.

The token is written to `<project>/.span/jira.json`, mode `0600`. Span also writes `<project>/.span/.gitignore` containing `*` so the file stays out of git. A later configure with an empty token keeps the token already stored for that project and updates the site or the email.

A login saved for one folder is not used by another folder. Environment variables do not log every project in. An older copy under the OS user config directory is not read. Save the token again in each project folder.

### Read a story

```bash
./span jira issue --project /path/to/salute AET-1478
```

That calls `GET /rest/api/3/issue/AET-1478` and prints the brief Span will execute:

```text
AET-1478 Show channels
Type: Story
Status: To Do
URL: https://your-site.atlassian.net/browse/AET-1478

Show the channel list.

Acceptance criteria:
Includes archived channels.
```

The description is the Jira document, turned into plain text. Acceptance criteria are included when the issue has a field whose name contains `Acceptance Criteria`. A story with no such field still runs from the summary and the description.

List up to 20 issues:

```bash
./span jira search --project /path/to/salute
./span jira search --project /path/to/salute --jql "project = AET AND type = Story ORDER BY updated DESC"
```

With no `--jql`, the query is `assignee = currentUser() ORDER BY updated DESC`. Search uses `POST /rest/api/3/search/jql`. The old `/rest/api/3/search` endpoint is not called.

On the page, **My stories** runs that default search. **Load one story** reads the key in the story field and fills the text area with the brief.

### Missing information

Frame reads the story in the language it is written. Criteria the story already states are written under the one standard heading `Acceptance criteria:`. Span does not keep a list of translated headings. A question is a hold on the same span. The page lists each question. **Answer and continue** records the reply and runs frame again. **Read the story again** loads the Jira issue and runs frame on that text. A missing Jira account is not a question and does not stop the run. When the account is configured, Span comments on the issue and does not move it.

A run with no issue key waits locally and does not call Jira. Out of scope stays a stop. That run does not continue.

### Run that story

Leave `--story` empty. The issue text is the brief, and the key is the branch suffix.

```bash
./span quickstart --workspace /path/to/salute-infra --issue AET-1478
./span quickstart --workspace /path/to/salute --repo salute-infra --repo salute-apollo --issue AET-1478
```

The first command writes in that checkout. The second starts one run in each selected repo, with the same brief, and tells the agent to write only in that repo. The branch looks like `feat/AET-1478-show-channels`. `--kind fix` or `--base master` still chooses `fix/` or `hotfix/` as in [Branches](#branches).

If you also pass `--story`, that text is the brief and the issue key is still used for the branch. The page does this when you edit the text area after **Load one story**.

### When it fails

| Message | What to fix |
| --- | --- |
| `not_configured` | Save the account for this folder with `./span jira configure --project <folder>` or the page |
| `unauthorized` | Email and token do not match, or the token was revoked. Create a new token and configure again |
| `not_found` | The key is wrong, or this account cannot browse that issue |
| `unreachable` | The site URL does not answer. Check `https://your-site.atlassian.net` |
| `invalid` | The site URL is missing, or Jira rejected the query |

## Branches

Only three prefixes exist.

| Prefix | Starts from by default | Use |
| --- | --- | --- |
| `feat/` | `dev` | a new story |
| `fix/` | `dev` | an adjustment |
| `hotfix/` | `master` | an adjustment from master |

The defaults apply only when nobody chose a base. A base named with `--base`, the desk's
**Base branch** field, or `base:` in `span.yaml` may be any valid branch name, for any prefix.

The text after the prefix is inferred when the span starts. A linked key such as `AET-1478`, from `--issue` or from the story text, produces `feat/AET-1478-show-channels`. Without a key, the first line becomes a slug: `feat/add-saved-cards`.

`--kind` sets the prefix. When it is omitted, a story that starts with `fix:` or `bug:` is `fix/`, a base of `master` is `hotfix/`, and anything else is `feat/` from `dev`.

```bash
./span quickstart --workspace /path/to/project --kind fix --issue AET-1478 --story "Show channels"
./span quickstart --workspace /path/to/project --base master --story "Refund the double charge"
```

### Where a branch starts and where its PR goes

The base is one branch with two jobs: the story's branch is cut from it, and the pull
request opens into it (`gh pr create --base=<base> --head=<branch>`). Span always passes the
base; without it GitHub picks the repository's default branch, usually `main`, whatever
the branch was cut from.

The base is chosen in this order, first match wins:

1. The run names it: `--base` on `quickstart` or `run`, the **Base branch** field on the
   desk, or the `base` form field of `POST /quickstart`. It is used for any kind.
2. `base:` in `span.yaml`, for `feat/` and `fix/` only. A hotfix keeps `master`, and this
   line never decides the kind: with `base: master`, a story without `--kind` is still a
   `feat/` (or a `fix/` when it starts with `fix:`), now cut from `master`.
3. The default: `dev` for `feat/` and `fix/`, `master` for `hotfix/`.

Without `--kind`, a story is `feat/` unless it starts with `fix:`, `bug:` or `hotfix:`. The
one exception is kept from before: `--base master` named for the run, without `--kind`,
still means `hotfix/`. Any other named base leaves the inferred kind alone.

```yaml
# span.yaml: every feat and fix in this checkout starts from, and opens into, release/2.1
base: release/2.1
```

Stacked work is a base that is another feature branch. SAL-4 builds on SAL-3, which is not
merged yet:

```bash
./span quickstart --workspace /path/to/project --issue SAL-4 --base feat/SAL-3-wallet
```

The branch `feat/SAL-4-...` starts from `feat/SAL-3-wallet` and its pull request targets
`feat/SAL-3-wallet`, so the review shows only SAL-4's change. When SAL-3 merges, retarget
SAL-4's pull request to SAL-3's own base.

**What a base may be.** A branch name, as `git check-ref-format --branch` accepts it, that
is not `HEAD`, does not start with `-`, has no spaces around it, and does not look like a
commit (7 to 40 hex digits). Tags and commits are not bases. Anything else is refused
before git or gh sees it, wherever it came from (flag, desk, form, `span.yaml`):

```text
invalid base branch "--repo=evil/x": use a branch name, not an option, HEAD, or a commit
```

**Where the branch is cut from.** Span first fetches the base from `origin` (best effort,
into `refs/remotes/origin/<base>` only; your own branch is never moved). When origin has it,
the new branch is cut from `origin/<base>`, even when your local branch is behind or has
diverged from it: a stale local `dev` must not decide where new work starts. Without
origin's copy, the local branch is used; offline, the local branch wins silently. The new
branch does not track its base.

**The base must be on origin.** Before the first stage, and again just before the push at
open, the base must exist on `origin` (a repository without an `origin` remote has no forge,
so a local branch is enough there). A base that is only local, a tag, or missing holds the
run, before any build work and before any push:

```text
base branch feat/SAL-3-wallet does not exist on origin; push it or choose another base
```

A run whose base is the branch it is on also holds before the first stage. A run with no
base at all holds at open instead of opening a pull request.

**A pull request on the wrong base holds.** Span looks at every open pull request for the
branch. One that targets the base is reused. When none does, for example one opened against
`main` before this check existed, Span does not reuse it and does not edit it:

```text
the open change https://github.com/o/r/pull/7 targets main but this run's base is dev; retarget it (gh pr edit 7 --base dev) or close it, then resume
```

With several, each is named and the command retargets the first. Retarget or close, then
resume the run.

**`span run` now has a base too.** It used to open into whatever GitHub called the default
branch. Now it uses `--base`, else `base:` in `span.yaml`, else `dev` (`master` for a
`--branch hotfix/...`). This is a deliberate change: on a repository without that base, or
on a checkout that is on the base itself, the run holds before the first stage. The base is
also recorded on the run, so [Prove only what the change can affect](#prove-only-what-the-change-can-affect)
diffs against it.

The desk fills the field's suggestions from `GET /branches?path=<folder>`: the local and
`origin` branches of that repository, `origin/` stripped, the default base first, then the
50 most recently committed, then the rest alphabetically, 200 in all; and `default`, the base
an empty field means (`base:` in `span.yaml`, else `dev`). The placeholder shows `master`
when the kind is `hotfix`.

## House style

`harness/skills` is the writing standard, copied from the skill set so two people produce one author's code. The build prompt lists each skill name, its trigger, and the path of `SKILL.md`. The agent reads that file only when the trigger matches the change. Reference files stay unloaded until the skill itself names one. Bodies are not pasted into the prompt.

A sealed `hold` or `stop` in frame, build, or inspect is stored in `<workspace>/.span/lessons.json`. The next prompt for that same stage includes at most five of those lines. A lesson that fails to prevent the same mistake twice is dropped. The file is local, the lookup does not call a model, and the skill files are not rewritten.

## Quick path

1. Build the binary in this repository.
2. Point `--workspace`, or the **Folder** field, at the project. Quickstart writes `span.yaml` when that file is missing.
3. When the story comes from Jira, save the account in that same folder.
4. Start the span. If it pauses, read the statement, adjust, and resume with `./span resume`.
5. To use the page, run `./span serve` on localhost from this folder.

```bash
cd /path/to/span
go build -o span ./cmd/span
./span run --brief /path/to/project/brief.md --workspace /path/to/project
./span resume <id> --workspace /path/to/project
./span serve --addr 127.0.0.1:8787 --spans /path/to/project/.span/runs
```

`advance` exits 0. `hold` and `stop` exit 1. A startup error (empty brief, invalid `span.yaml`) also exits 1 and prints the reason on stderr.

A finished run prints an identity line and a reason line:

```text
2026-11-18-1534-a1b2 note advance
lead time 4m0s
```

## Before the first run

The workspace is a git repository. Open runs `git push origin HEAD`, so a remote named `origin` is required when open is `auto`.

The agent is a CLI already on `PATH` with Gentle AI detectable:

| CLI | Used when | Gentle AI is detected in |
| --- | --- | --- |
| `cursor-agent` | it is on `PATH` | `~/.cursor/rules/gentle-ai.mdc` |
| `claude` | it is on `PATH` | `~/.claude/CLAUDE.md` contains `Organic Driven Development` |
| `codex` | it is on `PATH` | `~/.codex/AGENTS.md` contains `Organic Driven Development` |

With With `agent: any`, the order is cursor, claude, codex, pi. The first enabled CLI writes. If a second one is enabled, inspect runs on that other CLI. If none qualify, frame holds and no product code is written. The CLI is headless: `cursor-agent` gets `--trust --force`, `claude` gets `--dangerously-skip-permissions`, and `codex exec` gets `--sandbox workspace-write --approve-for-me`. A prompt for permission would have no one to answer it.

Build requires Organic Driven Development. `odd/tasks/<feature>.md` must contain a title (`# ...`), `## Specs`, `## Tasks`, and `## Log`. If that document is missing, the span stops and does not open a change.

## Basic use

### Build

```bash
go build -o span ./cmd/span
go test ./...
```

`go test ./...` does not call an agent and does not deploy. It checks the gates with fakes.

### Project policy

`span.yaml` lives at the workspace root unless you pass `--policy`. `feature` allows letters, digits, `.`, `-`, and `_`. An empty mode means `auto`.

```yaml
feature: payments
agent: any
stages:
  open: auto
  merge: hold
prove:
  focused:
    - go test {files}
  fast:
    - go test ./...
  acceptance:
    - go test ./acceptance
qa:
  report: test-results/junit.xml
repair:
  maxAttempts: 3
promote:
  environments:
    - name: preview
      command: echo preview
      mode: auto
    - name: production
      command: echo production
      mode: hold
manifest:
  build:
    skills: []
    rules:
      - rules.md
    mcp: []
  inspect:
    skills: []
    rules:
      - rules.md
    mcp: []
```

`open: hold` or `merge: hold` pauses before that effect, and only a person releases it: the page shows **Approve merge** (or **Approve open**, or **Approve promote to <env>**) instead of a retry, because a retry reaches the same hold. The approval is bound to the commit the page showed, so a new commit is held again, and it is spent only when the stage advances. Each promote environment has its own `auto` or `hold`. A failed environment stops the ladder. The next environment does not run.

`prove` and `promote` commands are argv words, not a shell line. `go test ./...` is valid. `go test ./... | wc`, redirects, shell quotes, parentheses, backticks, `;`, `|`, and `&` are not executed. The stage holds.

Promote receives the sha in `SPAN_SHA`.

The `rules` field in the yaml is accepted and is not injected by itself. The agent sees the names you list under `manifest`, not the skill body. The harness (Cursor, Claude, or Codex) loads that body on its own.

### Brief

`--brief` is a file path. The whole file is the request. An empty brief does not start.

```bash
./span run --brief brief.md --workspace /path/to/project
```

Flags for `run` and `resume`:

| Flag | Default | Effect |
| --- | --- | --- |
| `--workspace` | `.` | Repository to change |
| `--brief` | none | Required for `run` |
| `--spans` | `<workspace>/.span/runs` | Where state is stored |
| `--policy` | `<workspace>/span.yaml` | Policy file |

`resume` takes the id as an argument, not a flag:

```bash
./span resume 2026-11-18-1534-a1b2 --workspace /path/to/project
```

If you omit `--brief` on resume, Span reads `.span/runs/<id>/brief.md`.

### Stages

The order is fixed: intake, frame, build, prove, qa, inspect, open, merge, promote, note.

| Stage | What it does | When it does not continue |
| --- | --- | --- |
| intake | Stores the brief and its digest | It does not stop on its own |
| frame | Reads the story in its own language and carries what it decides into the standard | `hold` on questions or missing acceptance. `stop` on out of scope. `hold` when there are no claims or no agent |
| build | Runs ODD and commits a dirty tree | `stop` without the feature document. `hold` when the agent or git fails |
| prove | Runs `fast`, then `acceptance` if fast passed | `hold` when the command is not executable or did not start. A non-zero exit returns to build with its output |
| qa | Checks each acceptance line against a test declaration and reads a per-test report | `hold` when the command did not start. A failed gate returns to build with its output. A story with no acceptance lines skips this stage |
| inspect | A separate read of the diff | A blocking finding returns to build. The third block holds |
| open | Looks up the change, pushes, and opens it | `hold` when the mode is `hold`, the push fails, or the create result is unknown |
| merge | Merges when it is not already merged | `hold` when the mode is `hold` or the result cannot be confirmed |
| promote | Walks environments in order | `hold` on a `hold` environment or an unknown result. `stop` when the command finished and failed |
| note | Records lead time from frame | Reached only after promote advances |

`qa` numbers the lines under `Acceptance criteria:` as `AC-1`, `AC-2`, and so on. The build prompt tells the agent to put each id on a test declaration (`func Test`, `test(`, `it(`, or `def test_`), as `AC-1` or `AC1`. A comment is not enough. The command is `qa.command` when set. Otherwise a repo with `playwright.config.*`, or `qa.browser: true`, runs `npx playwright test`. Otherwise Span uses the first `prove.acceptance` command, then the first `prove.fast` command. The result is `.span/runs/<id>/qa-gate.json` with `pass`, `fail`, or `unverified`. This stage does not comment on Jira.

An acceptance id is only `fail` when a report names the test that ran and failed. `qa.report` points at a JUnit XML or a Vitest/Jest JSON report; with no path set, Span looks in `test-results/junit.xml`, `junit.xml`, `test-results/vitest.json`, `test-results/results.json`, `test-results.json`, and `report.json`. A report older than the command is ignored.

Without a report, a zero exit marks the declared ids `pass` and a non-zero exit marks them `unverified`. The process failing is not evidence that a named criterion failed.

QA does not run the fast command a second time. When prove already ran that exact command on this commit in a clean tree, the gate reads that result and only checks the declarations. A dirty tree or a different commit runs it again.

`./span resume` after `Fix and retry` continues at build, not at the stage that held.

A browser repo also tells the agent to test user-visible behavior with `getByRole` and web-first `expect`, and to record a trace on the first retry. `@playwright/mcp --caps=testing` is named as an authoring aid. The gate is the command, not an MCP session.

Span commits. The agent does not. The message is `feat(<feature>): span <id>`, which names the session. The author and committer are **whoever that checkout is configured with**: Span sets no identity, adds no co-author trailer, and never overwrites your `user.name` or `user.email`. `git add` excludes `.span`, so run state stays out of the commit. The default branch is not recreated. The agent is not allowed to push, merge, or deploy.

### Statements

Each stage writes its files before the next stage starts:

```text
.span/runs/<id>/iter/<n>/<stage>.json
.span/runs/<id>/iter/<n>/<stage>.md
.span/runs/<id>/cursor.json
.span/runs/<id>/brief.md
```

The JSON is the statement. It carries the verdict, the previous statement digest, the sha, the change URL, the commands, and the reason. The markdown file is the same record for reading.

The cursor is the resume point: iteration, stage, sha, URL, and any pending effect.

## Advanced use

### Instructions per stage

Instructions do not change the engine contract. They do not authorize push, merge, or deploy, and they do not change a verdict.

For every run of the project:

```text
span/stages/<stage>.md
```

For one run:

```text
.span/runs/<id>/instructions/<stage>.md
```

Both are read when that stage builds its prompt, which is when the stage starts. A stage that is already running does not read a note saved during that pass. Intake stores the digest and does not call an agent, so an intake note does not steer the run. If the note changes before the next time that stage starts, that start sees it.

From `./span serve`, the ledger keeps this control. While a stage is running, choose **Leave a note for a stage**. On hold or stop the control is already open, because resume reads the note when that stage starts again.

### Handoffs

Files are optional. A handoff can be a note, an image, or any other file. Span does not require a format.

Attach them when you load the story. The form field is **Handoff files**. The same call from this folder repeats `--file`:

```bash
./span quickstart --workspace /path/to/project --issue AET-1478 \
  --file ./board.png --file ./notes.md
```

Each selected repo receives a copy in `.span/runs/<id>/handoffs/`. The first frame is told to open those paths. Only the file name is kept, so `notes/board.png` is stored as `board.png`. A story accepts 20 files, and each file can be up to 32MiB. A file without a story or an issue key is `invalid: files need a story`.

You can still copy files into `.span/runs/<id>/handoffs/` while the span exists. The stage already running does not read them. They enter the next frame, with their digest.

### The weight of a story

Not every story earns the same machinery. Span derives a weight from the story itself and
each weight pays for a different path:

| Weight | When | Repairs | Whole suite |
| --- | --- | --- | --- |
| `light` | up to 4 criteria and no design handoff | 2 | not run |
| `standard` | 5 to 15 criteria | 3 | once per commit |
| `deep` | 16 or more criteria, or a design handoff (html, png, pdf, svg, fig) | 6 | once per commit |

The weight is read while the run walks, not frozen at intake: frame can install criteria,
and a brief that grows into twenty of them earns more checks. A prove pass is never left
with nothing to run — if a light story has no declaring files to focus on, the whole suite
decides again.

Override it when the derivation is wrong for a project:

```yaml
run:
  weight: standard
```

`repair.maxAttempts` still wins when it is set: the weight only decides the default.

### The story's own tests

Span runs the tests that belong to the story, not the repository. It already maps every
acceptance criterion to the file that declares it, for the QA gate; that map is the scope.

**You do not have to configure anything.** With `prove.fast: [npm test]` and a story whose
criteria are declared in test files, the scoped band is derived from the wide command:

| Wide command | Scoped to the declaring files |
| --- | --- |
| `npm test`, `pnpm test`, `yarn test`, `npm run <script>` | `<command> -- <files>` |
| `npx vitest run`, `vitest`, `npx jest`, `jest` | `<command> <files>` |
| `pytest`, `python -m pytest`, `npx playwright test` | `<command> <files>` |

When the runner is not in that table (`go test ./...`, `make test`), Span does not invent a
syntax: the wide band keeps deciding, as before.

Override it when the shape is unusual, or when one band is not enough:

```yaml
prove:
  focused:
    - npm test -- {files}
  fast:
    - npm test
```

The rule is one sentence long: **a focused band blocks, the wide suite reports.** When
`focused` is configured, a failure in it returns to build like any attributable failure,
and the wide `fast` list still runs but cannot spend a repair attempt or turn a criterion
red. Its output is kept, its band is recorded as `wide`, and the reason names it: a
failure outside this story's files cannot be repaired by this story's agent.

The wide suite runs **once per commit**. Once it passes on a commit, the next attempt on
that same commit skips it: the bytes did not change, so the answer cannot. A repair that
commits again gets a fresh wide pass.

Without `focused`, the `fast` band decides, exactly as before.

No declaration means no scope. If the story's criteria are not on test declarations yet,
there is nothing to focus on and the wide band decides, which is the honest outcome: the
change has not said what it is supposed to prove.

### Prove only what the change can affect

When neither the criteria nor `prove.focused` give a focused band, and a
[baseline](#failures-the-base-commit-already-had) was recorded, Span derives one from the
branch's diff, `git diff <base>...HEAD`, so a repair reruns the tests the change can reach
instead of the whole suite. It reads the first `fast` command it knows:

| Wide command | Focused on the diff |
| --- | --- |
| `go test [flags] ./...` | `go test [flags] <changed packages and every package that imports them>` |
| `vitest`, `vitest run`, `npx`/`pnpm`/`pnpm exec`/`yarn vitest` | `<runner> related --run [flags] <changed files>` |
| `jest`, `npx`/`pnpm`/`yarn jest` | `<command> --findRelatedTests <changed files>` |

For Go, importers are found with `go list` and include test-only imports, transitively. Flags
are kept; `-tags` is passed to `go list` too. A failure keeps the same fingerprint when the
package list changes between repairs, so the same failure twice still holds as `no progress`.

Only imports are followed. A test that runs `go run ../cmd/app`, reads a file at runtime, or
starts another process depends on code the narrowing cannot see. That is why a diff focus
needs a recorded baseline and why the wide band still runs and blocks.

It is conservative: whenever it cannot be sure, there is no diff focus and the wide band
decides exactly as before. That happens when the diff deletes, renames, copies, or retypes a
path; when it touches anything besides source files and Markdown outside every Go package
directory (`go.mod`, `go.sum`, `testdata`, a `.md` a test could embed, `package.json`,
lockfiles, `vite.config.*`, `vitest.config.*`, `vitest.workspace.*` and other config or setup
files, `__mocks__`, styles, assets);
when it has more than 200 files; when the Go command names packages other than `./...`,
passes `-args`, `-mod`, `-modfile` or `-C`, or a flag Span does not know takes a value; when a JavaScript command has positional words; when
`go list` fails; when every package would run anyway; when no baseline was recorded
(`baseline: false`, or a checkout with uncommitted changes); and for any other runner, `npm test`
and `pnpm test` scripts included. Python is not supported. The transcript says which way it
went: `prove: focused on the diff: ...` or `prove: not focused on the diff: <reason>`.

The wide band still runs after the focused one passes, once per commit. A wide failure the
base commit did not have blocks and spends a repair like a focused one, and a wide command
that cannot start holds: the diff focus may have missed what the change broke.

Turn it off with:

```yaml
prove:
  diff: false
```

### Tests and the repair loop

`fast` runs first. If a `fast` command fails, `acceptance` does not start.

A non-zero exit attributed to this change does not discard the span. It returns to build with the failing command, its exit code, the last lines it printed, and the path to the full log. The build prompt carries that packet, so the agent fixes the failure instead of reading the feature again. `prove` and `qa` use the same edge.

A command that never started does not consume an attempt. It holds, and `./span resume` repeats the band.

Two stops run before the cap:

- The same failure twice. The fingerprint of a failure is the command plus the lines that name what broke. If it comes back unchanged on the next attempt, the build did not fix it, and another run would spend the same money for the same result. The span holds with `no progress`.
- The same command passing and then failing on one commit. The result is not attributable to the change, so the span holds as `inconclusive` and the criteria stay unverified. `Re-verify` runs it again without touching the code.

`repair.maxAttempts` is how many times a failure may send the span back to build, from 1 to 8, default 3. The next failure after that holds and keeps the reports. A restart from a stage clears the count.

Inspect is separate. A blocking finding archives build and prove for that iteration and returns to build. Two returns can correct the change. The third holds.

### Failures the base commit already had

A red test in an unrelated module is not this story's to fix. Before the first build, Span
runs the `fast`, `acceptance`, and `qa.command` lines this story would use once on the base
commit and writes what failed to `.span/runs/<id>/baseline.json`. It is skipped when the
checkout has uncommitted changes, because those are not the base commit.

After the build, a failing command is excused only when it failed on the base commit too and
every failure line it prints now was already there, at least as many times. Only colors, a
parenthesised duration such as `(0.12s)`, and the timing column of a `go test` package line
are ignored; every other character counts, values inside messages included, so one new
failing test is a new failure and blocks as before. For `go test`, with or without `-v`,
every line a failing test printed is compared, tied to its test and package; what a passing
test logs is ignored. An output cut at the 1 MiB capture limit, on either side, and a
package that reports `[build failed]` or `[setup failed]`, are never excused.

What is and is not matched: only an identical failure is excused. A failing test that prints
timings, pointers, temporary directories, random seeds, or map order differs on every run,
so it is not excused even when the code did not change, and a panic stack never is. That is
the safe direction: the failure blocks, as it did before the baseline existed.

An excused command is never hidden: it stays on the statement marked `preexisting`, the
reason says how many were not counted, and it does not spend a repair or reach the build
prompt. In QA, an excused exit is judged as exit 0: without a per-test report the criteria
pass on it, and with one each criterion still decides.

The cost is one extra run of those commands per run. Turn it off with:

```yaml
prove:
  baseline: false
```

### Choosing the agent

Span drives four CLIs. All four run headless and write: nobody is present to approve
a prompt, so the flags are fixed per CLI.

| Agent | `agent:` | Binary | Called as | Ready when |
| --- | --- | --- | --- | --- |
| Cursor | `cursor` | `cursor-agent` | `-p --trust --force` | `~/.cursor/rules/gentle-ai.mdc` exists |
| Claude | `claude` | `claude` | `-p --dangerously-skip-permissions` | `~/.claude/CLAUDE.md` says Organic Driven Development |
| Codex | `codex` | `codex` | `exec --sandbox workspace-write --approve-for-me` | `~/.codex/AGENTS.md` says Organic Driven Development |
| Pi | `pi` | `pi` | `--print --approve` | `~/.pi/agent/npm/node_modules/gentle-pi` is installed |

`agent: any` or `auto` picks the first ready CLI in that order, and uses a second ready
one for inspect, so the review is not the same model that wrote the change. The binary
name is accepted too, so an older `agent: cursor-agent` keeps working.

Choose it on the page: the desk has an **Agent** picker under the folder. It lists all
four with the reason each one cannot run, and saving rewrites the one `agent:` line in
`span.yaml` without touching a comment. The picker needs the file to exist; a folder
without one is told to run `./span quickstart` once.

`GET /agent?path=<folder>` returns the same list, and `POST /agent` with `project` and
`agent` writes the choice.

### Choosing the agent per stage, and pausing for it

The project picks a default agent. One run can override it per stage without touching
`span.yaml`: on a started span, the ledger tab has **Agent for a stage of this run** with
frame, build, and inspect. The choice lives in `.span/runs/<id>/agents.json` and applies
the next time that stage starts.

Resolution order for a stage: the run's choice, then `agent:` from the policy, then the
first ready CLI. An empty choice, or `Policy default`, clears the exception. Inspect keeps
using a different CLI from build when nothing was chosen, so the review is not the model
that wrote the change.

Only frame, build, and inspect call a model. prove, qa, open, merge, and promote run
commands, so they cannot carry a choice and a request for them is refused.

**Pause after this stage** asks the run to hold at the next stage boundary. The stage that
is already running finishes; the run then holds with `paused before <stage>` and the
request is consumed. Resume continues from that stage, so the order is: pause, pick the
agent, resume.

The pause does not kill a stage that is mid-write, because that would leave a half written
statement. When you need the running stage to stop now, the run tolerates a killed
process on resume, but Span does not offer a button for it.

The page and the runner are different processes, and `cursor.json` belongs to the runner:
a cursor written by the page is erased by the runner's next save. That is why both
requests are files beside the cursor, the same way instructions and answers already work.

`POST /spans/{id}/agent` with `stage` and `agent`, and `POST /spans/{id}/pause`, back the
two controls. The snapshot the page reads carries `stageAgents`.

### When an agent runs out of quota or credentials

A stage that calls an agent can fail for a reason the project cannot fix. Span reads the
CLI's own output and names it, so the reason says what to do instead of repeating
`exit status 1`.

Two kinds, because the answer differs:

| Kind | What the CLI printed | What to do |
| --- | --- | --- |
| `quota` | a usage or rate limit, a spent credit balance | resume after the limit resets, or point `agent:` at another CLI |
| `credentials` | no API key, an invalid key, a login that expired | log in to that CLI, then resume |

The match is a curated list of phrases, not a status code, and it runs on text the CLI
already produced: no model, no extra process, no network. A build failure that happens
to print `429` stays an ordinary failure.

The run holds, it does not stop. `cursor.json` keeps the block, and the page shows
which CLI needs attention:

```json
"agentBlock": { "kind": "quota", "agent": "claude", "detail": "Claude AI usage limit reached.", "time": "..." }
```

The block clears when a later stage advances.

### Parallel runs get their own worktree

Two stories in one repository must not share a working tree. A run that walks a branch is
checked out in its own **git worktree**, so the checkout you are using is never switched and
two runs never see each other's files.

```bash
./span run --workspace /path/to/repo --brief notes.md --branch feat/sal-3 --worktree
./span quickstart --workspace /path/to/repo --issue SAL-3   # isolates when it derives a branch
```

The tree lives at `<repo>/.span/worktrees/<branch>`, with `/` in the branch name turned into
`-`. The state directory and `span.yaml` stay in the repository you pointed at, so the desk
keeps one list of runs and the policy is not copied into every tree.

These are the only worktree commands Span runs. Nothing is inferred:

| Command | When |
| --- | --- |
| `git worktree add -- <path> <branch>` | the branch already exists |
| `git worktree add --no-track -b <branch> -- <path> <base>` | the branch is new, from its [base](#where-a-branch-starts-and-where-its-pr-goes) (`origin/<base>` when origin has it) |
| `git fetch --quiet origin -- +refs/heads/<base>:refs/remotes/origin/<base>` | before every new branch and every base check; best effort |
| `git worktree list --porcelain` | deciding whether this branch already has a tree |
| `git worktree remove --force <path>` | dropping an isolated tree; the branch survives |
| `git worktree prune` | forgetting trees whose directory is gone |

A run in a worktree tells the agent so, in the prompt, with the rules spelled out: the branch
is already checked out; do not run `git checkout`, `git switch`, `git worktree add`, or
`git stash`; read `git worktree list` if you need to see the other trees; Span commits.

**What the lock protects.** A run holds the tree it edits, not the repository. From the page,
the first story in a free checkout edits it in place and holds it. A second story launched
while that checkout is held is not queued: it gets its own worktree and starts at once, and
nothing is written into the busy checkout (no `span.yaml`, no brief). Stories in separate
worktrees never wait for each other; only two runs that share one tree are serialized. A busy
checkout that was never configured, or a launch with no story, still refuses, because there
is nothing safe to isolate with. So does launching a story whose branch already belongs to
another run or exists in git: two runs on one branch would fight over its checkout. Resuming,
answering, or reloading an isolated run always lands on its own worktree and reads the policy
from the project.

**Dependencies in a fresh worktree.** A new worktree has the code but not `node_modules`.
Set a command in `span.yaml` and Span runs it once in the run's own worktree, before the
baseline and the first build; it never runs in your own checkout:

```yaml
prove:
  prepare:
    - pnpm install --frozen-lockfile
```

If `prepare` fails, or the tree has a `package.json` and no `node_modules` and no `prepare`,
no baseline is recorded and the transcript says why. Recording one there would mark the
environment as broken on the base commit and then excuse the same failures after the build.

### Restarting from a stage

When what the branch produced is not what the handoff asked for, a run can start again at
any stage that does work (everything except intake). The ledger tab has **Restart from a
stage**: pick the stage, write the instruction that stage must read, and confirm.

What a restart does, and why:

- `cursor.stage` moves to that stage and `iteration` goes up by one, so the attempt so far
  stays on disk under its own iteration instead of being overwritten.
- the repair ledger, `wideSHA`, the agent block, and any open questions are cleared: a
  person starting over deliberately is not the automatic loop the cap exists to stop, so
  the restarted streak gets its own budget.
- the note is written to `.span/runs/<id>/instructions/<stage>.md`, which is the file that
  stage reads when it starts. It is the same channel as **Leave a note for a stage**.

A restart refuses to start while a stage is running: stop it first, or the two writers
would race. `POST /spans/{id}/rewind` with `stage` and an optional `note` backs the control.

### Stopping what is running

**Stop now** cancels the stage in flight. The engine kills the command or the agent, the
loop fails, and the run comes back **held** with `the run stopped: …` — never idle, so the
page always has something to offer. The working tree may be left mid-edit, which is why the
next resume runs that stage again from its start.

`Stop now` and **Pause after this stage** are different on purpose: pause waits for a clean
stage boundary and loses nothing; stop is immediate and assumes the stage will be repeated.
`POST /spans/{id}/stop` backs the control and answers `nothing is running this span` when
there is no flight.

### Resume

`./span resume <id>` reads the cursor.

| Saved state | Resume behavior |
| --- | --- |
| `hold` | Continues at the paused stage |
| `stop` | Stays stopped. The agent is not called |
| `done` | Stays finished |
| Stage whose statement is `advance` | That stage is not repeated |
| Open when the change already exists | Does not open another one and does not push again |
| Promote with a receipt for that environment and sha | Does not run that command again |
| Promote started without a receipt | `hold`. The result is unknown and is not retried blind |

A successful environment receipt is `.span/runs/<id>/receipts/<environment>-<sha>`.

If the process dies, the agent drops, or the network cuts, the last cursor on disk is the start point. Intake is not repeated.

### Changes and GitHub

When `gh` is on `PATH`, open lists the branch's pull requests with `gh pr list --head=<branch> --json url,baseRefName`, creates one with `gh pr create --base=<base> --head=<branch>`, and merge uses `gh pr merge --merge`. The base is the run's, see [Where a branch starts and where its PR goes](#where-a-branch-starts-and-where-its-pr-goes). The workspace must be that repository's checkout, with commits already pushed to `origin`.

When `gh` is absent, the change is recorded as `local:<id>:<branch>` in `.span/runs/<id>/change.url`. A local merge writes `merged` beside it. That does not publish to a forge.

### Read statements and give instructions

`./span serve` has no `--workspace`. The desk at `/` is the quickstart. Its **Folder** field is the project for repos, Jira, and the run. `--spans` is the runs directory the page reads. The default is `.span/runs` relative to the directory where you launched the command, not the Folder field. Pass an absolute path when those statements live somewhere else. A run started from the page is stored in that directory, so the id the page receives is the id the page can open.

```bash
./span serve --addr 127.0.0.1:8787 --spans /path/to/project/.span/runs
```

Localhost does not need a token. Any other address without `SPAN_TOKEN` refuses to start.

```bash
SPAN_TOKEN=a-long-secret ./span serve --addr 0.0.0.0:8787 --spans /path/to/project/.span/runs
```

Every HTTP request sends `Authorization: Bearer a-long-secret`. A browser cannot set that header on a WebSocket, so the page sends the same secret as the subprotocol `span.` plus the raw URL-safe base64 of the token. On localhost, leave **Serve token** empty. A value left in the tab used to make the browser drop the live socket: the page offered a subprotocol and the server did not name one back. With no `SPAN_TOKEN`, the server accepts that offered protocol and the socket opens. When `SPAN_TOKEN` is set, type it there and choose **Use token** before **Configure and run**. The token stays in this browser tab.

### The page

Two routes, one process. There is no second port.

| Route | What it shows |
| --- | --- |
| `/` | The desk: folder, find repos, repo checkboxes, the Jira session for that folder, load one story, my stories, close session, kind, story text, handoff files, configure and run, and the span list |
| `/spans/<id>` | One run, in three readings |

**Configure and run** answers `202` with `{"id","stage"}` once `cursor.json` exists, then the run continues after the response. Closing the page does not cancel it. `./span quickstart` still waits and prints text.

The span page keeps one rail of the ten stages. The current stage is the mark. Under it, one sentence: the sealed reason. `hold` and `stop` stay on that rail, including a Jira block comment. A timeline band above the status line records each sealed decision and local action; a transcript flush is one line, and the agent text stays on Implementation. Every 2 seconds the page reviews the agent in this process and keeps one timeline line. If the stage looks running and no worker is left, **Relaunch** starts it again.

The three readings are the same run:

- **Ledger.** Each sealed statement: stage, verdict, iteration, reason, parent digest, commands, git SHA, and change URL. The instruction control stays here, closed while a stage is running. **Leave a note for a stage** opens it. The sentence under the control says when the selected stage reads the note.
- **Implementation.** The files changed from `base` to `HEAD`, the patch, and the agent transcript. The patch is cut at 256 KiB and the page says when that happened. While `build` is open, new transcript lines arrive on the socket.
- **QA.** Each acceptance line as `AC-1`, `AC-2`, the test declaration file and line, and `pass`, `fail`, or `unverified`. The command and its exit code sit on that same row. A story with no acceptance lines says the gate was skipped. A missing declaration is `unverified` and names no file.

A refresh replays the run from disk. The socket's first message is the span object plus the last 1000 transcript lines. Later messages are `cursor`, `statement`, `transcript`, and `gate`. Transcript lines flush about every 500 ms. A dead socket does not fail the run. The statements on disk stay the record.

`GET /spans/<id>/live` is that socket. Dev (`npm run dev` in `web/`) proxies HTTP and the socket to `127.0.0.1:8787`. The built page is served by `./span serve` from the same origin.

#### When the connection drops

- **It reconnects.** A closed socket (server restart, laptop sleep, flaky wifi) is retried
  after 0.5 s, then 1 s, 2 s, and so on up to 10 s, with a little jitter. The backoff only
  starts over after a connection stayed up and in order for 5 s. The timeline notes
  `socket closed` and when the next try starts; a long outage is noted once.
- **It resyncs from the replay.** Every connection starts with a replay, so a reconnect is
  a full resync, not a patch on a stale page. When the server cannot read the run, the
  replay still comes, without a span but with the transcript tail: the page takes the
  tail, keeps the rest of what it shows, reloads the run with `GET /spans/<id>`, and only
  lowers its numbering when the replay's `seq` is below the last one it saw (a server
  restart).
- **Frames are numbered.** Each live frame carries `seq`, one counter per run, and the
  replay names the `seq` the stream resumes after. A frame at or below the last one seen is
  dropped; a jump means frames were lost, so the page closes the socket and resyncs. The
  first gap resyncs at once; another gap within 5 s waits for the backoff.
- **Transcript lines are not repeated.** Transcript frames carry `start` and `offset`,
  the size of `transcript.log` before and after their lines, and the replay carries the
  size its tail was read up to. Flushes are published in the order they were taken. A
  flush can straddle a connect (one line before the replay, one after): the server cuts,
  per connection, the lines its replay tail already holds and still sends the frame, so
  the numbering has no hole. The page also drops a flush whose offset is not past the
  replay's.
- **A failed write stays out of the live view.** When the disk refuses part of a write,
  only the lines written whole are sent live; the agent keeps running and the log
  reports the error when it closes.
- **A slow client is cut, not starved.** When a page falls 32 frames behind, the server
  closes its socket instead of silently dropping frames (the final `done` cursor
  included), and a write that a client does not read within 10 s ends the socket too. The
  page reconnects and replays.
- **The serve token stays out of the socket.** The replay and every live frame have the
  token replaced inside their string values, never in the encoded bytes, so JSON keys stay
  intact and a token with a quote, a backslash, `<` or `&` is still found. The protocol
  fields `type`, `stage` and `verdict` are never rewritten. A token shorter
  than 8 characters still guards the socket but is not redacted, because it is too likely
  to be an ordinary word. `SPAN_TOKEN` is also one of the secrets the transcript redacts.
- **One broken component does not blank the app.** A render fault shows the error with
  **Try again** (render once more) and **Reload page**; moving to another route clears it.
- **One failing endpoint does not blank the page.** The span page reads the run, the
  implementation and the acceptance on their own: a missing diff before the first commit
  leaves that tab empty, and only an unreadable run is an error. An older acceptance reply
  never overwrites a newer one. **Find repos** shows the repositories even when Jira
  fails, and the Jira session even when the folder fails.

### Routes

| Method | Path | Effect |
| --- | --- | --- |
| GET | `/` | The desk |
| GET | `/repos?path=` | Checkouts in that folder |
| GET | `/jira?path=` | Whether that folder is logged in. The token is omitted |
| POST | `/jira/account` | Fields `project`, `url`, `email`, `token`. Saves `<project>/.span/jira.json` |
| POST | `/jira/logout` | Field `project`. Removes that folder's login |
| GET | `/jira/issues/<key>?path=` | Brief for that key, using that folder's login |
| GET | `/jira/search?path=&jql=` | Up to 20 issues. Empty `jql` lists issues assigned to the current user |
| POST | `/quickstart` | Fields `workspace`, `repo`, `kind`, `issue`, `story`, and `file`. `202` and `{"id","stage"}` when a run starts. Configuration text stays `200` |
| GET | `/spans` | `Accept: application/json` returns `{"spans":[...]}`. Any other accept is the desk |
| GET | `/spans/<id>` | `Accept: application/json` returns the cursor, statements, acceptance lines, and gate. Any other accept is the span page |
| GET | `/spans/<id>/implementation` | `git diff <base>...<head>` stat and a patch capped at 256 KiB |
| GET | `/spans/<id>/acceptance` | Each acceptance line, its declaration, and the gate result |
| GET | `/spans/<id>/live` | WebSocket. The first frame replays the run. Later frames wake the page |
| GET | `/spans/<id>/agent` | Whether this process still has a worker for that run |
| GET | `/agents` | The workers this process is running |
| POST | `/spans/<id>/relaunch` | Starts a run again when the cursor is not paused, stopped, or waiting, and no worker is left. `202` |
| POST | `/spans/<id>/answer` | JSON `{"answers":[{"question","body"}]}`. `202` and frame runs again |
| POST | `/spans/<id>/story` | Reads the Jira issue again. `202` and frame runs again |
| POST | `/spans/<id>/resume` | Continues a hold that is not waiting for an answer |
| POST | `/spans/<id>/instructions` | Form fields `stage` and `body` |
| POST | `/spans/<id>/instructions/<stage>` | `text/plain` body or form field `body` |

A failed call returns JSON `{"error","message"}`. `unauthorized` is 401, `not_found` is 404, `unreachable` is 502, and the other codes are 400.

Answer a waiting run, or read the Jira story again:

```bash
curl -sS -X POST -H 'content-type: application/json' \
  -d '{"answers":[{"question":"Which permission atom?","body":"Pago.Ver is new."}]}' \
  http://127.0.0.1:8787/spans/<id>/answer

curl -sS -X POST http://127.0.0.1:8787/spans/<id>/story
```

Both answer `202` and run frame again on that same span. A stage instruction is still only a note for the next time that stage starts:

```bash
curl -sS -X POST --data-binary @note.md \
  -H 'content-type: text/plain' \
  http://127.0.0.1:8787/spans/<id>/instructions/frame
```

The binary and the agents live on the same machine. `serve` reads and writes files. It does not drive a Claude process on another computer.

### Manifest

`manifest.<stage>` lists skill, rule, and MCP names. Those names go into the prompt, and their digest goes into the statement, so the run can be repeated. Span does not download skills, does not run installers, and does not paste a skill body into the prompt.

### What this version does not do

- It does not deploy just because the yaml names an environment. It runs the command you wrote, and `hold` on that environment pauses before the command.
- It does not resume a real `stop`, such as out of scope or a missing feature document. A run waiting on questions continues from **Answer and continue** or **Read the story again**.
- It does not sign statements. The chain is the digest of the previous JSON.
- It does not run a shell. If a check needs a pipe, put it in a script and declare the script path.
- `go test ./...` in this repository does not call a live agent or a live `gh`.

## Run checklist

- [ ] `span.yaml` has a valid `feature`, prove commands, and environments.
- [ ] The workspace is a git repo with `origin` when open is `auto`.
- [ ] When the story comes from Jira, that same folder has its own `.span/jira.json`.
- [ ] The chosen CLI is on `PATH` and Gentle AI is detected in its file.
- [ ] The brief states the request without invented requirements.
- [ ] After `hold`, you read `.span/runs/<id>/iter/<n>/<stage>.md` before `./span resume`.
- [ ] After `stop`, you do not expect resume to continue.
