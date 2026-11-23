# agent-limit

Notice when the selected agent CLI cannot work because it ran out of quota or
credentials, instead of failing as an unexplained stage error.

## Specs

S1. "Agregues una forma de capturar que los agentes que se usen ya quedaron sin token"

S2. The condition is a hold, not a stop: the run continues once that CLI can work
again, and the reason names what to do.

S3. The classification is deterministic: a curated phrase list per kind, no model, no
extra process. A diff or a test output that happens to contain a number must not be
read as a limit.

S4. "Documentar todos los cambios recientes del proyecto"

## Tasks

| ID | Spec | Route | What |
| --- | --- | --- | --- |
| T1 | S1, S3 | inline | agent: classify quota and credentials from the captured output, typed error |
| T2 | S1, S2 | inline | engine: record the block on the cursor and hold with a reason that says what to do |
| T3 | S1, S2 | inline | web: show the block and which CLI needs attention |
| T4 | S4 | inline | CHANGELOG.md and the documentation the behavior change needs |

## Log

L1. User request, verbatim: "Necesito tambien que 1. Agregues una forma de capturar que los
agentes que se usen ya quedaron sin token y 2. Documentar todos los cambios recientes del
proyecto."

L2. Two kinds, because the fix differs. `quota` means the limit or credit ran out and
the answer is to wait or switch CLI. `credentials` means the key is missing or rejected
and the answer is to log in. The same classification covers both, and the reason text
names the right action.

L3. Observed in this session: two subagent runs failed with "No API key found for
openai-codex". That is the `credentials` kind arriving from a real command, which is why
the signature list covers it and not only quota phrasing.

L4. Signature list is curated phrases, not bare status codes: "429" alone appears inside
diffs and test output. Precision over recall, and the unmatched case keeps the current
behavior of a hold with the raw error.

L5. Evidence of verification. `agent` tests: six quota phrases, five credential phrases,
four ordinary failures that must stay unclassified, a real script named `claude` whose
output is classified end to end, an ordinary failing script that stays an error, and a
joined error read through `errors.As`. Engine tests: a quota block holds with "quota
exhausted (claude)" and records `AgentBlock`; a credential block says to log in; an
ordinary agent error records no block. `web`: `tsc --noEmit` clean and 36 tests passing.
Differential check against the copy taken before the work: the binary's top-level output,
the `repos` output, and the exit code are identical.
