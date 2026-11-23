# conflict-resolution

A phase of the merge stage that resolves any git conflict: an agent reads both sides and the
story, proposes options per conflict group, the person chooses on cards, and nothing applies
without a choice. Branch: slice/09-conflict-resolution. Test runners: `go test -race ./...`,
`npx vitest run` (web). Delivery: ask-on-risk.

## Specs

S1. "Pues quiero que esa feature sea capaz de resolver cualquier tipo de conflicto no este en concreto"

S2. "pudiese ser que exista una fase o una opcion queda a tu criterio que se use agenticamente para poder resolver conflictos de forma rapida entiendo claramente muy bien el problema del conflicto y plantear opciones para solucionarlo tipos un flujo o unas cards de opciones"

S3. A conflict is detected before the merge is attempted, and a failed `gh pr merge` whose
stderr says the change cannot be merged or has conflicts is classified as a conflict, never
as "outcome unknown".

S4. Any kind of conflict is covered: both-modified content, add/add, modify/delete,
rename/rename and rename/delete, mode, binary, lockfiles and generated files. Each group of
files gets 2 to 4 options, each with a title, a summary, a risk (`low|medium|high`) and the
unified diff of what it would write.

S5. The base is integrated into the run's branch with `git merge --no-commit --no-ff
origin/<base>` in the run's own worktree. Never a rebase, never a force push.

S6. Nothing is written to the branch until the person chooses. The person can also give the
agent a hint (free text) to propose again, or abort, which runs `git merge --abort` and leaves
the branch exactly as it was.

S7. Applying a choice commits the merge, pushes it, and sends the run back to `prove`
(the code changed), so the sealed prove/qa/inspect statements never describe a different
head. Then the merge is attempted again.

S8. Lockfiles and generated files offer a `regenerate` option that runs a command the person
can read on the card before choosing it; it is never run on its own.

S9. Policy `stages.conflict: hold|auto`, default `hold`. `auto` applies the agent's first
option only when every group of the conflict is `low` risk and prove passes afterwards;
otherwise it holds like `hold`.

## Contract (shared by the Go and web units; do not change it without telling the parent)

`Cursor.Conflict` (JSON `conflict`, omitted when absent):

```
Conflict      { base, baseSHA, state: "proposing"|"choosing"|"applying"|"failed", note, groups[] }
ConflictGroup { id, kind: "content"|"add-add"|"modify-delete"|"rename"|"mode"|"binary"|"lockfile"|"generated"|"other",
                files[], summary, options[], chosen }
ConflictOption{ id, title, summary, risk: "low"|"medium"|"high",
                action: "ours"|"theirs"|"combine"|"regenerate"|"delete"|"keep",
                command, diff }
```

`command` is only set for `regenerate`. `chosen` is the option id once picked.

Live event: `{ "type": "conflict", "conflict": <Conflict> }` on every state change.

Endpoints (form-encoded, same auth as the other POSTs):
- `POST /spans/{id}/conflict/choose` with `group=<id>&option=<id>` repeated, one pair per
  group. Rejected unless every group has a pair and every option id exists.
- `POST /spans/{id}/conflict/hint` with `text=<free text>`: the agent proposes again.
- `POST /spans/{id}/conflict/abort`: `git merge --abort`, the conflict is cleared and the run
  holds with the reason "conflict: aborted".

The run holds with reason `conflict: choose how to resolve` while `state` is `choosing`.

## Tasks

| ID | Spec | Route | What |
| --- | --- | --- | --- |
| T1 | S3, S4, S5, S6, S7, S8, S9 | delegated (parallel unit, Go) | adapt: integrate/abort/apply/finish + conflict kinds + gh stderr classifier; engine: merge-stage phase, agent proposal, apply, return to prove; serve: three endpoints and the event; policy `stages.conflict`; docs |
| T2 | S2, S4, S6, S7 | delegated (parallel unit, web) | web: types, api parsing, option cards, hint and abort controls, wiring in SpanPage |

## Log

L1. 2026-11-23 "Por alguna razon ahora mis sesiones me estan arrojand esto: open /Volumes/External/dev/salute/salute-hermes/span.yaml: no such file or directory. No estoy pudiendo relanzar ni evaluar los merge que se hacen y aparte tengo un conflicto en la SAL-3 lo que me da una idea y es que pudiese ser que exista una fase o una opcion queda a tu criterio que se use agenticamente para poder resolver conflictos de forma rapida entiendo claramente muy bien el problema del conflicto y plantear opciones para solucionarlo tipos un flujo o unas cards de opciones"

L2. 2026-11-23 "Pues quiero que esa feature sea capaz de resolver cualquier tipo de conflicto no este en concreto"

L3. Map (read-only): no conflict detection exists; `doMerge` is engine.go:1071-1095; a conflicting `gh pr merge` reads "merge outcome unknown"; Source port has no fetch/merge; `Waiting()` is tied to Questions so a choice needs its own wait kind; `already()` skips stages with an advance statement, so applying a resolution must rewind to prove.
