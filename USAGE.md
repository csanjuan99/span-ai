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

`status` prints `configured true`, the project folder, the site, and the email. It does not print the token. `--json` on `status`, `configure` and `logout` prints the same object as `GET /jira?path=<project>` (`configured`, `baseUrl`, `email`, `auth`, never the token). `issue --json` prints the story object `GET /jira/issues/{key}` returns, and `search --json` prints `{"issues":[...]}`. A missing account still exits `1`; `GET /jira` answers `200` with `configured: false`. Flags may come before or after the issue key.

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

## Shaping a story

A story can be built inside Span instead of in another tool. Open **Shape a story** on the desk (`/shaping`), give the idea and, in a folder of checkouts, the repo it touches.

Each turn the agent reads the idea, your answers, the product context and the repository, and returns an example map: rules, examples under each rule, and at most three new questions. Every rule and example says where it came from: `prompt`, `answer`, `code` (the path it read) or `context`. A question shows what it blocks and the assumption Span would make. Answer it, or accept the assumption; an accepted assumption becomes a `Supuesto` in the story, never a rule. The method is Example Mapping, described in `harness/skills/user-story/references/interview.md`.

The interview must not change the repository. Span compares `git status`, the content of every listed file, where `HEAD` points and every branch and tag before and after each turn, so a commit, a checkout or a new tag counts too. A difference fails the turn with `shape_repo_changed` naming what changed. Span does not undo it. Files git ignores and files outside the repo are not compared. The repo of a session must be the folder or a checkout inside it (`shape_repo_outside`).

- **Product context.** The interview reads `span/context/product.md` in the folder when it exists: personas, glossary, known surfaces. The agent can propose additions. A note is written to the file only when you accept it; skipped notes are dropped, can be brought back until the session is sealed, and a note already written cannot be skipped. A preview shows the exact lines accepting would add.
- **Split.** When the map holds two outcomes a person could receive separately, the session shows the split and cannot be sealed. **Split** replaces it with one linked session per story (each starts from a copy of the map, so answered questions are not asked again, and drafts its own story on its first turn). **Keep one story** records the choice and runs the turn again so the agent writes a single story; the split is not proposed again.
- **Seal.** The story can be sealed only when no question is open (`shape_not_ready` lists the open ones). Sealing keeps the markdown, its sha256 digest and the time (`sealedAt`), and the session does not change after that. **New session from this story** (`fork`) starts a new session from a sealed one: the story and the answers carry over, assumptions reopen, and the sealed session stays.
- **Jira.** A sealed story can be created in Jira, and only in the backlog of the project's Kanban board. Give the project key, and the board id when the project has more than one Kanban board. The page shows a preview first and creates only after you confirm. The issue is a Story, unassigned, labelled `span`. Span reads its status back and moves it into a backlog status when it landed elsewhere. It never puts it in To Do or In Progress. The key is stored in the session, so asking again creates nothing.
- **Run.** **Start run** on a sealed session starts a run with the sealed story as the brief and the Jira key, when there is one, as the issue.

Sessions live in `<workspace>/.span/shapes/<id>/session.json`.

| Message | What it means |
| --- | --- |
| `shape_not_ready` | A question is still open. Answer it or accept its assumption |
| `shape_sealed` | The session is sealed and does not change. Start another one |
| `shape_not_sealed` | Jira and runs need a sealed story |
| `shape_repo_changed` | The agent changed the repository during a turn. The paths are listed |
| `shape_not_git` | The repo of the session is not a git checkout |
| `shape_repo_outside` | The repo named for the session is outside the folder |
| `confirm_required` | Creating in Jira was asked without confirming the preview |
| `jira_backlog_disabled` | The board's Backlog column has no status, so the backlog is off. Turn it on in the board's column settings, or nothing is created |
| `jira_board_ambiguous` | The project has more than one Kanban board. The message lists them; pass the board id |
| `jira_not_in_backlog` | The issue was created but no transition reaches a backlog status. The key is stored and shown |
| `shape_context_written` | The note is already in the product context, so it cannot be skipped |
| `shape_no_split` | The session has no split to accept or keep, or it was already split |
| `shape_source_not_found` | The `path:line` reference is not a readable file of the session's repository |

HTTP: `POST /shapes`, `GET /shapes`, `GET /shapes/{id}`, `POST /shapes/{id}/answers`, `POST /shapes/{id}/context`, `POST /shapes/{id}/seal`, `GET /shapes/{id}/jira/preview`, `POST /shapes/{id}/jira`. Each takes the `workspace`; a relative one is read from the folder the server runs in, as on the desk.

#### Listing, resuming and tidying sessions

Every session answers with the stored fields and these derived ones: `ref` (`SES-` and the tail of the id), `title` (the story heading, else the first line of the idea), `openQuestions`, `jiraKey` (the issue created for the story, else the one it started from) and `updatedAt` (the last change; for a session saved before that was recorded, its last turn or the time in its id). New stored fields: `archived`, `sourceIssue` and `failure`.

| Action | HTTP | Command |
| --- | --- | --- |
| List and filter | `GET /shapes?workspace=&status=&q=&repo=&agent=&sort=&archived=` | `./span shape list [--status s] [--query q] [--repo r] [--agent a] [--sort activity\|created] [--archived include\|only] [--json]` |
| One session, with its failure | `GET /shapes/{id}` | `./span shape show <id> [--json]` |
| Start | `POST /shapes` `{workspace, idea, repo?, agent?, model?, issue?}` | `./span shape start (--idea text \| --idea-file f \| --issue KEY) [--repo r] [--agent a] [--model m] [--json]` |
| Resume a stopped interview | `POST /shapes/{id}/resume` `{workspace, agent?, model?}` | `./span shape resume <id> [--agent a] [--model m] [--json]` |
| Change agent and model, no turn | `POST /shapes/{id}/agent` `{workspace, agent, model?}` | `./span shape agent <id> --agent a [--model m]` |
| Archive / unarchive | `POST /shapes/{id}/archive` `{workspace, archived?}` (default `true`) | `./span shape archive <id> [--undo]` |
| Delete | `DELETE /shapes/{id}?workspace=` | `./span shape delete <id> --yes` |

### Revisions, compare and merge

Every saved change to a story is a revision (`r1`, `r2`, ...): an answer, an accepted assumption, a context note, a split, a seal. They live in `<session dir>/revisions.jsonl`. Nothing is ever deleted: a restore adds a new revision with the old content. A sealed session can be compared but not restored; start a new session from a revision instead.

| Action | HTTP | CLI |
| --- | --- | --- |
| List revisions | `GET /shapes/{id}/revisions?workspace=` | `./span shape log <id> [--json]` |
| Compare two | `GET /shapes/{id}/diff?workspace=&from=r3&to=r4` | `./span shape diff <id> r3 r4 [--format inline\|side] [--json]` |
| Restore | `POST /shapes/{id}/revisions/{r}/restore` `{workspace}` | `./span shape restore <id> r3 [--json]` |
| New session from a revision | `POST /shapes/{id}/revisions/{r}/fork` `{workspace}` | `./span shape restore <id> r3 --new [--json]` |
| Compare 2 to 5 sessions of one repo | `GET /shapes/compare?workspace=&ids=a,b,c` | `./span shape compare <id>... [--json]` |
| Keep two apart | `POST /shapes/compare/dismiss` `{workspace, ids}` | `./span shape compare <id>... --dismiss` |
| Merge | `POST /shapes/merge` `{workspace, into, from[], against[], resolve{C1: id}}` | `./span shape merge <into> <from>... --yes [--against ids] [--resolve C1=id,...]` |

Matching between sessions is by shared words, so every figure of a comparison is an estimate. A merge needs every conflict decided (`shape_conflicts_open`, exit 4) and never changes a sealed session (`shape_sealed`); the merged-from sessions stay readable and link to the target (`mergedInto`). Unknown revision: `shape_revision_not_found` (404, exit 3).

#### Board, history and transfer: the same code for the page and the command line

| What | HTTP | Command |
| --- | --- | --- |
| Sessions in a column per status | `GET /shapes?group=status&workspace=&repo=&archived=&limit=` | `./span shape board [--repo r] [--archived include\|only] [--limit n] [--json]` |
| What the agents did lately | `GET /shapes/activity?workspace=&who=&since=&limit=` (`follow=1` streams server-sent events) | `./span shape activity [--who agents\|you\|everyone] [--since t] [--limit n] [--follow] [--json]` |
| One action on several sessions | `POST /shapes/bulk` `{workspace, action, ids[], dryRun?}`, action `assume`, `seal`, `archive` or `unarchive` | `./span shape bulk <action> <id\|SES-ref>... [--ready] [--dry-run] [--yes] [--json]` |
| Steps of a session | `GET /shapes/{id}/history?workspace=&actor=you\|agents&kind=answers\|assumptions\|context\|split` | `./span shape history <id> [--actor a] [--kind k] [--json]` |
| What undoing a step changes | `GET /shapes/{id}/history/{h}?workspace=` | `./span shape undo <id> <h> --plan [--json]` |
| Undo a step | `POST /shapes/{id}/history/{h}/undo` `{workspace, cascade?}` | `./span shape undo <id> <h> [--cascade] --yes [--json]` |
| Start from Jira or a markdown file | `POST /shapes/import` `{workspace, jira \| markdown+name, repo?, agent?, model?, dryRun?, run?}` | `./span shape import (--jira KEY \| --file story.md) [--dry-run] [--no-run] [--json]` |
| Export a sealed story | `GET /shapes/{id}/export?workspace=&format=md\|json&sources=1&assumptions=1&digest=1` | `./span shape export <id> [--format md\|json] [--sources] [--assumptions] [--digest] [--out file]` |
| Read-only links | `POST /shapes/{id}/shares` `{workspace, days?}`, `GET /shapes/{id}/shares`, `DELETE /shapes/{id}/shares/{token}`, `GET /shapes/{id}/shared/{token}` | `./span shape share <id> [--days n] \| --list \| --revoke TOKEN \| --open TOKEN` |

