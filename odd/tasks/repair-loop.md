# repair-loop

Make a failed check return to build with evidence, and stop an unchecked process
exit from painting every acceptance criterion red.

## Specs

S1. "Guardar un tramo corto de la salida en el statement y mostrarlo en el próximo prompt."

S2. "QA no debería repetir la suite rápida. La cobertura de declaraciones y "el proceso salió 0" son chequeos distintos."

S3. "Un prove o un QA en rojo debería volver a build con esa evidencia, con un tope que diga qué falló."

S4. "Una historia en dos repos son dos spans."

S5. "El camino local sigue barato: sin otro modelo para explicar el fallo, sin índice, y sin una segunda suite completa cuando ese commit ya pasó en un árbol limpio."

S6. "El sistema debe ser resiliente, poder decir que si fallaron las pruebas se necesita iterar build para que se construyan como un fix y asi sucesivamente hasta que se valida."

S7. "Pueden ser N iteracione hasta conseguir que el codigo sea correcto" (user decision on the repair loop).

S8. "Re-corre sólo si el árbol cambió" (user decision on QA and the fast suite).

S9. "Los 22 criterios, de AC-100 a AC-601, figuran como fail. En el código, un id cubierto pasa a fail cuando el proceso sale distinto de cero. Un id que no aparece en una declaración queda unverified. Los 22 están declarados. El proceso entero falló. No hay evidencia de que cada criterio se haya ejecutado y haya fallado."

## Tasks

| ID | Spec | Route | What |
| --- | --- | --- | --- |
| T1 | S1 | inline | statement: per-command output tail, log path, fingerprint; repair ledger on the cursor |
| T2 | S1, S5 | inline | adapt.Exec and prove.Checker return captured output; engine writes the full log |
| T3 | S9, S2 | inline | qa: read a machine-readable report; a process exit no longer fans out to per-id fail |
| T4 | S3, S6, S7 | inline | engine: one repair edge from prove and qa to build, fingerprint guard, durable attempt ledger |
| T5 | S2, S5, S8 | inline | engine: qa consumes prove evidence when commit, tree and command match; inconclusive on a same-commit flip |
| T6 | S1, S3 | inline | prompt: failure packet in the build prompt |
| T7 | S7, S8 | inline | policy: repair block and qa.report |
| T8 | S6 | inline | web: output tail, attempt counter, and one action on a held stage |
| T9 | S1-S9 | inline | USAGE.md and README describe the repair loop |

## Log

L1. User request, verbatim:

> Necesito que me ayudes a solucionar este problema en Span: Qué pasó
>
>  Las dos corridas son la historia SAL-3, rama feat/SAL-3-ver-el-estado-de-cuenta-al-entrar-a-carter desde dev. Los árboles están limpios, salvo .span/. No se pusheó nada.
>
>  Infra (2026-11-19-0253-1a7d) sí llegó a QA y se frenó ahí, en la iteración 3.
>
>  • A las 13:05 UTC, npm test pasó en el commit eb24a18. Prove avanzó con "fast and acceptance passed". La banda de acceptance está vacía: esa frase significa que el comando rápido pasó.
>  • A las 13:27 UTC, QA volvió a correr npm test sobre el mismo commit y salió 1. El motivo sellado es acceptance gate fail.
>  • Los 22 criterios, de AC-100 a AC-601, figuran como fail. En el código, un id cubierto pasa a fail cuando el proceso sale distinto de cero. Un id que no aparece en una declaración queda
>    unverified. Los 22 están declarados. El proceso entero falló. No hay evidencia de que cada criterio se haya ejecutado y haya fallado.
>  • El log de la feature cuenta que una corrida anterior dejó registration.http.spec.ts en PREPARING por carga paralela, y que después la suite completa pasó: 174 suites, 1598 tests. Esa nota
>    la escribió el agente. Span no guardó la salida de las 13:27. No da para afirmar que el fallo de QA sea ese.
>
>  Hermes (2026-11-19-0253-87f0) no entró a QA. Está en prove, iteración 3, con iteration cap: npm test exited 1, commit ac338b2, a las 13:24 UTC. npm test es vitest run de todo el repo y salió 1
>  en las tres iteraciones. El agente había corrido un subconjunto de 142 tests y el typecheck, y ambos pasaron. El nombre del test que rompe la suite completa no quedó guardado.
>
>  Por qué no termina
>
>  Exec.Run se queda solo con el código de salida. Prove, al tercer fallo, hace hold y no vuelve a llamar al agente. QA, si el gate no pasa, hace hold y tampoco vuelve a build. Reanudar repite la
>  misma etapa: en Hermes vuelve a prove y, si npm test sigue en rojo, se frena otra vez; en infra solo vuelve a correr QA. El prompt de build pide una declaración por criterio, y QA busca ese id
>  en la línea de la declaración.
>
>  QA, además, repite el comando rápido. En infra eso son 1598 tests, veinte minutos después de que prove ya los había pasado. Un fallo ajeno pinta los 22 criterios en rojo.
>
>  Qué conviene discutir
>
>  1. Guardar un tramo corto de la salida en el statement y mostrarlo en el próximo prompt.
>  2. QA no debería repetir la suite rápida. La cobertura de declaraciones y "el proceso salió 0" son chequeos distintos.
>  3. Un prove o un QA en rojo debería volver a build con esa evidencia, con un tope que diga qué falló.
>  4. Una historia en dos repos son dos spans. Infra puede estar en QA mientras Hermes no sale de prove.
>  5. El camino local sigue barato: sin otro modelo para explicar el fallo, sin índice, y sin una segunda suite completa cuando ese commit ya pasó en un árbol limpio.
>
>  El primer cambio que destrabaría estas dos corridas es el 1 junto con el 3: sin la salida, el agente no tiene qué corregir, y sin volver a build la herramienta se queda en el hold. Este es el analisis de otro agente de algo que me bloqueo en la interfaz y es que no pude hacer mas nada en el pipeline porque quedo en QA y alguna otra no se ejecuto. El sistema debe ser resiliente, poder decir que si fallaron las pruebas se necesita iterar build para que se construyan como un fix y asi sucesivamente hasta que se valida. Necesito que contrastes eso con estandares de la inudstria en internet, flujos profesionales, herramientas de desarrollo de este tipo, hasta que logremos de hacer de esta una intuitiva sencilla de usar y funcional

