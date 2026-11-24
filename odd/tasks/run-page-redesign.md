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
| T4 | S5 | inline | Proof summary from `buildReport` | |
| T5 | S5 | deferred | Backend: test counts from the prove gate (own feature) | |
| T6 | S7 | inline | Mobile layout | |

## Log

L1. User request, verbatim: "Es que hay que entender que la UX y UI es nueva y no necesariamente tenemos que reemplazarla directamente con lo del codigo. Hay que evaluar lo bueno de esa propuesta y si no esta soportado SOPORTARLO. Ahi es donde esta el valor. Poder ingresar cosas neuva que faciliten el trabajo y sea intuitivo. Con eso en mente en que procedemos?"

L2. 2026-11-24: foundation committed as 3787bc6 (tokens, Geist, ThemeProvider mounted, DESIGN.md and PRODUCT.md rewritten). Design canvas: https://claude.ai/artifact/QeGKGXsth2t4gtvvDgFdoT (v8, states S1 to S9). Baseline: 41 test files, 263 tests, tsc clean.

L3. Inventory result: only three items need the backend (test counts, a "note and resume" action, a prediction of files a retry will touch). The third is dropped. "Open the other run" on a busy checkout also needs the holder run id exposed.