- `bulk` runs each session on its own and answers one item per session (`ok`, `skipped` or `failed` with its code); one failure does not stop the others. At most 50 sessions. Without `--yes` the command only says what would change.
- History is read from the revisions of the session: each step has who did it (`you` or an agent), what it gives and what it needs. Undo adds a step; it never rewrites the log. A step others depend on is refused (`shape_has_dependents`, 409) unless `cascade` is true; a sealed story never changes (`shape_sealed`). Undoing an accepted context note removes it from the context file, and refuses if the file was edited since (`shape_context_changed`). Accepting a split (`split_accept`) and creating the story in Jira (`jira`) are steps of the history too, with `undoable: false` and the reason in `why`: the smaller stories already exist as their own sessions, and the issue already exists in Jira, which Span never deletes. Unknown step: `shape_step_not_found` (404, exit 3).
- Importing from Jira only reads the story; nothing is written to Jira. A markdown file becomes rules (headings), examples (bullets) and notes, each with its `file:line` as source (`source.kind` `import`); at most 256 KB.
- Export needs a sealed story (`shape_not_sealed`). A share link is read-only, lives 7 days by default (90 at most) and can be revoked (`shape_share_revoked`, 410; expired `shape_share_expired`, 410). It does not skip the server's access token.

- `status` is `interviewing`, `ready`, `split` or `sealed`. `q` matches the idea, title, id, ref, repo and Jira key, ignoring case. `agent=auto` matches a session that names no agent. `sort` is `activity` (default) or `created`, newest first. Archived sessions are hidden unless `archived=include` or `archived=only`. An unknown value answers `400 invalid`.
- `issue` starts from a Jira story: the issue is read with the Jira account saved for the workspace, its brief becomes the idea, and `idea`, when also given, is added as more context. The key is kept as `sourceIssue`. A missing issue starts nothing.
- When a turn after the first fails, the answers and the map so far are kept and `failure` is set: `{at, agent, kind, message}`, with `kind` `usage_limit`, `repo_changed` or `error`. It stays until a turn succeeds. `resume` runs the next turn from the saved map, on another agent or model if given (they stay on the session even if the turn fails again), and clears `failure`. A first turn that fails still leaves nothing on disk.
- A sealed session cannot be deleted (`409 shape_sealed`); archive it instead. Archiving never changes its story or digest.
- Exit codes of `span shape`: `0` done, `1` failed, `2` usage, `3` no such session, `4` refused because of the session's state (sealed, not ready, repository changed), `5` the interview agent failed (the failure is saved on the session). With `--json` an error prints `{"error": code, "message": text}` on stdout.

#### Working a session: the page and the command line are two clients

Every action of the session page is a route and a `./span shape` verb that call the same Go function in `internal/shape`. Flags go before the id. The same exit codes apply; `shape_context_written`, `shape_no_split`, `confirm_required` and the `jira_*` refusals exit `4`, `shape_source_not_found` exits `3`.

| Action | HTTP | Command |
| --- | --- | --- |
| Answer, or use the assumption | `POST /shapes/{id}/answers` `{workspace, answers:[{question,text}], assume:[ids]}` | `./span shape answer [--answer Q1=text]... [--assume Q1,Q2] <id>` |
| Answer instead of an assumption | the same route, with the assumed question's id in `answers` | the same verb |
| Accept, skip or restore context notes | `POST /shapes/{id}/context` `{workspace, accept?, skip?, restore?}` | `./span shape context [--accept P1,P2] [--skip P3] [--restore P3] <id>` |
| Preview what accepting writes | `GET /shapes/{id}/context/preview?workspace=&accept=P1,P2` | `./span shape context --preview --accept P1,P2 <id>` |
| Accept the split, or keep one story | `POST /shapes/{id}/split` `{workspace, decision: "split"\|"keep"}` | `./span shape split (--split \| --keep) <id>` |
| Seal | `POST /shapes/{id}/seal` | `./span shape seal <id>` |
| Code behind a source | `GET /shapes/{id}/source?workspace=&ref=path:line` | `./span shape source --ref path:line <id>` |
| Preview the Jira issue | `GET /shapes/{id}/jira/preview?workspace=&project=&board=` | `./span shape preview --project KEY [--board N] <id>` |
| Create in Jira | `POST /shapes/{id}/jira` `{workspace, project, board?, confirm: true}` | `./span shape jira --project KEY [--board N] --confirm <id>` |
| New session from a sealed story | `POST /shapes/{id}/fork` | `./span shape fork <id>` |
| The sealed story | the `story` and `digest` of `GET /shapes/{id}` | `./span shape story [--json] <id>` |
| Create a run from the sealed story | `POST /spans` with the story and `issue` | `./span shape story <id> > story.md`, then `./span quickstart --story-file story.md [--issue KEY]` and `./span run` |

