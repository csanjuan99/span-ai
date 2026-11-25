# run-page-redesign

Rebuild the run page around the Span AI design: what needs the person comes first, with what happens next stated beside the action. Branch: feat/design-tokens. Delivery: ask-on-risk. Test runner: `npm test` in `web/`, plus `npx tsc --noEmit` and `npm run build`.

## Specs

S1. "Hay que evaluar lo bueno de esa propuesta y si no esta soportado SOPORTARLO. Ahi es donde esta el valor. Poder ingresar cosas neuva que faciliten el trabajo y sea intuitivo."

S2. One "needs you" block sits above the tabs and holds whatever the run waits on: questions (`Waiting`), a decision (`Decision`), a failed check (`FailureCard`), a merge conflict (`ConflictPanel`), an approval (`approvalOf`), a blocked agent (`agentBlockLine`), a busy checkout (`isCheckoutBusy`). Today the questions and the decision hide inside the Ledger tab. Each case states what happens if the person acts.

S3. The stage rail shows the ten real stages as pills with the `+`, `>` and `·` marks kept, and the time each sealed stage took, derived from `Statement.time`. The header shows the issue, a status badge and the branch.

S4. Run controls (agent per stage, note, restart, pause and stop) and "Safe and kept" sit beside the main column. Pause and Stop render only while the run is active; Restart is unavailable while a merge conflict is open.

S5. A proof summary (verdict, diff totals) uses the existing `buildReport`. A test count ("41 passed · 2 failed") needs backend work and is its own task.

S6. A visible theme toggle exists; the `d` key keeps working.

S7. Mobile (390px) keeps one column with the "needs you" block first.

## Tasks

| ID | Spec | Route | What | Commit |
| --- | --- | --- | --- | --- |
| T1 | S3, S6 | inline | Shell and StageRail on the tokens: pills, durations, theme toggle | done, see git log |
| T2 | S2 | inline | `NeedsYou` block lifted out of the Ledger tab, covering every case | done, see git log |
| T3 | S4 | inline | Run controls and "Safe and kept" aside | done, see git log |
| T4 | S5 | inline | Proof summary from `buildReport` | done, see git log |
| T5 | S5 | deferred | Backend: test counts from the prove gate (own feature) | |
| T7 | S8 | inline | Running card while the run goes on by itself | done, see git log |
| T8 | S8 | inline | Done card: where the work went and the change link (Jira link needs the backend: no Jira URL on the page) | done, see git log |
| T9 | S8 | inline | Relative times beside held and waiting | done, see git log |
| T10 | S8 | inline | Transcript and command log as closed code-view entries (no dialog primitive exists, so they open in place) | done, see git log |
| T11 | S8 | inline | Required marker on open questions | |
| T12 | S8 | deferred | Backend: commit count | |
| T13 | S8 | deferred | Backend: expose the run holding the checkout | |
| T14 | S8 | deferred | Backend: note and resume in one call | |
| T6 | S7 | inline | Mobile layout, in the order the design draws it | done, see git log |

S8. The page matches the design's desktop artboards S1 to S9. What the code already supports is drawn as is; what it does not is built, in this order: frontend-only first, backend after.

## Design against code (checked 2026-11-24)

| Design element | In code today | Needs | Task |
| --- | --- | --- | --- |
| Running card, "Nothing needed from you", Pause after this stage (S1) | no block while the run goes on | frontend | T7 |
| Done state: Open story in Jira, Open merge commit, Files changed, Risks, Run details (S6) | `changeURL` exists in the cursor; no done block | frontend | T8 |
| Relative times: "held 14:36 · 16 min ago", "waiting since 14:19" | statements carry `time`; no relative text | frontend | T9 |
| Agent transcript and Command log as entries that open a code view (S2 to S9) | transcript is a collapsed list in Implementation; commands are in the ledger | frontend | T10 |
| Question fields marked "Needs an answer / Required" (S3) | `Waiting` has no required marker | frontend | T11 |
| "Held at prove: 41 passed · 2 failed" test counts (S2) | none; the gate records checks, not tests | backend | T5 |
| "3 commits on <branch>" in Safe and kept | no commit count recorded | backend | T12 |
| "Open the other run" on a busy checkout (S9) | the holder run id is not exposed | backend | T13 |
| "Leave a note" and resume as one action | two calls today | backend | T14 |
| "If you fix and retry, it touches N files" prediction | cannot be known | dropped | none |

## Log

L1. User request, verbatim: "Es que hay que entender que la UX y UI es nueva y no necesariamente tenemos que reemplazarla directamente con lo del codigo. Hay que evaluar lo bueno de esa propuesta y si no esta soportado SOPORTARLO. Ahi es donde esta el valor. Poder ingresar cosas neuva que faciliten el trabajo y sea intuitivo. Con eso en mente en que procedemos?"

L2. 2026-11-24: foundation committed as 3787bc6 (tokens, Geist, ThemeProvider mounted, DESIGN.md and PRODUCT.md rewritten). Design canvas: https://claude.ai/artifact/QeGKGXsth2t4gtvvDgFdoT (v8, states S1 to S9). Baseline: 41 test files, 263 tests, tsc clean.

L3. Inventory result: only three items need the backend (test counts, a "note and resume" action, a prediction of files a retry will touch). The third is dropped. "Open the other run" on a busy checkout also needs the holder run id exposed.

L4. 2026-11-24: for mobile I proposed Run controls before the tabs. The user said the design link is the source of truth. The mobile artboards order the blocks Stages, Needs you, Proof of work, Safe and kept, Run detail (tabs), Run controls, so the page follows that and the proposal is dropped. The design also shows a "Running · Nothing needed from you" card with "Pause after this stage" while a run goes on its own; the page draws no block in that case yet.