L2. Scope decision: A + B + C. Evidence packet, the repair edge, and the split gate
in one slice.

L3. Policy decision: "Pueden ser N iteracione hasta conseguir que el codigo sea
correcto". Implemented as a no-progress budget, not an attempt count: the loop keeps
going while each failure is a new fingerprint and stops when the same fingerprint
returns on a different commit.

L3.1. Correction: "Entendido entonces no agregues ese antipatron de infinito". There is
no unlimited mode. `repair.maxAttempts` defaults to 3 and is validated to 1..8. The
fingerprint guard stops before the cap is spent in the common repeating-failure case;
the cap remains the hard ceiling.

L4. QA decision: "Re-corre sólo si el árbol cambió". Evidence from a clean tree at the
same commit is reused; a dirty tree or a different commit re-runs.

L5. Evidence: `adapt.Exec.Run` (internal/adapt/adapt.go:22-31) calls `cmd.Run()` with no
Stdout or Stderr, so the output is discarded. `qa.Judge` (internal/qa/qa.go:118-121)
maps any non-zero exit to `fail` for every covered id. `engine.go:468-471` returns a
prove failure to `frame`, not `build`. `engine.go:684-687` turns a QA gate failure into
`statement.Hold`, which has no outgoing repair edge.

L6. Constraint: the repository has no commit yet (every path is staged and `git log`
fails), so work-unit commits are not created without an explicit decision, and the
differential check runs against a copy of this tree in a temporary directory.

L7. Evidence of verification. `go vet ./...` clean. `go test ./...` green across all 20
internal packages. `web`: `tsc --noEmit` clean and 33 tests passing. Differential check
against a copy of this tree taken before the change: the binary's top-level output, the
`repos` output, and the exit code are identical.

L8. New tests pin the rules: `qa.Judge` leaves a covered id unverified on a non-zero exit
(the incident, 22 criteria, zero fail), a report attributes each id, a declared id the
report never ran is unverified, a zero exit without a report passes declared ids. Prove
keeps the output and one fingerprint per failure. `adapt.Exec` keeps both streams, the
exit code, and the end of a long output. Engine: the same failure twice holds as
`no progress` after two runs, new failures reach the cap, a failing QA gate returns to
build with the output and a log path, QA does not repeat the fast command on a clean
tree, and a same-commit flip holds as `inconclusive`. The QACases test already pinned
that a missing declaration reads unverified.

L9. Note for in-flight runs: a statement written before this change still parses, because
every new field is optional. Resuming a held run now enters the repair edge. The infra
run can resume into QA, reuse prove's green fast command on eb24a18 in a clean tree, and
pass on declarations alone.
