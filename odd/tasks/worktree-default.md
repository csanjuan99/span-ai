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
S9. A new worktree prepares itself with nothing configured: installs detected from lockfiles, ignored `.env` files copied; `prove.prepare` replaces detection; delete discards only what loses nothing.

Assumptions (not requested): no span.yaml `workspace` setting, no UI toggle for in-place, no automatic cleanup on merge.

## Contract

Consumer: web desk and CLI/script users. Auth: unchanged. New optional input only (`inPlace`, default false); responses and error codes unchanged. Success: `POST /spans` with `story` → 202 `{id,...}`, run in `.span/worktrees/<branch>`. Error: `GET /start/precheck?inPlace=true` on a busy checkout → refusal 16 `checkout_busy`.

## Tasks

- T1 S1-S5, S8 (launch) · inline · done (d16151b) · RED: build failed on placement_test; GREEN: go vet + go test ./... pass
- T2 S6 · inline · done · RED/GREEN checkout_test · f98ce44
- T3 S7, S8 (delete) · inline · done · delete_test · f98ce44
- T4 independent verification · delegated (high risk items 3, 4) · done: S1-S8 after two corrections; B1 rechecked with 520 races
- T5 S9 · inline · done · RED/GREEN autoprepare_test, localfiles_test, delete_test; verified (five defects fixed and rechecked) · f98ce44

## Log

L1. "Usa opus 5.5 como Adivsor. Span no esta usando worktrees por defecto. Me paso que al iniciar dos historias en un mismo proyecto detecto que otro estaba toqueteando el repo, valide y no tenia worktree solo rama y entonces la sesion me dijo esperar hasta que el otro run termine"
L2. 2026-11-26 "Si. Lo ideal es que hagas o sea compatible con lo que ya hay en la UI y en el CLI." (answer to: worktree always by default, `--in-place` as exception)
L3. T2/T3 commit blocked: the pre-commit review reviews whole files and fails on code in engine.go that the change did not touch.
L4. 2026-11-26 verification found S5, S6, S7 failing and regressions R1-R3; one correction batch applied (launch lock, owner records, SamePath, strict delete, repo-root-only trees, cleanup on failed run). Recheck pending.
L5. Recheck: B2-B4 and R1-R3 pass; B1 still failed (the losing launch removed the winner tree). Second correction: look-and-create under the launch lock; RED then GREEN (TestALostLaunchKeepsTheTreeOfTheWinner, 10x stable).
L6. B1 recheck passed. go vet and go test ./... pass.
L7. 2026-11-26 user: "Arregla esta deuda ahora." and "Importante que la funcionalidad de worktree funcione sin mi. Es decir por que tengo que configurar algun prove.prepare y no se hace solo todo lo necesario?" -> S9: a new worktree prepares itself (detected installs, ignored .env copied); delete discards what is rebuildable.
L8. Verification of S9 found: committable .env, write through symlinked folder, write-back through symlinked .env, commits lost on delete with detached HEAD, half-installed tree trusted on resume. All fixed; recheck passed.
L9. 2026-11-26 user approved ("Si hazla", "Si") further debt rounds. Committed f98ce44 (S5-S7, S9) and 32d21dc (Jira login and repository name read from the project, found by the pre-commit review). go vet and go test ./... pass. Not pushed. Fixes made after the S9 recheck are covered by tests only.
L10. 2026-11-26 user: "Si, ajustalo luego commit y push y luego pr a main" -> the merge stage holds when the target branch lookup fails (open or already merged).
