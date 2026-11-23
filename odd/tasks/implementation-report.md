# Implementation report

Objective: replace the transcript panel of the Implementation tab with a report a person can scan before trusting a change. Branch feat/implementation-report. Delivery: ask-on-risk. Tests: go test, vitest.

## Specs

- S1: "la transcripcion resuma correctamente para mostrar elementos como, el porque se hizo de esa forma, mostrar posibles riesgos asociados a los cambios que hizo, que decisiones asumio".
- S2: "expon todos los posibles riesgos, mejora la auditoria".
- S3: "la visualizacion sea mas chevere y facil de leer a nivel de UI".
- S4: "investiga ... que data es buena para un informe de ese tipo, estandares de la industria tambien en internet, y personas referentes en la industria, y herramientas similares".
- S5 (assumption): nothing is invented. Every line says where it came from: detected by Span from the diff and the run, reported by the agent in its feature document, or verified by a check or the inspector.
- S6 (assumption): the raw transcript stays reachable, collapsed.

## Tasks

- [x] T1 (S1, S5): `GET /spans/{id}/feature` returns the feature document, bounded. Inline.
- [x] T2 (S1, S5): build prompt asks for `Decision:` and `Risk:` lines in the Log. Inline.
- [x] T3 (S2, S5): risk and audit rules in `web/src/lib/report.ts`, from the diff, statements and cursor. Inline.
- [x] T4 (S1, S3): report components. Inline.
- [x] T5 (S4): research notes in USAGE (section "The implementation report").

## Log

L1 (2026-11-23): "1. Haz lo que tu creas que es lo recomendado 2. Una nueva modificacion con /impeccable necesito que en la seccion implementacion donde esta la transcripcion quiero que lo mejores a nive de funcion y visualizacion. La idea es poder que la transcripcion resuma correctamente para mostrar elementos como, el porque se hizo de esa forma, mostrar posibles riesgos asciados a los cambios que hizo, que decisios asumio, aparte quiero que la visualizacion sea mas chevere y facil de leer a nivel de UI tal vez con mejor html, expon todos los posibles riesgos, mejora la auditoria, basicamente investiga en intener que data es buena para un informe de ese tipo, estandares de la industria tambien en internet, y personas referentes en la industria, y herramientas similares y hazlo facil de interpretar y leer"

L2: route inline, no writer reason (units depend on each other). Research runs in a read-only explorer.

L3: research returned (SmartBear 200-400 LOC, Google ~100 reasonable, ADR/MADR, SLSA, CodeRabbit order, Fowler Ship/Show/Ask). Kent Beck, Gergely Orosz and Devin/Cursor/Graphite presentations were not verified and are not cited. Evidence per task: go test -race ./... ok, GOOS=windows build ok, web tsc/eslint/vitest 199 ok.
