# focused-prove

Run the tests that belong to the story, and treat the rest of the suite as a report
instead of a verdict.

## Specs

S1. "es la herramienta la que debe entender que las pruebas deben ser focalizadas en su historia no en todo el proyecto"

S2. Span already maps every acceptance criterion to the test file that declares it. That
mapping is the scope: no model, no heuristic, no new configuration to guess.

S3. A focused band blocks. The wide suite, when a focused band exists, reports: a failure
outside this story's files cannot be repaired by this story's agent, so it must not spend
the repair budget or paint the criteria.

S4. Without a focused band configured, behavior is exactly what it is today: the fast band
blocks.

## Tasks

| ID | Spec | Route | What |
| --- | --- | --- | --- |
| T1 | S2 | inline | qa: the declaring files for a set of criteria, as a scope |
| T2 | S3, S4 | inline | policy and prove: a focused band that blocks and a wide band that reports |
| T3 | S1-S4 | inline | engine: resolve `{files}` from the scope, record both bands |
| T4 | S1-S4 | inline | docs and changelog |

## Log

L1. User request, verbatim: "Claro pero entonces es la herramienta la que debe entender que las
pruebas deben ser focalizadas en su historia no en todo el proyecto"

L2. Evidence that forced it: run `2026-11-19-0253-87f0` (salute-hermes) fails `npm test`
with 15 tests in `src/modules/catalog`, `src/modules/persons`, and `src/app-routes.test.tsx`.
None of them is in `src/modules/collections`, which is what that story implements. The agent
cannot repair them: they are other stories. The cap of three attempts was spent on a failure
that was never this story's.

L3. The scope comes from the declaration scan Span already performs for QA
(`qa.Locate`). Asking the person to write a path filter is asking them to repeat a map the
tool holds.

L4. The wide band is not deleted. It still runs and its output is still kept, because a
regression outside the story is real information. It just stops being the verdict, and the
reason names it.

## Log continued

L5. Evidence of verification. `qa` tests cover the scope list: unique, sorted, relative, and
empty when nothing is declared. `prove` tests cover a focused failure iterating while the
wide band does not, the wide band alone reporting without iterating, and no focused band
keeping today's behavior. `engine` tests cover `{files}` substitution and the recorded
bands. `web` and the full Go suite stay green.
