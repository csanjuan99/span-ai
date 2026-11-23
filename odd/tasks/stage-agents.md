# stage-agents

Choose which CLI writes each agent stage of a live run, and pause the flow at a stage
boundary to make that choice.

## Specs

S1. "Tambien seria bueno poder seleccionarlo por etapas."

S2. "Asi por ejemplo en una sesion ya iniciada puedo seleccionar en una etapa que modelo usar si esta parada o algo asi."

S3. "Tambien me gustaria poder parar esas etapas de un flujo para por ejemplo decidir cambiar el modelo."

S4. Only the stages that call an agent can carry a choice: frame, build, and inspect.
Prove, qa, open, merge, and promote run commands, not a model.

S5. The choice belongs to the run, not the project. `span.yaml` keeps the default; the
run keeps the exception.

## Tasks

| ID | Spec | Route | What |
| --- | --- | --- | --- |
| T1 | S1, S2, S4 | inline | engine: per-run, per-stage agent file, `pick(stage)`, override wins over policy |
| T2 | S3 | inline | engine: a pause request consumed at the stage boundary, as a hold |
| T3 | S1-S3 | inline | serve: `POST /spans/{id}/agent` and `POST /spans/{id}/pause`, plus the choice in the snapshot |
| T4 | S1-S3 | inline | web: per-stage picker on a live run and a pause button |
| T5 | S1-S5 | inline | docs and changelog |

## Log

L1. User request, verbatim: "Tambien seria bueno poder seleccionarlo por etapas. Asi por
ejemplo en una sesion ya iniciada puedo seleccionar en una etapa que modelo usar si esta
parada o algo asi. Tambien me gustaria poder parar esas etapas de un flujo para por
ejemplo decidir cambiar el modelo."

L2. The cursor belongs to the runner, and the page is another process. Two writers on one
`cursor.json` means the runner's next `save()` erases what the page wrote. So both
requests are files beside the cursor, and the runner reads them where it needs them:
`agents.json` for the per-stage choice and `pause` for the stop request. This is the same
shape the engine already uses for instructions and answers.

L3. The pause is cooperative and consumed at a stage boundary: it holds before the next
stage starts. It does not kill a stage that is mid-write, because that leaves a half
written statement. A stage already running finishes, then the hold happens, which is
exactly the moment a person needs to change the model.

L4. Resolution order per stage: the run's choice for that stage, then `agent:` from the
policy, then the first ready CLI. An empty choice, or `any`, clears the exception.

L5. Not implemented, and the reason: killing a running agent to change the model now.
The engine tolerates a killed process (resume reads the cursor), but a button that kills
mid-write is a separate decision about partial output, not a configuration feature.

## Log continued

L6. Evidence of verification. `engine` tests: the override wins for the stage it names and
not for the others, an empty choice falls back to the policy, the pause holds before the
named stage and is consumed so the next resume runs, a request for a stage that uses no
agent is refused, an unknown agent is refused, and a broken `agents.json` is ignored.
`serve` tests cover both endpoints and the snapshot. `web` typechecks and its suite stays
green.
