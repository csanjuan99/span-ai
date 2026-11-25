# Start a run (N1–N4) and Settings Connection (P6)

Branch: feat/start-a-run · Delivery: parent commits · Checks: `go test ./...`, `cd web && npx tsc -b && npx eslint src && npx vitest run`

## Specs

S1. Replace the desk first-version form (`Setup`, long form, `$ settings` Show/Hide) with N1–N4 per `handoffs/CURSOR-BRIEF.md` and canvas.json notes n1n–n4n, p2n8.
S2. P6: Connection section on `ProjectSettings` (server, health test, access token tab-only); Jira Close session; remove `Settings.tsx` from desk.
S3. API-first: new routes/verbs for prechec, what-will-happen, shaped sealed list, run key, health; refusal codes 6,8,14–18; USAGE.md; tests.
S4. Sample brief one-click test stays green.

## Tasks

- T1 S1-S3 delegated · backend start package, serve, CLI
- T2 S1-S2 · web StartRun flow, Desk, P6
- T3 S4 · tests and verification

## Log

L1. Read handoffs/CURSOR-BRIEF.md in this worktree and execute it completely.