- Each answered or assumed question gets an answer id (`answerId`: `A1`, `A2`...), kept when an assumption is replaced by an answer. An answered question is final; only an assumed one can be answered instead, until the session is sealed.
- Source references (`code` kind, `path:line` or `path:line-line`) are read from the session's repository as it is now, never from outside it, and never through a link that leaves it. The page shows the lines around the cited one.
- A session created by a split has `parent`, and the parent has `children`. A session created by a fork has `parent` too.
- The command line does not hold the server's per-session lock: do not run a `./span shape` verb on a session while a request to the same session is in flight.

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
commit (7 to 40 hex digits; a branch named like a sha, such as `deadbeef`, cannot be a base).
Tags and commits are not bases. Anything else is refused
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
so a local branch is enough there). Span asks origin itself (`git ls-remote --exit-code origin
-- refs/heads/<base>`), so a branch deleted on origin mid-run counts as gone even though the
checkout still has its `origin/<base>` ref; that stale ref is dropped, your local branch is
not touched. Offline, the refs already known decide. A base that is only local, a tag, or
missing holds the run, before any build work and before any push:

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
branch. Now it uses `--base`, else `base:` in `span.yaml` for feat and fix, else `dev`, or
`master` for a hotfix. The kind comes from `--branch`, or, without it, from the branch that is
checked out: a run on `hotfix/h` opens into `master`, not `dev`. `--branch` is the branch the
run works on with or without `--worktree`: it must be `feat/`, `fix/` or `hotfix/`, and a new
one is cut from the base. Before, `--branch` without `--worktree` was ignored. This is a deliberate change: on a repository without that base, or
on a checkout that is on the base itself, the run holds before the first stage. The base is
also recorded on the run, so [Prove only what the change can affect](#prove-only-what-the-change-can-affect)
diffs against it.

The desk fills the field's suggestions from `GET /branches?path=<folder>`: the local and
`origin` branches of that repository, `origin/` stripped, the default base first, then the
50 most recently committed, then the rest alphabetically, 200 in all; and `default`, the base
an empty field means (`base:` in `span.yaml`, else `dev`). The placeholder shows `master`
when the kind is `hotfix`.

## Start a run (desk)

The desk runs `GET /start/precheck` before it shows the start form. The first blocking refusal is
returned as JSON with `ok: false`, `code` (6, 8, or 14–18; see the canvas notes), and `message`.
When nothing blocks, the body is `{ "ok": true }`.

- `GET /health` answers whether the server is reachable and authorized (`token` query or `Authorization: Bearer`).
- `GET /start/precheck?path=<folder>&story=...&issue=...&kind=...&base=...&repo=...` runs the same checks as a start.
- `GET /start/what-will-happen?path=&kind=&base=` summarizes policy, agent, and stages for the launch.
- `GET /start/sealed?path=` lists shape sessions with `status=sealed` for picking a sealed story.
- `GET /start/run-key?path=&shape=` returns the Jira key or the next preview span id for the run label.
- `POST /spans` starts one run per named repository (same fields as `POST /quickstart`) and returns `results` per repo plus refusals when a repo cannot start.

From a shell, `./span start precheck|what-will-happen|sealed|run-key|health` mirrors those routes. Add
`--json` for machine-readable output. Remote checks use `SPAN_TOKEN` and `--serve http://127.0.0.1:8787`
when the verbs call `health` against a running server.

`POST /quickstart` remains for the older desk path; new UI prefers precheck and `POST /spans`.

## First-run checks

The welcome page (`/welcome`) checks your setup against the running server instead of guessing.

- `GET /onboarding/folder?path=<folder>` reads one folder: `exists`, `git`, the checked-out `branch`, the `origin` remote URL, and `dirty`, the number of files with uncommitted changes (untracked ones included). `reason` says why a folder cannot be used.
- `GET /onboarding/folders?path=<folder>` lists the directories under a path for the Browse picker, hidden folders left out, each marked when it is a git repository, with `parent` for the way up. An empty path starts at your home folder.
- `POST /jira/check` (`project`, `url`, `email`, `token`) asks Jira whether it accepts the account, without saving it. It answers `status`: `ok`, `unauthorized`, `unreachable` or `invalid`, and a `message`. An empty `token` tests the one already saved. `POST /jira/account` saves it.
- `GET /agent` gives each CLI that is not on your PATH an `install` command. The page shows it with a Copy button. Whether a CLI is logged in cannot be read, so Span never claims it is.

The merge stage checks it too. If the change targets another branch (for example `main`
when the run's base is `dev`), the run holds instead of merging: retarget the change
(`gh pr edit <n> --base <base>`) or approve the merge stage to merge it where it is; the run's base then becomes that branch, so
the conflict check reads it. This is what a stacked change needs when its base branch was
merged and deleted and the host moved the change. If the
change was already merged into another branch, the run holds with that branch named, so a
merge into the wrong place is not closed as if it were right; check that branch, then
approve the stage to close the run. A host that cannot report the target, such as the local
review used without `gh`, is not checked.

## House style

`harness/skills` is the writing standard, copied from the skill set so two people produce one author's code. The build prompt lists each skill name, its trigger, and the path of `SKILL.md`. The agent reads that file only when the trigger matches the change. Reference files stay unloaded until the skill itself names one. Bodies are not pasted into the prompt.

A sealed `hold` or `stop` in frame, build, or inspect is stored in `<project>/.span/lessons.json`, the project checkout and not the run's own worktree, so parallel runs share what was learned. Each write re-reads the file under a short lock, so two runs finishing together keep each other's lessons. The next prompt for that same stage includes at most five of those lines. A lesson that fails to prevent the same mistake twice is dropped. The file is local, the lookup does not call a model, and the skill files are not rewritten.

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

`span serve` has no workspace of its own. Its `--spans` default, `.span/runs`, is relative to the folder you start it from, so start it inside the project or pass the absolute path above. `run` and `resume` default to `<workspace>/.span/runs`.

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

With `agent: any`, the order is cursor, claude, codex, pi. The first enabled CLI writes. If a second one is enabled, inspect runs on that other CLI. If none qualify, frame holds and no product code is written. The CLI is headless: `cursor-agent` gets `--trust --force`, `claude` gets `--dangerously-skip-permissions`, and `codex exec` gets `--approve-for-me`, which already uses the workspace-write sandbox and cannot be combined with `--sandbox`. A prompt for permission would have no one to answer it.

Build requires Organic Driven Development. `odd/tasks/<feature>.md` (the run's feature, from its branch; see [What span.yaml is for](#what-spanyaml-is-for)) must contain a title (`# ...`), `## Specs`, `## Tasks`, and `## Log`. If that document is missing, the span stops and does not open a change.

## Basic use

### Build

```bash
go build -o span ./cmd/span
go test ./...
```

`go test ./...` does not call an agent and does not deploy. It checks the gates with fakes.

### Project policy

`span.yaml` lives at the workspace root unless you pass `--policy`. It is optional. An empty mode means `auto`.

#### What span.yaml is for

`span.yaml` is project policy only: the agent, the checks, the stage modes, the base, the
environments, the manifest. It never names a run. Every run takes its identity from its
own branch, one feature per branch name:

| Branch | Feature |
| --- | --- |
| `feat/sal-4-cards` | `sal-4-cards` |
| `fix/sal-4-cards` | `fix-sal-4-cards` |
| `hotfix/sal-4-cards` | `hotfix-sal-4-cards` |
| `feat/x/y`, `feat/fix-a` | slugged, plus `-` and six hex digits of the branch name's SHA-1 |
| `dev`, `main`, a detached `HEAD` | the run id |

A name that had to change (`/`, a character other than a letter, digit, `.`, `-` or
`_`, more than 80 characters, or a `feat/` suffix that reads as another kind) carries the
hash, so two branches never share a feature. A capital letter alone only lowercases the name
(`feat/SAL-4-cards` is `sal-4-cards`), since branches that differ only in case cannot coexist
on a case-insensitive checkout. A Windows device name (`con`, `nul`, `com1`)
gets a `-` after it. A branch that is not `feat/`, `fix/` or `hotfix/` is shared by every
run started on it, so the run id names the run instead. That feature names the run's
document `odd/tasks/<feature>.md` and its commit scope `feat(<feature>): span <id>`. The
pull request title is the story's first line (without `#` or control characters, at most
100 characters); a first line that is only a story key (`SAL-4`) is joined with the next
one. Two runs on two branches therefore never write the same file, so they can run in
parallel without conflicting. The feature is recorded in the run, so a resume keeps it; a
run recorded before this reads it from its branch.

Without `span.yaml` a run still starts, resumes, wakes, and gets its own worktree when the
checkout is busy: it gets the policy
`./span quickstart` would write (`agent: any`, `rules: harness`, open `auto`, merge
`hold`, the detected check such as `go test ./...` for a `go.mod`, the harness skills).
Nothing is written to disk for that. A `span.yaml` that exists but does not load is still
an error, and so is a `--policy <path>` that does not exist: only the implicit
`<workspace>/span.yaml` may be missing. `feature:` in an older file is read and ignored; you can delete it.

```yaml
agent: any
stages:
  open: auto
  merge: hold
  conflict: hold
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
`conflict` is how a merge conflict is resolved: `hold` (the default, unlike the other stages) waits for a person; `auto` applies low-risk options on its own. See [When the merge conflicts](#when-the-merge-conflicts).

`prove`, `qa` and `promote` commands are argv words, not a shell line. `go test ./...` is valid. `go test ./... | wc`, redirects, parentheses, backticks, `;`, `|`, and `&` are not executed. The stage holds.

Two things a shell would do are understood without one. Single or double quotes group words, so `pytest -k "a and b"` passes one argument (inside double quotes `\"` and `\\` are escapes, and outside quotes a backslash is a plain character, so a Windows path is safe). Words of the form `NAME=value` before the program are environment for that command only, so `CI=1 pnpm test` works. There is no `cd`: point the tool at the folder instead, for example `npm --prefix web test`, `pnpm -C web test`, `go -C service test ./...`, or `make -C web test`.

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
| merge | Checks the base for conflicts, then merges when it is not already merged | `hold` when the mode is `hold`, a conflict waits for a choice, or the result cannot be confirmed. A resolved conflict returns to prove |
| promote | Walks environments in order | `hold` on a `hold` environment or an unknown result. `stop` when the command finished and failed |
| note | Records lead time from frame | Reached only after promote advances |

`qa` numbers the lines under `Acceptance criteria:` as `AC-1`, `AC-2`, and so on. The build prompt tells the agent to put each id on a test declaration (`func Test`, `test(`, `it(`, or `def test_`), as `AC-1` or `AC1`. A comment is not enough. The command is `qa.command` when set. Otherwise a repo with `playwright.config.*`, or `qa.browser: true`, runs `npx playwright test`. Otherwise Span uses the first `prove.acceptance` command, then the first `prove.fast` command. The result is `.span/runs/<id>/qa-gate.json` with `pass`, `fail`, or `unverified`. This stage does not comment on Jira.

An acceptance id is only `fail` when a report names the test that ran and failed. `qa.report` points at a JUnit XML or a Vitest/Jest JSON report; with no path set, Span looks in `test-results/junit.xml`, `junit.xml`, `test-results/vitest.json`, `test-results/results.json`, `test-results.json`, and `report.json`. A report older than the command is ignored.

Without a report, a zero exit marks the declared ids `pass` and a non-zero exit marks them `unverified`. The process failing is not evidence that a named criterion failed.

QA does not run the fast command a second time. When prove already ran that exact command on this commit in a clean tree, the gate reads that result and only checks the declarations. A dirty tree or a different commit runs it again.

`./span resume` after `Fix and retry` continues at build, not at the stage that held.

A browser repo also tells the agent to test user-visible behavior with `getByRole` and web-first `expect`, and to record a trace on the first retry. `@playwright/mcp --caps=testing` is named as an authoring aid. The gate is the command, not an MCP session.

Span commits. The agent does not. The message is `feat(<feature>): span <id>`, which names the session; `<feature>` comes from the run's branch. The author and committer are **whoever that checkout is configured with**: Span sets no identity, adds no co-author trailer, and never overwrites your `user.name` or `user.email`. `git add` excludes `.span`, so run state stays out of the commit. The default branch is not recreated. The agent is not allowed to push, merge, or deploy.

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

Once a run is done and its branch was merged with a merge commit, the page reads the change from the base as it was just before that merge, since the branch is already in the base and a plain `base...head` diff would be empty. A fast-forward leaves no merge commit, so such a run shows no diff.

When a command prints a test summary Span can read, the statement keeps the counts (`tests`: passed, failed, skipped) and the run page shows them as "41 passed · 2 failed". Span reads the summary lines of `vitest`, `jest` and `pytest`, and the `--- PASS` lines of `go test -v`. Plain `go test` prints no pass lines, so it has no count; use `go test -v` to get one. A command that prints no summary, such as a build or a linter, has no count and the page says so instead of guessing.

Two stops run before the cap:

- The same failure twice. The fingerprint of a failure is the command plus the lines that name what broke. If it comes back unchanged on the next attempt, the build did not fix it, and another run would spend the same money for the same result. The span holds with `no progress`.
- The same command passing and then failing on one commit. The result is not attributable to the change, so the span holds as `inconclusive` and the criteria stay unverified. `Re-verify` runs it again without touching the code.

`repair.maxAttempts` is how many times a failure may send the span back to build, from 1 to 8, default 3. The next failure after that holds and keeps the reports. A restart from a stage clears the count. It also reads the commit the branch is on when the run continues, so edits you committed by hand are what prove, QA and inspect check, and a restart before open forgets the old change link (open reuses a live change on the branch). A restart never resets the branch or the working tree: the attempt stays in git.

Inspect is separate. A blocking finding archives build and prove for that iteration and returns to build, and build is given the finding: the next prompt starts with "Inspect sent this change back" and its reason, until inspect passes. Two returns can correct the change. The third holds with `inspect cap:` and the finding. Resuming a run held there is a request for one more round: build gets the finding again and the count starts over, so each extra round needs a person to ask for it. To fix it by hand instead, change the files and use Rewind to restart at prove, which runs inspect again.

### When the repository refuses the commit, and decisions

Span commits the build with the repository's own hooks on. A refusal is a blocker of class `commit-rejected`, not a plain hold. The hook's output, without terminal colors, is kept on the run and goes into the next build prompt, so the agent sees what the hook said. The second refusal with the same output means another build would end the same way: Span stops spending agent runs and opens a **decision**. Span never skips a hook, and no option of a decision offers to.

A decision is a question with the evidence quoted, two to four options that each say what Span does next, a recommended one, and a box for your own words. Answer it on the run's page. An option does one of three things: `build` runs the build again with your words as guidance, `retry` tries the commit again without running the agent (you changed the rule or the hook in the repository), `stop` ends the run. While a decision is open, `Resume` does nothing and the page offers the decision instead; a restart from a stage clears it.

The build agent can raise a decision too. When the brief asks for something a rule of the repository forbids, or it cannot go on without you, the prompt tells it to write the request to `.span/runs/<id>/decision.json` and stop, instead of choosing a side or asking in the chat. Span reads the file once, checks it, and holds with `decision needed`. A request it cannot show (fewer than two options, an action Span does not offer) holds with the reason, and the agent gets no commit made for it. Your answer is written to `answers.md` and stays in the prompt of every later build. Each option may carry `gain` and `cost`, a few words each (one line, up to 120 characters), which the page shows side by side under the option so you can weigh them; both are optional.

From a shell, the page's two actions are:

```
./span decision show <run> [--json]                                  # the question, the evidence and the options
./span decision choose <run> <option> [--text "..."] [--decision <id>] [--no-resume] [--json]
```

`choose` is "Choose and continue": the option and your words are the same answer the page posts to
`POST /spans/<id>/decide`, checked the same way (an option the decision does not offer, or an answer for
a decision that was replaced, exits 1). It then continues the run as `./span resume` does.

| Page action | HTTP route | Command |
| --- | --- | --- |
| Read the decision | `GET /spans/<id>` (`cursor.decision`) | `span decision show <run>` |
| Choose and continue | `POST /spans/<id>/decide` | `span decision choose <run> <option> --text ...` |
| Read the conflict and every option's diff | `GET /spans/<id>` (`cursor.conflict`) | `span conflict show <run> --options` |
| Apply resolution | `POST /spans/<id>/conflict/choose` | `span conflict choose <run> <group>=<option>...` |
| Leave a hint for the agent | `POST /spans/<id>/conflict/hint` | `span conflict hint <run> --text ...` |
| Abort merge | `POST /spans/<id>/conflict/abort` | `span conflict abort <run>` |


### What you decided stays decided

Everything a person decides or writes goes to `.span/runs/<id>/decisions.jsonl`, one line each, and the file only grows. An answer that steers the build (`build` options) is a **decision**: it is added to the prompt of every stage, frame and inspect included, as `[D3] <when>, at <stage>, on "<title>": <your words>`, and it outranks the story where they differ. It never outranks a rule the repository enforces. A later decision on the same problem supersedes the earlier one, and both stay in the file. Each version of a note you save for a stage is logged too, so a second note no longer erases the first; the stage's own file still holds the latest and is what the prompt reads.

The story is read again from Jira on `Reload`, and a decision is not part of the story, so it survives. When the story that comes back differs from the one the decisions were taken against, Span does not apply them silently or drop them: the next frame or build opens a decision that quotes each one and asks whether they still hold. `They still hold` anchors them to the new story, `Drop them` retires them (they stay in the log), `Stop` ends the run. Reading the story again and getting the same text asks nothing.

### Gates before the build: rules and plan

Frame is given the repository's `AGENTS.md` (the first 8,000 bytes) and asked to list each place where the story asks for something a rule forbids. It does not resolve one. Span opens a decision that quotes both sides, recommends following the rule because the rule is enforced and the story is not, and offers `Follow the story, I will change the rule` and `Stop`. Your answer is a decision in the log, so it reaches every later prompt, and the same conflict is not asked again for that version of the story, however frame words it next time. A repository without `AGENTS.md` skips this.

`stages.plan` decides whether the plan frame proposes waits for you before anything is built:

```yaml
stages:
  plan: conditional   # auto (default), conditional, hold
```

`auto` never waits. `hold` always waits. `conditional` waits only when the story weighs `deep`. The plan comes as a decision: the claims and assumptions quoted, `Build this plan`, `Revise it with my words` and `Stop`. Approving does not run frame again, so the plan you read is the plan that is built. Revising runs frame with your words and puts the new plan back for review. `open` and `merge` keep their own `auto` or `hold`; `merge` still defaults to `auto`, so set `merge: hold` where a person must release it.

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
| Codex | `codex` | `codex` | `exec --approve-for-me` | `~/.codex/AGENTS.md` says Organic Driven Development |
| Pi | `pi` | `pi` | `--print --approve` | `~/.pi/agent/npm/node_modules/gentle-pi` is installed |

`agent: any` or `auto` picks the first ready CLI in that order, and uses a second ready
one for inspect, so the review is not the same model that wrote the change. The binary
name is accepted too, so an older `agent: cursor-agent` keeps working.

Choose it on the page: the desk has an **Agent** picker under the folder. It lists all
four with the reason each one cannot run, and saving rewrites the one `agent:` line in
`span.yaml` without touching a comment. A folder without `span.yaml` gets one holding just
that `agent:` line, but only when the folder is a git repository or already has `.span/`;
anything else is refused, so the picker never leaves a file in a home directory.

`GET /agent?path=<folder>` returns the same list, and `POST /agent` with `project` and
`agent` writes the choice.

### Editing `span.yaml` from Settings

The Settings page edits the base branch, the three check lists (`prove.focused`, `prove.fast`,
`prove.acceptance`), `qa.command`, `repair.maxAttempts` and the `stages` modes (`open`, `merge`,
`promote`, `plan`, `conflict`), the CLI of each model stage (`agents.frame`, `agents.build`,
`agents.inspect`) and `jira.doneStatus`. A change shows under **What changes** as the old and new value
before anything is written; **Save N changes** writes it and **Discard changes** drops it.

`jira.doneStatus` moves the story to that status when a run ends; a story that cannot be moved is
noted in the run and never fails it. Without the key nothing is moved. Test counts are also read
from `go test -json`. The **What changes** panel lists what each save does to runs.

A save rewrites only the lines of the keys that changed. Comments, key order and every other key
stay as they were, and the new file is checked with the same rules as a run before it replaces
the old one. A value the file would refuse (a mode that is not `auto` or `hold`, `repair` outside
1 to 8, a base such as `HEAD`, a check with shell punctuation) is refused with the reason and
nothing is written. A section written inline (`stages: {open: auto}`) is refused too: write it as
a block first. A folder without `span.yaml` gets one only when it is a git repository or already
has `.span/`, as with the agent picker.

`POST /config/preview` returns `{"lines": [{"key", "before", "after"}]}` and writes nothing;
`POST /config` writes and returns the same lines plus the settings as they now read. Both take
`project` and any of `base`, `open`, `merge`, `promote`, `plan`, `conflict`, `focused`, `fast`,
`acceptance` (one check per line), `qa` and `repair`. A field that is absent is left alone; an
empty `focused`, `fast` or `acceptance` empties the list, and an empty `base` or `qa` removes
the key. Any other field is refused. Not editable here: `manifest`, `promote.environments`,
`commit`, `limits`, `jira`.

### A folder of several repositories

Span works from one repository or from a folder that holds several. Where a run starts decides
which `span.yaml` speaks:

- **One repository, started in it** (`--workspace` is the repository, or the desk's Folder is
  the repository): its own `span.yaml` is the policy. Nothing else is read.
- **A folder, one repository selected, or several:** each selected repository runs with its own
  `span.yaml`. What that file leaves unset comes from the folder's `span.yaml`: an `agent` of
  `any` (or none), and the `models` of any CLI it names no model for. An agent or a model the
  repository chose itself always wins. The agent and model picked on the desk are saved in the
  folder's file, so they reach every repository that has not chosen its own.

The run remembers the folder it was launched from, so `resume` and a resume from the page apply
the same policy. A folder with no `span.yaml` changes nothing.

### Deleting a run

Each run in the desk's `spans` list has a **Delete** button. It asks once more, then removes that
run's record (its journal, evidence and brief) from the spans folder. The branch, the pull request
and any worktree the run made stay, because they may hold work you still want. A run that is being
driven, by this server or by `span resume`, is refused: stop it first.

### Metrics

`/metrics` (linked from the desk) summarises every run under the server's state directory: runs
by outcome, how many finished without a repair or a second iteration, and per stage the
advances, holds and stops with the median and p90 time. A stage's time runs from the previous
sealed record to its own, so it includes waiting on a person. The agents table counts the
stages each CLI and model ran and how they ended; runs started before the journal recorded
agents do not appear in it. Cost and tokens are not shown because runs do not record them.
The same data is `GET /metrics` with `Accept: application/json`.

### Choosing the model of each agent

**Gentle AI and its configuration are required.** Span does not pick models by itself:
each CLI keeps the models Gentle AI set up for it, including the ones it uses for its own
sub-agents. A CLI without Gentle AI is not ready, cannot be chosen, and cannot be given a
model. Choosing a model here only changes the one that orchestrates the work.

`models:` in `span.yaml` holds one model per CLI id. Whenever that CLI runs (frame, build,
inspect, or proposing options for a conflict) Span adds `--model <name>` to its fixed flags.
A CLI with no entry gets no flag and runs on the model Gentle AI configured.

```yaml
models:
  cursor: gpt-5.3-codex
  claude: opus
  pi: xai/grok-4.7
```

On the page, each CLI has a **model** field under the Agent picker and a **Browse N models**
list beside it: the whole list the CLI reports, with a search, whatever is already chosen. The
field accepts any valid name, so a model the list does not show can still be typed:

| CLI | Where the suggestions come from |
| --- | --- |
| Cursor | `cursor-agent --list-models`, the models of your account |
| Pi | `pi --list-models`, written `provider/model` |
| Claude | the aliases `opus`, `sonnet` and `haiku` (it cannot list models) |
| Codex | none: it cannot list models, so the name is typed |

`span.yaml` is read again before each stage, not once when the run starts. An agent or model
picked on the desk while a run is going is the one the next stage uses; a stage already
running keeps what it started with. If the file cannot be read at that moment (a half-written
save), the stage keeps the policy the run began with. Every agent stage writes an `agent.run`
line to the run's journal with the CLI and the model it ran with, and the run page shows them
under **ran with**. A CLI with no model chosen shows "no model chosen, Gentle AI default"
instead of a name Span cannot know.

Clearing the field removes the entry, and the block when it is empty. A model name is
letters, digits and `. _ : / @ + -`, up to 100 characters, and never starts with a dash, so a
value cannot be read as a flag; it is checked when `span.yaml` loads, when the page saves,
and again before the CLI starts, and it is passed as its own argument. A `span.yaml` whose
`models:` is written inline (`models: {cursor: x}`) loads, but the page will not rewrite it.

`GET /agent/models?cli=<id>` returns the suggestions (listed CLIs are asked at most once a
minute, a failure for 30 seconds, and concurrent requests share one process; a failure comes
back as `error` and the field stays editable), and
`POST /agent/model` with `project`, `cli` and `model` writes the choice. `GET /agent` also
returns the chosen `models`. A change reaches the runs that start after it, not one already
running.

### Choosing the agent per stage, and pausing for it

The project picks a default agent. One run can override it per stage without touching
`span.yaml`: on a started span, the ledger tab has **Agent for a stage of this run** with
frame, build, and inspect. The choice lives in `.span/runs/<id>/agents.json` and applies
the next time that stage starts.

Resolution order for a stage: the run's choice, then `agents.<stage>` from the policy, then
`agent:` from the policy, then the first ready CLI. An empty choice, or `Policy default`, clears the exception. Inspect keeps
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

**`span run` on a busy checkout.** From the terminal, `span run --branch feat/x` does the same
when another run holds the checkout: it prints `this checkout is busy (...): running feat/x in
its own worktree at <path>` and goes on there, instead of holding. A free checkout is edited in
place as before, and `--worktree` still forces a tree. Without `--branch` there is no name for
a tree, so it refuses and tells you to pass one.

**Capping parallel runs.** Many agents at once can exhaust the quota of their CLI.
`span serve --max-parallel 4` drives at most four runs in that process; a launch or a wake past
that is refused with `span serve is at its limit of 4 runs at once (--max-parallel 4): wait for
a run to finish, or raise --max-parallel` (HTTP 409), and nothing is created for it. A slot is
freed when a run ends, fails or is stopped. The default, `0`, is no limit. The cap counts the
runs of one `span serve` process; `span run` and `span resume` from other terminals are not
counted. Two launches that arrive in the same instant at the edge of the cap can both pass the
early check; the second is then refused later and leaves a run in hold, which you resume.

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
there is no flight. `./span stop <run>` does the same from a terminal, and from any machine
that can reach the run's folder: see "Stop from the CLI" below.

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

### When the merge conflicts

Before `merge` asks the host to merge, it checks whether the base merges cleanly into the
run's branch without touching the checkout: `git fetch origin
+refs/heads/<base>:refs/remotes/origin/<base>` (so a single-branch or CI clone gets the base
too; the local `<base>` when there is no `origin`), then `git merge-tree --write-tree HEAD
origin/<base>`. A clean result changes nothing: a branch that is merely behind its base is
left to the host exactly as before, with no merge commit and no extra prove. When the base
cannot be fetched or found, or git is older than 2.38 and has no `merge-tree --write-tree`,
nothing is checked, the host decides, and the merge statement says why. A host that refuses
the merge because of conflicts (`gh pr merge` saying the merge commit cannot be cleanly
created, or that the change has merge conflicts) is not an unknown outcome either: the run
resolves the conflict. Only gh's stderr is read, never the command line, and a refusal from
branch policy, a required check or a missing review stays a plain failure. A refusal because
the change is behind its base (`not up to date with the base branch`) holds with `the change
is behind its base; resume to merge again after it is updated`; span does not update the
change on its own.

On a conflict, the run merges the base in its own checkout with `git merge --no-commit
--no-ff origin/<base>`. It never rebases and never force-pushes. It holds with the reason
`conflict: choose how to resolve`, and nothing is written to the branch until a person
chooses. Any conflict git reports is covered, and each file has a kind: `content`, `add-add`,
`modify-delete`, `rename` (rename/rename and rename/delete), `mode`, `binary`, `lockfile`
(`package-lock.json`, `yarn.lock`, `pnpm-lock.yaml`, `go.sum`, `Cargo.lock`, `poetry.lock`,
`Gemfile.lock`, `composer.lock`, `bun.lockb`), `generated` (`.generated.`, `_generated.`,
`generated/`, `dist/`, `*.pb.go`, `*.min.js`, or a `@generated` or `DO NOT EDIT` header in the
first five lines), and `other`: a path whose sides are different kinds of thing (a file
against a symlink or a directory, or a submodule). git parks one side of those under a
temporary `path~HEAD` or `path~<sha>` name; span folds it back into the real path, offers
only `ours` or `theirs`, and keeps the side's real mode, so a symlink stays a symlink and a
submodule records its commit.

The build agent reads the story and both sides of every file, framed by delimiters that
carry a random nonce and marked as data, not instructions, and groups the files that one
decision resolves. Each group gets 2 to 4 option cards with a title, a summary, a risk and
the diff of what it would write. Span computes that diff from the bytes, against this
branch's version. The risk on the card is span's, not the agent's: the agent can raise it but
not lower it below what the action can lose. `combine` and `keep` can be `low`, `ours` is at
least `medium`, and `theirs`, `delete` and `regenerate` are always `high`. An action is
`ours`, `theirs`, `combine` (the agent's merged text), `keep` (the file one side deleted),
`delete`, or `regenerate`. A reply that breaks these rules is refused and the agent is asked
once more; a second refusal, a missing agent, or a fourth proposal leaves only the mechanical
options: keep this branch's version, take the base's, or keep or delete a deleted file.

Every option's diff is served before anything is chosen: `GET /spans/<id>` returns the open
conflict as `cursor.conflict.groups[].options[].diff`, and the run page's **Compare options**
toggle (per group) shows them side by side, with the lines the options do not share marked;
the arrow keys move between options and Enter chooses. From a shell, the same data and the same decisions, calling the engine functions the routes
call (every verb takes `--spans .span/runs` and `--json`, never prompts, and exits 0 when it
worked, 1 when it was refused or failed, 2 on a usage error):

```
./span conflict show <run>                                  # groups, files and summaries
./span conflict show <run> --options [--json]               # plus every option, its risk, action and diff
./span conflict choose <run> g1=combine g2=theirs           # one <group>=<option> for EVERY group
./span conflict choose <run> g1 combine                     # the same, for a conflict with one group
./span conflict hint <run> --text "keep the new rounding"   # ask the agent for a new proposal
./span conflict abort <run>                                 # undo the merge; the run holds
```

`choose`, `hint` and `abort` record the decision and then continue the run in this process,
the way `./span resume` does, and print where it stopped next. `--no-resume` only records it:
the run goes on from `./span resume <run>` or from a running `./span serve`. A run another
process is driving is refused (exit 1) rather than raced. `show` exits 1 when the run has no
conflict open.

The agent runs in the checkout. Whatever it does there is put back before its proposal is
used: staged, edited or new files, commits, amends, resets, a checkout of another branch, the
merge state, every ref (branches, tags, notes, remote-tracking refs), hooks, git config and
`.git/info` (attributes, exclude). The git directory is put back with plain file operations
before span runs any git command, and every git command of the conflict phase runs with no
hooks, no fsmonitor, no pager and no `ext::` transport, so nothing the agent configured runs
while span cleans up. A symlink the agent planted in the git directory or the tree is removed
as a link; nothing is written or deleted through it. The card's note names what it touched.
Ignored files it created are named too but never deleted. When span cannot prove the
repository is back where it was, the conflict fails and only abort is offered. The merge
commit is made with every hook disabled, and it is only accepted when its parents are exactly
the run's head and the base span merged; any other commit is refused and never pushed. No file
reaches the merge commit unless a card showed it.

When the conflict starts, span pins the repository: where `.git` and the git directories are,
and their shape (`.git` as a directory or gitfile, `commondir`, `gitdir`, `HEAD`, the index,
`objects/info/alternates`, `modules/`, and the value of every local branch no worktree has
checked out). A repository shared with other parallel runs is not "changed" when git repacks
its refs or another run adds a branch or a worktree; moving a quiet branch, or losing this
run's own worktree, still is. Before it runs git again
it checks that shape with plain file operations. What plain file operations can put back is put
back first (`commondir`, `gitdir`, `alternates`, a `HEAD` replaced by a symlink). Anything else
(`.git` replaced by a symlink or a gitfile, the git directories moved, the index replaced, a
a quiet branch that moved, new modules) fails closed: the conflict moves to `failed`
with `the agent changed the repository structure; span did not run git. Inspect <path> by
hand`, and abort does nothing until the repository is fixed by hand. Every git command span
runs names the pinned repository explicitly (`GIT_DIR`, `GIT_COMMON_DIR`, `GIT_WORK_TREE`) and
runs with no fsmonitor, no pager, no `ext::` transport and no submodule recursion, so a nested
repository the agent staged never starts git inside it. Repository hooks are off only during
the conflict phase; span's ordinary commits and pushes keep running your repository's own
commit and push hooks. Your global config, credential helpers and ssh settings still apply, so pushing
authenticates as before.

What this does not cover: your global and system git config (`~/.gitconfig` and the files it
includes), anything the agent does outside the checkout (network calls, other directories),
packs, objects and reflogs under `.git`, and an agent that escapes its sandbox. Span cannot
fully contain an agent that has write access to `.git`: the structural check fails closed
instead of guessing.

`regenerate` is only offered for lockfiles and generated files. Its command must start with
one of `npm`, `pnpm`, `yarn`, `bun`, `npx`, `go`, `cargo`, `poetry`, `pip`, `pip3`, `uv`,
`bundle`, `composer`, `dotnet`, `gradle`, `mvn` or `make`, by name, and carry no shell syntax.
It is shown on the card and runs as argv in the checkout only after you choose it: the files
are put back to this branch's version, the command runs, and what it wrote is staged. A
command that changes any other file is undone and the cards come back with a note naming
those files.

Choose one option per group. Span stages each path through git itself (`git update-index`
and `git checkout-index`), never by writing the file directly, refuses a path through a
symlinked directory, and refuses a text that still has a conflict (a `<<<<<<<` line, a
`=======` line and a `>>>>>>>` line, in that order). It commits `merge: bring <base> into <branch>
(resolve conflicts)` and the run goes back to `prove`, because the sealed prove, QA and inspect
statements describe the head before the merge. The merge commit stays local until prove, QA
and inspect passed on it: it is pushed right before the host is asked to merge, so a failing
check never publishes it. If prove fails, the normal repair loop takes over. With `merge:
hold`, the merged head needs its own approval before that push. A choice that cannot be
applied puts the cards back with what went wrong. A run resolves at most three conflicts;
after that it holds with `conflict: too many resolutions; the base keeps moving`.

Not satisfied with the cards? Send a hint, free text, and the agent proposes again with it
(three proposals per conflict). **Abort** runs `git merge --abort`, puts back any tracked file
a command touched outside the merge, clears the conflict, and holds with `conflict: aborted`:
the branch is exactly as it was. Restarting from a stage is refused while a conflict is open.

`stages.conflict: auto` applies the first option of every group without waiting, only when
span's risk for every one of them is `low` (so never a `regenerate`, `theirs` or `delete`).
Prove runs afterwards like after any merge, and nothing is pushed until it passes. Anything
else holds as with `hold`, the default.

The page reads the conflict from the cursor's `conflict` field and from live
`{"type":"conflict","conflict":{...}}` frames sent on every state change (`proposing`,
`choosing`, `applying`, `failed`).

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

Every HTTP request sends `Authorization: Bearer a-long-secret`. A browser cannot set that header on a WebSocket, so the page sends the same secret as the subprotocol `span.` plus the raw URL-safe base64 of the token. On localhost, leave the **Access token** in **Settings** empty. When a server answers 401, the desk opens Settings by itself and asks for it. A value left in the tab used to make the browser drop the live socket: the page offered a subprotocol and the server did not name one back. With no `SPAN_TOKEN`, the server accepts that offered protocol and the socket opens. When `SPAN_TOKEN` is set, type it under **Settings** and choose **Save token** before **Configure and run**. The token stays in this browser tab.

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

| GET | `/spans/<id>` | `Accept: application/json` returns the cursor, statements, acceptance lines, gate, and what the run is about: `title` (the brief's first line without the story key) and `repo` (the folder it writes in). Any other accept is the span page |
| GET | `/spans/<id>/implementation` | `git diff <base>...<head>`: `stat`, a `patch` capped at 256 KiB, `commits` (how many) and `commitLog`, the same commits as `{sha, subject, time}`, oldest first. `commitLog` is empty, not absent, when the branch adds nothing |
| GET | `/spans/<id>/log?path=iter/N/logs/<name>.log` | The full output of one command, `{path, text, lines, truncated}`. Only a clean relative `.log` path under `iter/`, resolved inside the run (no `..`, no link that leaves it), a regular file, the last 512 KiB. Any refusal is `404`, so a caller cannot tell a missing file from one that is not theirs |
| GET | `/spans/<id>/feature` | The run's `odd/tasks/<feature>.md`, capped at 64 KiB, for the report. Empty when there is none |
| GET | `/spans/<id>/acceptance` | Each acceptance line, its declaration, and the gate result |
| GET | `/spans/<id>/live` | WebSocket. The first frame replays the run. Later frames wake the page |
| GET | `/spans/<id>/agent` | Whether this process still has a worker for that run |
| GET | `/agents` | The workers this process is running |
| POST | `/spans/<id>/relaunch` | Starts a run again when the cursor is not paused, stopped, or waiting, and no worker is left. `202` |
| POST | `/spans/<id>/answer` | JSON `{"answers":[{"question","body"}]}`. `202` and frame runs again |
| POST | `/spans/<id>/story` | Reads the Jira issue again. `202` and frame runs again |
| POST | `/spans/<id>/resume` | Continues a hold that is not waiting for an answer |
| POST | `/spans/<id>/conflict/choose` | Form fields `group=<id>&option=<id>`, one pair per group. Every group needs one known option. `202` and the merge stage applies it. `409` when no conflict is waiting for a choice |
| POST | `/spans/<id>/conflict/hint` | Form field `text`. `202` and the agent proposes again. `409` as above |
| POST | `/spans/<id>/conflict/abort` | `git merge --abort`, clears the conflict, holds with `conflict: aborted`. `202` |
| POST | `/spans/<id>/instructions` | Form fields `stage` and `body` |
| POST | `/spans/<id>/instructions/<stage>` | `text/plain` body or form field `body` |

Each run in `{"spans":[...]}` carries `id`, `label`, `issue`, `repo`, `stage`, `verdict` (`running`, `hold`, `stop` or `done`), `title` (the first line of the brief without the story key), `branch`, `reason` (why a run is held, stopped or failing), `updatedAt` (the last time the cursor moved), `startedAt`, and, for a held run, `need` (`conflict`, `questions` or `decision`) with `count` (how many files, questions or open decisions). The desk reads them to say "3 questions" and to offer Answer, Decide or Resolve. Done runs older than 30 days stay out of the list unless the page filters by Done.

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

- [ ] `span.yaml`, when present, has prove commands and environments. Without it the quickstart defaults apply.
- [ ] The workspace is a git repo with `origin` when open is `auto`.
- [ ] When the story comes from Jira, that same folder has its own `.span/jira.json`.
- [ ] The chosen CLI is on `PATH` and Gentle AI is detected in its file.
- [ ] The brief states the request without invented requirements.
- [ ] After `hold`, you read `.span/runs/<id>/iter/<n>/<stage>.md` before `./span resume`.
- [ ] After `stop`, you do not expect resume to continue.

## The run page from a terminal

Every action of the run page is a route of `./span serve` and a `./span` verb that calls the same Go function, so a run on a remote machine can be driven over SSH with no browser. The verbs read and write the state directory (`--spans`, default `.span/runs`), never prompt, take `--json`, and exit `0` when done, `1` when the engine refuses or fails (an unknown run, a state that cannot take it, a run another process is driving) and `2` on a wrong call. The verbs that wake a run (`rewind`, `approve`, `answer`, and the `conflict` and `decision` ones) go on in the same process, as `./span resume` does; `--no-resume` records the choice and leaves the run held for `./span resume <run>` or a running `serve`.

| Page action | Route | Command |
| --- | --- | --- |
| The desk's list: stage, verdict, `need`, `count`, `startedAt` | `GET /spans` | `./span runs list [--json]` |
| One run: cursor, statements, acceptance lines, gate | `GET /spans/<id>` | `./span runs show <run> [--json]` |
| Implementation tab: files, commits, diff | `GET /spans/<id>/implementation` | `./span runs implementation <run> [--patch] [--json]` |
| Feature document report | `GET /spans/<id>/feature` | `./span runs feature <run> [--json]` |
| QA tab: cases and the gate | `GET /spans/<id>/acceptance` | `./span runs acceptance <run> [--json]` |
| Audit trail and its chain check (`1` when the chain is broken) | `GET /spans/<id>/events` | `./span runs events <run> [--json]` |
| "Open full log" | `GET /spans/<id>/log?path=` | `./span log <run> <path> [--json]` |
| Pause after the stage in flight | `POST /spans/<id>/pause` | `./span pause <run>` |
| Stop now: cancel the stage in flight | `POST /spans/<id>/stop` | `./span stop <run> [--json]` |
| Restart from a stage, with a note | `POST /spans/<id>/rewind` | `./span rewind <run> <stage> [--note "..."] [--no-resume]` |
| Leave a note for a stage | `POST /spans/<id>/instructions/<stage>` | `./span note <run> <stage> --text "..." \| --file path` |
| Agent for a stage (`any` clears it) | `POST /spans/<id>/agent` | `./span stage-agent <run> <stage> <agent\|any>` |
| Approve a stage that holds | `POST /spans/<id>/approve` | `./span approve <run> <stage> [--environment name] [--no-resume]` |
| Answer the questions of a waiting run | `POST /spans/<id>/answer` | `./span answer <run> --answer "question=text"... \| --file answers.json` |
| Continue a paused run, or a relaunch | `POST /spans/<id>/resume`, `/relaunch` | `./span resume <run>` |
| Delete a run | `POST /spans/<id>/delete` | `./span delete <run> --yes` |
| Decision panel | `GET /spans/<id>`, `POST /spans/<id>/decide` | `./span decision show\|choose` |
| Conflict panel | `GET /spans/<id>`, `POST /spans/<id>/conflict/*` | `./span conflict show\|choose\|hint\|abort` |

What a verb reads is what the route answers, field for field: `--json` prints the route's body. `runs list --json` prints `{"spans": [...]}`. `decision show --json` is `cursor.decision` and `conflict show --options --json` is `cursor.conflict` (without `--options` the diffs are left out). `conflict choose`, `hint`, `abort` and `decision choose` record the same choice the route records, then continue the run in this process unless `--no-resume`. Their `--json` is that outcome: `id` and `stage` (the same two fields the route answers), `action`, `outcome` (`recorded` when only the answer was stored, `continued` when the run went on in this process), `resumed`, and, when the run went on, `verdict` and `reason`. The route answers `202 {id, stage}` and the run continues inside `serve`, because the continuation happens in the server; the CLI waits for it and prints the result. `jira status --json` prints the body of `GET /jira` and exits 0 whether or not an account is saved (read `configured`); without `--json` it exits 1 when no account is saved. A QA reading of a run that is not on the desk is a JSON `404 {error: "not_found", message}`, like every other QA route.

The log of a command. A statement stores the path of each command's log (`iter/1/logs/prove-1.log`) and `lines`, how many lines the command printed, so the page can say "5 of 142 lines" over the last lines it shows. `./span log` and the route read the same file with the same limits: a clean relative `.log` path under `iter/`, resolved inside the run, a regular file, and the last 512 KiB (`truncated: true` when it was longer; the plain output then says so on stderr).

One page action has no verb, and why:

- **Read the Jira story again** (`POST /spans/<id>/story`) needs the server's Jira login and rebuilds the brief; run `./span quickstart --issue <key>` for a new run from the current story.

#### Stop from the CLI

`./span stop <run> [--json]` is **Stop now**. It writes a `stop` marker in the run's folder, the same way a pause is requested, and every process that drives a run (`serve`, `span run`, `span resume`, `span answer`) looks for it about every 300 ms. When it finds the marker it consumes it and cancels the flight, which is what the button does: the command or agent is killed and the run comes back held with the error. So a stop reaches a run driven by another process, and by a `serve` on a remote server when the CLI runs there against the same `--spans`.

The route and the verb answer alike. When nothing drives the run (nobody holds its lock) there is no stage to cancel: the route answers `{"stopped": false, "reason": "nothing is running this span"}` and the verb prints `done: false` with `nothing is running this run`, both exit 0, and neither leaves a marker. A marker left from before a flight started is discarded when the flight starts, so an old request never cancels a later resume. Exit codes: 0 asked or nothing running, 1 error, 2 usage, 3 the run is finished (refused), 4 no such run. `--json` prints `{id, action: "stop", done, reason}`.

`./span pause` writes the pause request the running process reads at its next stage boundary, so it also reaches a run driven by `serve`. The route answers `paused: false` for a run nothing is running; the verb cannot see another process's workers, so it only refuses a finished run, and a request made while nothing runs waits until the next resume discards it.


## The Implementation tab from a terminal

The Implementation tab (Overview, Diff, Commits, Reasoning, Audit) reads these routes. Each has a `./span implementation <run> <mode>` command that calls the same Go function (`internal/implementation`), so the page and a shell cannot disagree. All take `--spans` and `--json`, never prompt, and exit `0` when done, `1` when refused or failed, `2` on a wrong call, `4` when the run, file, commit or decision is not there, `3` when `--reasoning` finds assumed decisions nobody checked, and `12` when the audit chain is broken. With `--json` a failure prints `{"error","message"}` on stderr, as the routes do.

| Page action | Route | Command |
| --- | --- | --- |
| Files with status, counts, flags and viewed marks | `GET /spans/<id>/files` | `--files` |
| One file's diff (add `?commit=<sha>` for one commit; `?force=true` prints generated, binary and large files) | `GET /spans/<id>/files/<path>/diff` | `--diff <path> [--commit <sha>] [--force]` |
| Mark a file viewed (`?viewed=false` takes it off) | `POST /spans/<id>/files/<path>/viewed` | `--viewed <path> [--unview]` |
| Commits of the branch, newest first, with files and lines | `GET /spans/<id>/commits` | `--commits` |
| What one commit changed, and nothing before it | `GET /spans/<id>/commits/<sha>/diff` | `--commit <sha>` |
| Specs, tasks and decisions with "because", source and the assumed flag | `GET /spans/<id>/reasoning` | `--reasoning` |
| Mark assumed decisions as checked (JSON `{"ids":[...]}` or a form with `ids`) | `POST /spans/<id>/decisions/check` | `--check D1 D3` |
| The journal with each entry's hash and the previous hash it should have | `GET /spans/<id>/audit` (`?format=json` downloads it) | `--audit` |
| Verify the chain | `GET /spans/<id>/audit/verify` | `--audit --verify` |

The whole-branch diff is `GET /spans/<id>/implementation` (also `./span runs implementation`).

**Files.** `status` is `A`, `M`, `D` or `R` (a rename carries `from`); `binary`, `generated` and `large` (over 5,000 changed lines) tell a reader that a line diff is not the right view. A diff of such a file answers with its flags and an empty patch unless `force` is set, so a remote reader is not sent megabytes nobody reads. `viewed` is the person's place in the list, kept in `viewed.json` of the run; it changes no code and nothing waits for it. `<sha>` may be a unique prefix of at least 4 characters and must be a commit of the run's branch.

**Reasoning.** The feature document `odd/tasks/<feature>.md` is read in Go: `## Specs` become `S1`.. (or the `S`/`AC` id the line starts with), `## Tasks` lines keep a trailing commit hash, and each `Decision:` line becomes `D1`.. in order. Write `Decision: <what> because <why> [source: S2; files: a.go, b.go]`. A decision with no `source` is **assumed**: the story did not say. `decisions/check` saves a person's mark on assumed decisions in `decisions.json`, records `decisions.checked` in the journal and changes no code; a settled decision or an unknown id is refused.

**Audit.** Every entry carries `hash` (SHA-256 of its line) and `want` (the hash the previous line really has). The chain holds when each entry's `prev` equals `want`; `brokenAt` is the first entry where it does not, and the entries after it are returned as stored but cannot be trusted.

## The implementation report

The Implementation tab opens with a report, then the diff, then the audit trail and the raw transcript (both closed). Nothing in it calls a model: it is computed from the diff, the sealed statements and the feature document. Each line says where it came from: detected by Span, reported by the agent, or verified by a check.

Order: verdict, risks, what and why, decisions, evidence, then the diff. The verdict is the worst risk level and says how hard to read ("Read every flagged file line by line", "A quick look is enough. Absence of signals is not proof of safety"). The order follows what reviewers act on first: how risky, then what could go wrong, then the reasoning, then the proof.

Detected risks, all heuristics of our own except the size bands:

| Signal | Level |
|---|---|
| More than 400 changed lines (review stops finding defects past about 400 lines at a time, SmartBear/Cisco) | high |
| 200 to 400 changed lines, or more than 10 files | medium |
| Auth, session, token, secret or crypto source files; schema or migration files; build, deploy or infrastructure files; `.env` files | high |
| A deleted test file; a secret-shaped string on an added line; inspect still blocking at the cap | high |
| Dependencies or configuration changed; generated or binary files committed; source changed with no test changed; a check that passed and failed on one commit; inspect sent the change back twice or more | medium |
| `TODO`, `FIXME`, `HACK` added | low |

The report also lists what it could not check (a cut patch, no prove or QA run, inspect not run, no feature document log), so a quiet report is never read as a safe one.

Build is asked to write two kinds of line in the feature document's `## Log`: `Decision: <what> because <why>` and `Risk: <what could go wrong> (<how likely, and what would catch it>)`. The report shows them as "assumed by the agent" and "reported by the agent". A run whose log has none says so.

Sources for the shape: Architecture Decision Records (Nygard) and MADR for decisions with their reasons, Google's change-description and small-change guidance, SmartBear/Cisco on review size, SLSA provenance for the audit trail (who ran what, on which commit, when), CodeRabbit's walkthrough and merge-risk presentation, and Martin Fowler's Ship/Show/Ask for matching review depth to risk.

## What a commit takes

After a stage that edits files, Span stages everything the agent left (except `.span` and `span.yaml`) and then checks it before committing:

- Build output and caches are left in the working tree and not committed, without stopping the run: `__pycache__/`, `*.pyc`, `*.pyo`, `node_modules/`, `.DS_Store` and `.atl/` (the agent tooling's skill registry, which is not part of any story). To version one of them on purpose, list it under `commit.allow`.
- A file that looks like a secret stops the commit. Nothing is committed, the index is cleared, and the run holds with the file and the reason. The match is never printed. The checks are the name (`.env` and `.env.*`, `*.pem`, `*.key`, `*.p12`, `*.pfx`, `id_rsa`, `id_ed25519` and the like; `.env.example`, `.sample`, `.template` and `.dist` are allowed) and the added lines (an AWS access key id, a private key block, a GitHub token, or a quoted credential of 20 or more characters).
- To let a reviewed path through, list it in `span.yaml`, then resume. An entry is an exact path or a folder written with a trailing slash, relative to the repository; a path that leaves the repository is refused when the policy loads. A listed path skips both checks.

```yaml
commit:
  allow:
    - fixtures/keys/
    - .env.test
```

These checks are heuristics, not a secret scanner: a secret that matches none of the shapes is committed. Run a scanner such as gitleaks in your own pipeline for coverage.

### Private origins and a host that lags

Two details of resolving a conflict that only show against a real host:

- The base is fetched with your system git config, which is where macOS keeps the keychain credential helper. The rest of the conflict phase still runs with the system config off. Before this, a private origin made the fetch ask for a password git may not prompt for, and the merge held with `the base dev could not be fetched from origin`.
- After the resolved merge commit is pushed, GitHub can still call the change conflicting for a few seconds. Span retries the merge four times over about half a minute (2, 4, 8 and 12 seconds) before reporting it, and only in that case. A conflict that is still there after that, a base that moved on its own, or any other refusal is reported at once.

## Limiting how long an agent may run

`limits.stageTimeout` bounds one agent call: frame, build or inspect. It is a Go duration from `1s` to `48h`, for example `45m`. An absent limit is no limit.

```yaml
limits:
  stageTimeout: 45m
```

When a call runs past it, Span stops the agent and the stage holds with `<stage> ran past limits.stageTimeout (45m0s) and was stopped. Raise the limit or split the story, then resume.` A timeout is not a failing check, so it does not spend a repair. Resuming starts that stage again.

Not covered: a limit on spend. Span does not read token counts or cost from the CLIs, so there is no budget to set. The limit above bounds time only.

## The run journal

Every run keeps `.span/runs/<id>/events.jsonl`, an append-only record of what happened to it and who did it. Statements already seal what each stage decided. The journal adds the rest:

- `span` events: one `stage` line per sealed statement, with the stage, verdict, iteration and the statement digest.
- `person` events: each action taken on the page that succeeded (`stop`, `pause`, `resume`, `approve`, `rewind`, `agent`, `answer`, `story`, `relaunch`, `instructions`, `conflict.*`). A refused request leaves no line.

Each line has a `seq`, a time, the actor and the type. A line also carries `prev`, the SHA-256 of the line before it, so a removed or edited line breaks the chain. `GET /spans/<id>/events` returns the events with `intact` and, when broken, `brokenAt`; the Audit panel of the Implementation tab lists the actions by a person and warns when the chain is broken.

Free text a person typed (an answer, a note, an instruction) is never stored in the journal, only its length. Short fixed fields such as the stage are.

Limits: the chain shows tampering, it does not prevent it, since anyone who can rewrite the file can rewrite the whole chain. Runs started before this version have no journal, and an empty one reads as intact. Nothing is exported to OpenTelemetry.

## When Span itself is killed

A `kill -9`, a crash or a power cut stops Span without letting it pause the run or stop its agent. Two things are handled:

- **The run shows as interrupted.** A run the cursor calls running, with nobody holding its lock for more than 10 seconds, is shown as paused with the reason `the run was interrupted: the process driving it ended. Resume to continue from this stage.` Resume it from the page or with `span resume <id>`; it continues from the stage it was in. Before this, such a run looked alive and `resume` refused it with `this run is not paused`.
- **The agent it left running is ended.** While an agent runs, Span writes its pid and name to `<tree>/.span/agent.pid`. When a run of that tree starts again, with the tree's lock held, it ends that process group if the pid is still alive and its command line still names the agent, so it cannot keep editing files under the next agent. A pid the system has since reused for something else is left alone. The journal records `agent.reaped`.

Limits: on Windows the note is cleared but the process is not killed, because its command line cannot be checked safely. Only one agent per tree is recorded at a time. Work an agent half-finished in the tree is not undone: the resumed stage starts from the files as they are.

## The QA tab from a terminal

Everything the QA tab shows is one HTTP route and one verb of `./span`, both reading the same code. Every verb takes `--spans .span/runs` and `--json`.

| Page | Route | Terminal |
| --- | --- | --- |
| Proof overview | `GET /spans/{id}/qa` | `./span qa show <run>` |
| Cases | `GET /spans/{id}/qa/cases?status=failed\|passed\|skipped\|new&group=file&q=` | `./span qa cases <run> [--failed\|--passed\|--skipped\|--new] [--group file] [--q text]` |
| One case | `GET /spans/{id}/qa/cases/{name}` | `./span qa case <run> <name>` |
| Rerun cases | `POST /spans/{id}/qa/rerun` `{"cases":[...]}` | `./span qa rerun <run> --failed` or `<name>...` |
| Go on without proof | `POST /spans/{id}/qa/accept` `{"reason":"..."}` | `./span qa accept <run> --reason "..."` |
| A gate's whole log | `GET /spans/{id}/logs/{gate}?from=&to=&q=&first_error=1` | `./span qa log <run> <gate> [--lines from:to] [--grep text] [--first-error]` |
| What the agent printed | `GET /spans/{id}/transcript?stage=&from=&limit=` | `./span transcript <run> [--stage name] [--from n] [--limit n]` |

Exit codes: `0` ok, `1` failed, `2` usage, `4` run, case, gate or stage not found, `10` a gate failed (`qa show`, and `qa rerun` when a case still fails), `11` no proof (the run never ran the gate, found no cases or has no sealed build), `13` the start of a log was cut (the lines kept are still printed).

What the page does not claim: a skipped or missing gate is `held`, never `pass`; a run with no sealed build says so; cases need `go test -json` or `-v` output; a rerun supports gates that are `go test` commands; accepting only records the person's reason; the agent's output is plain text, so a transcript has blocks, not roles or tool calls, and `structured` is false.
