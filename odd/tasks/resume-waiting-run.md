# Resume a waiting run

Branch: working tree. Delivery: ask-on-risk. Test runner: `go test` and `npm test` in `web/`.

## Specs

S1. "No deberia poder continuar ahi mismo? Respondiendo en span o modificado la historia de jira e indicar la reactivacion del flujo?"

S2. "Lo extraño es que en la historia si estan los criterios de aceptacion"

S3. "es entender la historia y lo que eso tenga llevarlo al tema decisorio del estandar de Span, NO se trata de agrear alias a los elementos en Sspan es decir no agregar como alias Criterios de aceptacion en span la idea es que de la historia se pueda entender no importa que sea en cualquier idioma y esto debe ser agnostico y generico"

S4. A question from frame is a hold on the same span. Answering on the span, or reading the Jira story again, runs frame again. A real stop, such as out of scope, stays stopped.

S5. Frame reads the story in the language it is written and returns the criteria it already states. Those lines are written under the one standard heading `Acceptance criteria:`. The engine does not keep a list of translated headings.

S6. A missing Jira account is not a question and does not stop the run. A waiting issue gets a comment and is not moved to Block.

## Tasks

- T1. S3 S5. Frame carries story criteria into the standard section. inline. done. `go test ./internal/qa/ ./internal/prompt/ ./internal/engine/`
- T2. S1 S4 S6. Waiting runs resume from an answer or a story reload. inline. done. `go test ./internal/engine/ ./internal/serve/`
- T3. S1 S2. The span page shows the questions, Answer and continue, and Read the story again. inline. done. `npm test` and `npm run build` in `web/`

## Log

L1. Ajusta eso entonces

L2. Ojo, es entender la historia y lo que eso tenga llevarlo al tema decisorio del estandar de Span, NO se trata de agrear alias a los elementos en Sspan es decir no agregar como alias Criterios de aceptacion en span la idea es que de la historia se pueda entender no importa que sea en cualquier idioma y esto debe ser agnostico y generico

Mirror pending: Engram project is ambiguous between span and web.
