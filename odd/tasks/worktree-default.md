# Worktree by default

Branch: feat/worktree-default · Delivery: ask-on-risk · Checks: `go test ./...`, `go vet ./...`, `cd web && npx vitest run` (only if web changes) · Test-first: yes

## Specs

S1. A run that has a branch works in its own git worktree (`.span/worktrees/<branch>`) from every launch path: `POST /spans`, `POST /quickstart`, `span quickstart`, `span run --branch`. One decision for all of them. The person's checkout is not switched.
S2. Compatibility: if the checkout is already on the run's branch it is used in place (git cannot make a second tree). A project that is not a git repository keeps the old behaviour. `span run` without `--branch` stays in place. `--worktree` keeps working. Existing runs keep the tree recorded in their cursor.
S3. Explicit in-place opt-out with CLI/API parity: `--in-place` on `span run` and `span quickstart`; form field `inPlace=true` on the start routes; `inPlace=true` query on `GET /start/precheck`; `--in-place` on `span start precheck`. `--worktree` with `--in-place` is a usage error (exit 2).
S4. A busy checkout no longer refuses a launch that will get its own worktree. Refusal 16 `checkout_busy` stays for launches that would run in place.
S5. A launch does not share a branch with an unfinished run: it is refused as before ("already belongs to another run").
S6. An in-place run does not switch branches over uncommitted work that belongs to another unfinished in-place run; it holds and says which run.
S7. Deleting a run removes its own worktree only when it has no uncommitted changes. The branch always stays.
S8. Help text, README and UI copy describe the new default.

Assumptions (not requested): no span.yaml `workspace` setting, no `.env` copying, no UI toggle for in-place, no automatic cleanup on merge.

## Contract

Consumer: web desk and CLI/script users. Auth: unchanged. New optional input only (`inPlace`, default false); responses and error codes unchanged. Success: `POST /spans` with `story` → 202 `{id,...}`, run in `.span/worktrees/<branch>`. Error: `GET /start/precheck?inPlace=true` on a busy checkout → refusal 16 `checkout_busy`.

## Tasks

- T1 S1-S5, S8 (launch) · inline · done (feat(start) commit on this branch) · RED: build failed on placement_test; GREEN: go vet + go test ./... pass
- T2 S6 · inline · pending
- T3 S7, S8 (delete) · inline · pending
- T4 independent verification · delegated (high risk items 3, 4) · pending

## Log

L1. "Usa opus 5.5 como Adivsor. Span no esta usando worktrees por defecto. Me paso que al iniciar dos historias en un mismo proyecto detecto que otro estaba toqueteando el repo, valide y no tenia worktree solo rama y entonces la sesion me dijo esperar hasta que el otro run termine"
L2. 2026-11-26 "Si. Lo ideal es que hagas o sea compatible con lo que ya hay en la UI y en el CLI." (answer to: worktree always by default, `--in-place` as exception)
