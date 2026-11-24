# resilient-blockers

Objective: Span classifies blockers, asks the human structured questions, keeps corrections durable, and pauses at the right gates. Branch: feat/resilient-blockers. Delivery: ask-on-risk, one commit per task. Runner: `go test ./...`.

## Specs

S1. A commit rejected by the repository's hook is a classified blocker (`commit-rejected`), not a plain hold. The hook output is kept on the cursor and the statement, and the next build prompt carries it.
S2. The same rejection output twice is non-retryable: resuming does not run the build agent again. It only retries the commit, and holds again with the same fingerprint when nothing changed.
S3. A person's rewind clears the blocker, so a changed brief or note gets a fresh agent run.
S4. Any stage can raise a structured decision request (title, what happened, evidence, 2-4 options with consequence, recommendation, free text), and answering it resumes at the right stage. A build agent that stops to ask is detected, not counted as a finished build. There is no option that bypasses the hook.
S5. Decisions go to an append-only log with their full text. A second note no longer replaces the first. Decisions are applied on top of the brief on every build, with provenance, and a decision survives `Reload`.
S6. Before build, the brief is checked against the repo's `AGENTS.md`; a contradiction raises a decision request. After frame, a plan pause fires only on conflict, ambiguity or risk. `merge` always holds.
S7. Each behavior has a regression test, including the incident replay (brief contradicts hook, one request, answer persists).

## Tasks

- [x] T1 (S1-S3, S7) classify commit rejection, cap the loop. inline. RED: build failure on missing Blocker, GREEN: 3 engine tests + 3 unit tests, full suite ok
- [x] T2 (S4, S7) structured decision request for any stage. inline. RED: compile failure on Cursor.Decision; GREEN: 7 engine tests, 4 decision tests, 2 serve tests, 3 web tests. Full suites ok
- [x] T3 (S5, S7) decision log and overlay on the brief. inline. RED: compile failure on Input.Decisions/BriefEpoch; GREEN: 5 overlay tests, 5 engine tests, 1 serve test. Full suite ok
- [x] T4 (S6, S7) preflight against AGENTS.md and conditional gates. inline. RED: compile failure on decision.Clash/Stages.Plan; GREEN: 7 engine gate tests, 3 decision tests, 1 policy test. Full suite ok

## Log

L1 (2026-11-24, user): "Si, implementa en ese orden con odd, luego haces judgment day y luego haces revisión rdd"
L3: T1 landed in 1c3a6ba. The pre-commit hook (GGA) rejected the first T1 commit for a duplicated test helper; fixed, not bypassed.
L4: S5 design note: story drift is detected by BriefEpoch (bumped only when Reload brings different text), not by hashing brief.md, because frame and answers also edit brief.md and would raise false drift.
L5: S6 deviation, on purpose: `merge` keeps its default of auto. Flipping it to hold would change what existing span.yaml files do (a consumed config contract, high-risk item 3). The plan gate is new and defaults to auto for the same reason; the creator's span.yaml can set plan: conditional and merge: hold.
L2: Route: all inline, tasks depend on each other so no writer trigger holds. Risk lines per task.
