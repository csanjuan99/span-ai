# Span engine

Current product use is [USAGE.md](../../USAGE.md). This file is the original engine contract. Quickstart, repo choice, per-project Jira, branches, and handoffs are specified there.

Objective: a local or remote Go binary that runs one feature from brief to deploy, with reports as the control surface.
Branch: main
Delivery: single repo, no hosted UI
Test runner: `go test ./...`

## Specs

S1. The tool lives at `dev/span` and builds one Go binary, `span`.

S2. A span moves through intake, frame, build, prove, inspect, open, merge, promote, and note. Each stage writes a statement before the next stage starts.

S3. Verdicts are `advance`, `hold`, and `stop`. Promote and merge do not run unless the previous statement is `advance`.

S4. Build is Gentle AI ODD. The agent prompt says ODD is mandatory. After Build, `odd/tasks/<feature>.md` must contain a heading, `## Specs`, `## Tasks`, and `## Log`. Missing that document stops the span before any change request.

S5. Prove runs the project commands in two bands, `fast` then `acceptance`, on the recorded git sha. A fast failure does not start acceptance. A command that is prose, or that cannot be started, is `hold`. A command that runs and fails is attributable and starts another iteration at frame, up to 3. The fourth attributable failure is `hold`.

S6. Each stage can read extra instructions from `span/stages/<stage>.md` in the project and `.span/runs/<id>/instructions/<stage>.md`. Their digest is stored on the statement. They do not authorize push, merge, or deploy.

S7. Files added under `.span/runs/<id>/handoffs/` are hashed into the next frame. They are not read mid-stage.

S8. A killed process or a dropped call resumes from the cursor. Completed `advance` stages are not repeated. Open, merge, and promote do not repeat an effect that already exists. If a remote effect was started and the port cannot say whether it happened, the span holds.

S9. `span serve` shows statements and accepts instruction writes. A non-loopback address without `SPAN_TOKEN` refuses to start.

S10. Build the project in the dev folder and create a folder named span.

## Tasks

- T1. S1-S3. Engine, statements, cursor. Route: inline.
- T2. S4-S8. Build gate, prove, iteration, resume, idempotent effects. Route: inline.
- T3. S9. Serve. Route: inline.

## Log

L1. Build the project in the dev folder and create a folder named span.
