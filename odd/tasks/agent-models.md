# agent-models
Objective: choose one model per agent CLI (cursor, claude, codex, pi), stored in span.yaml and shown on the desk next to the agent picker. Gentle AI and its configuration stay required.
Branch: feat/agent-models. Delivery: ask-on-risk. Runner: go test, npx vitest.

## Specs
- S1 "por cada agente, tanto cursor, claude, pi etc, necesito que se muestren los modelos para seleccionar dicho modelo como el que orquesta el trabajo". One model per CLI, listed from the CLI where it can list (`cursor-agent --list-models`, `pi --list-models`), fixed aliases for claude, free text for codex.
- S2 "Un modelo por cada CLI": `models:` in span.yaml maps a CLI id to a model; whenever that CLI runs (frame, build, inspect, conflict proposal) it gets `--model <value>`. Empty means no flag: the CLI keeps what Gentle AI configured.
- S3 "hay que dejar claro que gentle-ai y toda su configuracion es necesaria": the desk says so next to the model choice, and a CLI without Gentle AI cannot be chosen.
- S4 (assumption) A model value is validated (`^[A-Za-z0-9][A-Za-z0-9._:/@+-]*$`, max 100) at load, at write and before exec, and travels as its own argv element.
- S5 Out of scope: the install wizard ("Tal vez mas adelante exlorar la opcion de hacer un onboarding o un wizard").

## Tasks
- T1 S2,S4 registry + exec: Spec.ModelFlag, ValidModel, model arg in the command (inline)
- T2 S2,S4 policy: models map, validation, SetModel (inline)
- T3 S1 model lists: cursor and pi parsers, claude aliases, bounded listing (inline)
- T4 S1,S3 serve: models in GET /agent, GET /agent/models, POST /agent/model (inline)
- T5 S1,S3 web: model picker and Gentle AI notice (inline)
- T6 docs, changelog (inline)

## Log
L1 (2026-11-23) user: "Tambien necesito que por cada agente, tanto cursor, claude, pi etc, necesito que se muestren los modelos para seleccionar dicho modelo como el que orquesta el trabajo. Ya dado en cada agente se maneja internamente sus modelos con para gentle-ai, y OJO, hay que dejar claro que gentle-ai y toda su configuracion es necesaria. Tal vez mas adelante exlorar la opcion de hacer un onboarding o un wizard que ayude a instalar todo."
L2 (2026-11-23) user chose "Un modelo por cada CLI (Recomendado)".
L3 Mirror to Engram pending: mem_save is blocked by a hook in this session.
