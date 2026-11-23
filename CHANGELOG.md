# Changelog

This file starts at the repair loop. The repository has no baseline commit yet, so
earlier work is recorded only in `odd/tasks/`, one document per feature:
`span-engine`, `harness-lessons`, `resume-waiting-run`, `repair-loop`, `agent-limit`.

## Unreleased

- **Fixed**: a stage held by its mode (`merge: hold`, `open: hold`, a promote environment in `hold`) could never be released: the page offered "Fix and retry", which reached the same hold again. The page now offers an approval (`POST /spans/{id}/approve`), bound to the commit it showed and spent when the stage advances.
- **Fixed**: `span.yaml` was staged into the story's branch by the commit, so a checkout of another branch lost it and a run recorded without a project could not be resumed (`open .../span.yaml: no such file`).

### Merge conflicts are resolved on cards

- **Detected before the merge, without touching the tree.** The merge stage runs `git
  merge-tree --write-tree HEAD origin/<base>` after fetching the base (single-branch clones
  included). A clean base is left to the host exactly as before: no merge commit, no extra
  prove. No base, no fetch, or a git older than 2.38 skips the check and says so. A `gh pr
  merge` refused for conflicts (read from gh's stderr only) is classified as a conflict, never
  as `merge outcome unknown`; branch policy, required checks and reviews stay plain failures.
- **Any conflict.** Content, add/add, modify/delete, rename/rename, rename/delete, mode,
  binary, lockfiles, generated files, and type conflicts (file against symlink or directory,
  submodules), which keep their real path and mode. The build agent groups the files and
  proposes 2 to 4 options per group; span computes each option's diff from the bytes.
- **Span's risk, not the agent's.** `combine` and `keep` can be low, `ours` is at least
  medium, `theirs`, `delete` and `regenerate` are always high.
- **Nothing written without a choice, nothing published without prove.** The run holds with
  `conflict: choose how to resolve`. Choose per group, give the agent a hint, or abort
  (`git merge --abort`, with tracked leftovers put back). Applying stages through git itself,
  commits the merge and sends the run back to `prove`; the merge commit is pushed only right
  before the host merge, once prove, QA and inspect passed on it. At most three resolutions
  per run.
- **The checkout stays honest.** What the agent or a regenerate command changes is put back:
  files, the index, commits, resets, branch checkouts, the merge state, every ref, hooks,
  config and `.git/info`, the git directory first and without running git, then with hardened
  git commands (no hooks, fsmonitor, pager or `ext::`) and never through a planted symlink;
  ignored files are named, never deleted. The merge commit runs no hooks and is accepted only
  with the run's head and the base as its parents. No resolution writes or deletes through a
  symlink, including a parent that becomes one while the choice is applied.
- **A pinned repository.** The conflict phase pins `.git` and its git directories, checks their
  shape without git before running git again, and fails closed (no git, no abort) when the
  agent redirected or replaced them. Every git command span runs, ordinary commits and pushes
  included, is hardened: no fsmonitor, pager, `ext::` transport or submodule recursion. Repository
  hooks are off only during the conflict phase; span's own commits and pushes still run them.
- **Behind is not unknown.** A host refusal because the change is behind its base holds with
  its own reason; span does not update the change.
- **Regenerate, never on its own.** Lockfiles and generated files can offer a `regenerate`
  command from an allowlist of package managers and build tools, shown on the card and run
  without a shell only after it is chosen.
- **Policy.** `stages.conflict: hold|auto`, default `hold`. `auto` applies the first options
  only when span's risk for every group is low.
- **Routes.** `POST /spans/<id>/conflict/choose`, `/hint` and `/abort`; the cursor carries
  `conflict`, `resolutions` and `unpushed`, and the socket sends `conflict` frames.
### The desk survives a dropped socket, lost frames, a render crash and a failing endpoint

- **Reconnect with backoff.** The span page reopens a closed socket after 0.5 s, doubling to
  a 10 s cap with jitter, and resyncs from the replay that starts every connection. The
  backoff resets only after 5 s of a healthy connection, so a failing server is not hammered.
- **Numbered frames.** The hub stamps every live frame with `seq`, one counter per run, and
  the replay carries the `seq` it covers. The page drops frames at or below it and resyncs
  on a gap (at once for the first gap, with the backoff for the next ones within 5 s). The
  replay is sent even when the run cannot be loaded, without a span but with the
  transcript tail: the page takes the tail, keeps its baseline unless the server
  restarted, and reloads the run.
- **No repeated transcript lines.** Transcript frames carry `start` and `offset`, their
  byte range in `transcript.log`, and the replay carries the size its tail ends at.
  Flushes publish in take order. The server cuts, per connection, the lines of a flush
  the replay tail already holds and still sends the frame, so seq has no hole.
- **Write failures.** A short or failed transcript write sends only the lines written
  whole; the agent is not stopped, and `Close` reports the error.
- **No silent drops.** A subscriber whose 32-frame buffer is full is cut, and a socket
  write has a 10 s deadline: the socket ends, the page reconnects and replays, instead of
  missing frames for good.
- **Token redaction.** The replay and the live frames have `SPAN_TOKEN` replaced inside
  their string values (keys and the `type`, `stage` and `verdict` fields untouched,
  JSON-escaped tokens found); a token under 8 characters is not redacted. `SPAN_TOKEN` joins the secrets the transcript redacts.
- **Error boundary.** A render fault shows the error with **Try again** and **Reload page**
  instead of a blank app; each page has its own boundary, reset when the route changes.
- **Partial loads.** The span page loads the run, the implementation and the acceptance
  independently (`Promise.allSettled`); only an unreadable run is an error, and an older
  acceptance reply never overwrites a newer one. **Find repos** keeps the repositories
  when Jira fails and the reverse. A malformed live frame is ignored.

### Repairs prove only what the change can affect

- **A focused band from the diff.** When neither the criteria nor `prove.focused` name one
  and a baseline was recorded, prove derives it from `git diff <base>...HEAD`: `go test` runs the changed packages and
  every package that imports them, test-only imports included; `vitest` runs
  `related --run` and `jest` runs `--findRelatedTests` on the changed files. Anything it
  cannot reason about (a deleted or renamed path, `go.mod`, a `.md` inside a package, config
  files, lockfiles, assets, `-args`, more than 200 files, an unknown runner, Python, no
  baseline) keeps the wide band deciding as before.
- **The wide band still guards.** It runs once per commit after the focused band passes, and
  a wide failure the base commit did not have blocks and spends a repair. The fingerprint
  ignores the narrowed package list, so the same failure twice is still `no progress`.
- **Off switch.** `prove.diff: false` turns the derivation off.

### A second Ctrl-C ends every command Span started

- **Graceful first, forced second.** The first interrupt still cancels and gives each command
  its grace period. A second one kills every tracked process tree at once and exits with 130,
  so an agent or test runner cannot keep editing the tree after Span is gone. Before, the
  second Ctrl-C killed Span and left them running.
- **Windows builds again.** The process check moved to `proc.Alive`, which has a Windows
  implementation; `internal/agent` no longer calls `syscall.Kill` directly.

### Parallel stories run side by side instead of queueing

- **The lock follows the tree.** A second story launched from the page while a checkout is
  held no longer ends in a hold: it gets its own git worktree under `.span/worktrees/` and
  starts at once. Nothing is written into the busy checkout. A free checkout is still edited
  in place, and the CLI releases the checkout as soon as its run has a worktree, so the next
  story is not blocked for the whole run.
- **`prove.prepare`.** A command, for example `pnpm install --frozen-lockfile`, that runs once
  in a run's own worktree before its baseline and first build. It never runs in your checkout.
- **No baseline on an unprepared tree.** If `prepare` fails, or a worktree has a `package.json`
  and no `node_modules` and no `prepare`, no baseline is recorded and the transcript says why.
  Otherwise the environment would look broken on the base commit and every later failure
  would be excused.

### A failure the base commit already had does not spend a repair

- **Baseline.** Before the first build, the run executes once, on the base commit, the
  `fast`, `acceptance`, and `qa.command` lines this story would use, and stores what failed in
  `.span/runs/<id>/baseline.json`: per command, the exit code and the failure lines as a
  multiset. Only colors, parenthesised durations, and the `go test` package timing column are
  removed; values inside messages are kept. `go test` output, with or without `-v`, is read
  per test and package: every line a failing test printed is included, a passing test's log
  is not. A package with `[build failed]` or `[setup failed]` is never excused. Focused commands are not
  recorded: they depend on the diff. A checkout with uncommitted changes gets no baseline,
  and the frame prompt now says not to edit files.
- **Judged as fail-to-pass and pass-to-pass.** A failing command is excused only when it
  also failed on the base commit and every failure line it prints now was already there, at
  least as many times. It is kept on the statement with `preexisting: true`, the reason
  counts it, and it never reaches the repair loop or the next build prompt. One new line, no
  comparable line, an output cut at the capture limit (`prove.Output.Truncated`) on either
  side, or no record for that exact command fails exactly as before.
- **QA.** An excused QA exit is judged as exit 0: without a per-test report the criteria pass
  on it; with one, each criterion still decides. An inconclusive result still wins.
- The baseline survives rewind and resume. A run already past build gets none and is judged
  as before. A command that does not start never holds the run; a cancelled baseline stops
  the run before the build agent starts.
- **Cost:** one extra run of those commands per run. `prove.baseline: false` turns it off.

### One driver per run, one run per checkout, one run per click

- **Run lock.** `Resume`, `Continue`, `Reload`, and `Rewind` take a non-blocking `flock` on
  `.span/runs/<id>/lock`. `serve` plus `span resume` on one run, or two `serve` processes,
  no longer race on `cursor.json`; the second is refused with the holder's pid. The kernel
  frees the lock when the process dies, so a crash leaves nothing to clean up.
- **Checkout lock.** A run takes `<workspace>/.span/workspace.lock` when work starts. A
  second run in the same checkout holds with `another run is active in this checkout`
  instead of checking out its branch under the first and committing its files. Resume it
  from the page once the other run ends. `beginOne` refuses before writing anything.
- **Double submit.** `POST /quickstart` starts one run for identical submissions inside
  10 s, or for one `Idempotency-Key` inside 10 min. A refused launch is not remembered, so
  a retry after fixing the cause works.
- **Every repo's run is reported.** The response keeps `id` and `stage` (the first run)
  and adds `runs: [{id, stage, repo}]`. If repo N fails to start, the runs already started
  are returned with a note instead of being dropped.
- A busy run or checkout answers `409 busy` with the holder named, and a wake from the page
  is refused up front when `span resume` holds the run, so answers are not silently lost.
- If repo N cannot start, the other repos still do. The response sets `partial: true`, is
  not remembered by the double-submit window, and lists what failed in `notes`.
- A filesystem without `flock` (some network mounts) runs unlocked instead of refusing.
- **The checkout is taken before anything is written into it.** `beginOne` claims
  `.span/workspace.lock` first, then configures the project, and hands the claim to the run
  (`Options.HeldWorkspace`). Before, two launches could both pass a check and the second
  rewrote `span.yaml` and the brief under the run that was starting.
- **A resumed run drops the "checkout is busy" reason** instead of showing it while it runs.
- **`<workspace>/.span` ignores itself** whatever the state root is, so the lock files no
  longer show up as untracked in the project.
- **A reused `Idempotency-Key` with another story is refused** (`422 idempotency_mismatch`)
  instead of replaying the old run. The same key with the same story still replays it.
- **The desk shows every run a submission started.** With several repos, or a partial start,
  it stays on the desk and lists each run with its repo and what failed to start. A run held
  because the checkout was taken says "Resume when the other run ends".
- Windows has no `flock` in the standard library, so the two locks are not enforced there.

### Stop ends the whole process tree, and the local server refuses other sites

- **Stop works from the first frame.** A run started from the page now registers its stop
  handle through `live.Hub.Fly`, the same path a woken run uses. Before, Stop answered
  "nothing is running this span" until the run had held once.
- **Cancel reaches the grandchildren.** `proc.Tame` puts every agent, check, and release
  command in its own process group. Cancel sends SIGTERM to the group and SIGKILL after
  `proc.Grace` (5 s); `Wait` is bounded so an orphan holding the pipes cannot hang it.
- **Ctrl-C still stops everything.** Because those groups no longer share the terminal's,
  `run`, `resume`, and `serve` catch SIGINT/SIGTERM and cancel their flights first. A
  second Ctrl-C kills the process at once.
- **No cross-site POST.** The server wraps its routes in `http.CrossOriginProtection`, so a
  page on another site cannot start an agent on `127.0.0.1`. Without `SPAN_TOKEN`, `serve`
  also refuses any `Host` that is not `localhost` or a loopback IP (DNS rebinding). The
  token is compared in constant time.

### A failing check returns to build with its evidence

The engine no longer throws a failing check's output away. `adapt.Exec` captures both
streams (the last 1 MiB), writes the whole capture to `.span/runs/<id>/iter/<n>/logs/`,
and keeps a 4 KiB tail on the statement. The next build prompt carries the command, the
exit code, the tail, the log path, and the attempt counter.

`prove` and `qa` share one repair edge (`engine.repair`): a failure returns to `build`,
not `frame`, and `qa` no longer holds with no way out. Three stops bound the loop:

- `no progress`: the same failure fingerprint twice in a row. The fingerprint is the
  command plus the lines that name what broke, computed with no model.
- `inconclusive`: the same command passed and failed on one commit, so the result is not
  attributable to the change.
- `iteration cap`: `repair.maxAttempts`, 1 to 8, default 3. There is no unlimited mode.

`Cursor.repairs` is the durable ledger of attempts, written before the next build starts.

### A process exit no longer marks every criterion failed

`qa.Judge` is now a table over evidence. An id is `fail` only when a per-test report names
the test that ran and failed. `qa.report` accepts JUnit XML or Vitest/Jest JSON; with no
path set, Span looks in `test-results/junit.xml`, `junit.xml`, `test-results/vitest.json`,
`test-results/results.json`, `test-results.json`, and `report.json`. A report older than
the command is ignored.

Without a report, a zero exit marks declared ids `pass` and a non-zero exit marks them
`unverified`. The previous behavior — 22 criteria painted red because one process exited
1 — is gone, and a test pins it.

### QA does not repeat the fast suite

When `prove` already ran that exact command on this commit in a clean tree, the gate reads
that result and checks only the declarations. A dirty tree or a different commit runs it
again. The gate records `evidence` as `prove`, `junit`, `vitest-json`, or `exit-only`.

### Four agents, one registry, and a picker on the page

Cursor, Claude, Codex, and Pi are first-class. `internal/agent/registry.go` is a single
table: id, label, binary, headless arguments, and the marker that proves that harness has
Gentle AI. Detection, selection, invocation, the policy check, and the page all read it,
so adding an agent no longer means editing four switches.

- Pi runs as `pi --print --approve`: one shot, and project-local files trusted because
  nobody is present to answer a prompt.
- `agent:` accepts the id, the legacy binary name (`cursor-agent`), `auto`, and `any`.
  `Validate` rejects anything else and names the allowed values.
- `GET /agent?path=<folder>` and `POST /agent` back an **Agent** picker on the desk. It
  lists all four with the reason each one cannot run, and saving rewrites the one
  `agent:` line in `span.yaml`, comments intact, after validating the result.
- A requested agent that cannot run now says why: `<binary> is not on PATH` or
  `<Label> does not have Gentle AI installed`.

### Parallel runs are isolated in git worktrees

`internal/worktree` holds the whole surface Span uses — `add` (new or existing branch),
`list --porcelain`, `remove --force`, `prune` — and it is implemented, not inferred. `span
run --branch <name> --worktree` and `quickstart` when it derives a branch check the run out
at `<repo>/.span/worktrees/<branch>`; the state directory and `span.yaml` stay in the
repository, and the shared checkout is never switched. Reusing a path lands on the same
tree, and `remove` keeps the branch.

A run in a worktree is recorded as such (`cursor.worktree`) and the build prompt states the
rules: the branch is already checked out, do not switch, do not stash, `Span commits`.

### A commit belongs to the person the checkout is configured with

`adapt.Git` forced `GIT_AUTHOR_NAME=Span` and `GIT_COMMITTER_NAME=Span` on every git
invocation, so every commit was attributed to a fictional identity. The overrides are gone:
the author and committer are the configured `user.name` and `user.email`, nothing is added
as a co-author, and the message still names the session (`feat(<feature>): span <id>`) so
traceability does not depend on authorship.

### The flow adapts to the weight of the story

`run.weight` is `light`, `standard`, or `deep`, derived from the story unless declared:
criteria count and whether the handoff is a design file. A fix earns two repairs and skips
the whole suite; a module with twenty criteria and a design handoff earns six repairs and
the wide suite once per commit. The weight is read as the run walks, so criteria installed
by frame count. A prove pass is never left with no command: a light story with nothing to
focus on falls back to the whole suite.

### Restart from a stage, and stop what is running

**Restart from a stage** rewinds a run to any stage that does work, with an instruction that
stage reads when it starts. The cursor moves there, the iteration goes up by one so the
earlier attempt stays on disk, and the repair ledger, `wideSHA`, the agent block, and open
questions are cleared: a deliberate restart gets its own budget, which is why
`repair.maxAttempts` now counts repairs rather than iterations (`Cursor.Repairs` is the
ledger, cleared by a restart). A restart refuses to run beside a live stage.

**Stop now** cancels the flight. `live.Hub` keeps the cancel handle for each run that the
server launched, `POST /spans/{id}/stop` fires it, and the engine's `run` wrapper turns the
cancelled loop into a hold with the reason, so a stopped run is never left looking idle.

### The check runs the story's tests, not the whole repository

The band is scoped to the test files that declare this story's acceptance criteria, a map
Span already built for the QA gate. It is derived from the wide command (`npm test` becomes
`npm test -- <files>`, and the same for vitest, jest, pytest, and playwright), so a project
that configures nothing still gets its own tests. `prove.focused` with `{files}` overrides
the derivation, and an unknown runner keeps the wide band deciding. When a focused band
is configured it blocks and the wide `fast` list only reports, so a failure in another
module cannot spend a repair attempt or paint the criteria. Without `focused`, behavior is
unchanged.

The wide suite runs once per commit (`Cursor.wideSHA`): the same bytes cannot produce a
different answer, so a repair that commits again pays for it and one that does not, doesn't.

Two defects found while doing it:

- `qa.testFile` did not recognize `*.test.tsx` (the jest and vitest convention), only
  `*.spec.*` and `*_test.*`. A story whose declarations live in `statement.test.tsx`
  reported zero declarations.
- A wide-suite failure is recorded with its own band, so the log and the page can show it
  without it being the verdict.

### The agent can be chosen per stage, and the flow can be paused for it

On a started span the ledger tab offers **Agent for a stage of this run**: frame, build,
or inspect, and one of the four CLIs or the policy default. The choice is written to
`.span/runs/<id>/agents.json` and read at each stage boundary, so it applies the next time
that stage starts. Resolution order per stage: the run's choice, then `agent:` from the
policy, then the first ready CLI.

**Pause after this stage** writes `.span/runs/<id>/pause`. The runner consumes it at the
next stage boundary and holds with `paused before <stage>`, which is the moment a person
needs to change the model. Resume discards a pause written while the run was already
stopped.

Both requests are files, not cursor fields, because the cursor belongs to the runner: a
cursor written by the page would be erased by the runner's next save. `POST
/spans/{id}/agent` and `POST /spans/{id}/pause` back the controls, and the snapshot
carries `stageAgents`.

Fixed while doing it: the page offered a `Resume` button on a run that was still running.

### An agent that runs out of quota or credentials is named

`agent.Classify` reads the CLI's own output against a curated phrase list and returns a
typed `*agent.Blocked`. The engine records `Cursor.agentBlock` with the kind (`quota` or
`credentials`), the CLI, and the matching line, then holds with the action that fixes it:
resume after the limit resets or switch CLI, or log in. No model, no extra process. An
ordinary failure that happens to print a status code stays an ordinary failure.

### The page shows the evidence and always offers an action

A held stage renders the failing command, its exit code, the output tail in a monospace
block with scroll, the attempt counter, a sentence when the result is inconclusive, and
the log path. Every hold offers exactly one action: `Fix and retry`, `Re-verify`,
`Answer and continue`, or `Resume`. A blocked agent gets its own banner above it.

### Files

- `internal/adapt/adapt.go`: `Exec.Run` returns `prove.Output`, `OutputCap`, `tailBuffer`.
- `internal/prove/prove.go`: `Output`, `Checker`, `Fingerprint`, `Report.Fingerprint`.
- `internal/qa/qa.go`, `internal/qa/report.go`: the `Judge` table, `Outcomes`, `Find`, JUnit
  and JSON parsing, `Gate.Evidence` and `Gate.Report`.
- `internal/statement/statement.go`: `Command.Output/LogPath/Fingerprint`,
  `Statement.Fingerprint/Inconclusive/Cap`, `Tail`, `Signature`.
- `internal/engine/engine.go`: `Cursor.Repairs`, `Cursor.AgentBlock`, `repair`,
  `failurePacket`, `agentBlocked`, QA evidence reuse, shared `stamp`/`writeLog`.
- `internal/agent/limit.go`: `Classify`, `Blocked`, `AsBlocked`, `tailOutput`.
- `internal/qa/qa.go`: `Scope`, and `.test.tsx` recognized as a test file.
- `internal/prove/prove.go`: `Plan`, `DeriveFocus`, a blocking focused band, a reporting wide band.
- `internal/agent/registry.go`: `Spec`, `Specs`, `Find`, `Known`, `Report`, `Available`, `Home`.
- `internal/engine/agents.go`: `AgentStages`, `UsesAgent`, `ReadStageAgents`, `SetStageAgent`, `RequestPause`, `pick`.
- `internal/engine/rewind.go`: `Rewind`, and the cursor file helpers it shares.
- `internal/policy/weight.go`: `Derive`, `RepairCapFor`, `WideOn`.
- `internal/worktree/worktree.go`: `Add`, `List`, `At`, `Remove`, `Prune`, `Path`.
- `internal/adapt/adapt.go`: `IsWorktree`, `Git`.
- `internal/live/live.go`: `WatchStop`, `Stop`.
- `web/src/components/rewind/Rewind.tsx`.
- `web/src/components/stage-agent/StageAgent.tsx`, `web/src/lib/stage-agents.ts`.
- `internal/serve/agentcfg.go`: `GET /agent`, `POST /agent`.
- `web/src/components/agent-picker/AgentPicker.tsx`.
- `internal/policy/policy.go`: `repair.maxAttempts`, `qa.report`.
- `web/src/lib/failure.ts`, `web/src/lib/agent-block.ts`,
  `web/src/components/failure/FailureCard.tsx`, `web/src/pages/SpanPage.tsx`.
- `USAGE.md`, `README.md`, `span.example.yaml`.
