# agent-registry

Four agent CLIs Span can drive, one registry, and a way to choose one from the page.

## Specs

S1. "Quiero que soportemos estos agentes: Cursor, Codex, Claude y PI"

S2. "a ellos debemos poder configurarlos facilmente desde la interfaz"

S3. "que el flujo sepa como usarlos finalmente"

## Tasks

| ID | Spec | Route | What |
| --- | --- | --- | --- |
| T1 | S1, S3 | inline | agent: one spec table driving detection, selection, arguments, and readiness |
| T2 | S2, S3 | inline | policy: validate `agent:` against the registry and rewrite the one line |
| T3 | S2 | inline | serve: `GET /agent` and `POST /agent` |
| T4 | S2 | inline | web: the Agent picker on the desk |
| T5 | S1-S3 | inline | docs and changelog |

## Log

L1. User request, verbatim: "Quiero que soportemos estos agentes: Cursor, Codex, Claude y
PI, a ellos debemos poder configurarlos facilmente desde la interfaz y que el flujo sepa
como usarlos finalmente."

L2. Evidence for Pi came from the installed Pi docs, not from guessing:
`docs/cli.md` documents `-p, --print` ("run the supplied prompts, write the final
assistant text to stdout, then exit") and `-a, --approve` ("trusts project-local files for
one command"). Span calls `pi --print --approve`.

L3. The Gentle AI marker for Pi is the installed package directory
`~/.pi/agent/npm/node_modules/gentle-pi`, which exists on this machine and is the same
package that carries its skills. The other three keep their documented rules files.

L4. One table, because the flow had four places that each encoded the same list: `Names`,
`commandArgs`, `Gentle`, and the policy. A new agent is now one row.

L5. Legacy values keep working: `agent: cursor-agent` is accepted, so an existing
`span.yaml` does not stop loading after the rename to ids.

L6. The picker refuses to invent a policy. A folder without `span.yaml` is told to run
`./span quickstart` once, because guessing a feature name to write a valid file is worse
than one clear sentence.

L7. Evidence of verification. `agent` tests cover the registry order, Pi's arguments, the
legacy binary name, both halves of readiness, a directory marker, `Select` for a chosen
and an absent agent, the `any` order, the report shape, and the value forms `Known`
accepts. `policy` tests cover the one-line rewrite that keeps comments, insertion when the
key is missing, refusal of an unknown id, refusal to write a file that would not load, and
the legacy name in `Validate`. `serve` tests cover the four-row list, a folder without a
policy, a saved choice, an unknown id, and the message that names quickstart. `web` has
39 tests and a clean typecheck.
