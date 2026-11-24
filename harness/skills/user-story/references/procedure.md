# Procedure

Load on every run. Story input comes from four labeled sources: `prompt`, `answer`, `code` (a repository path read, read-only), `context` (`span/context/product.md`). There is no sample story.

## Extract

Write two lists before any prose.

- Stated: fact, its source label, and the words or path that carry it.
- Needed: fact a BR, AC, or Entonces would have to assert, which no source states.

A composition of two stated rules is stated. When those rules produce one user-visible sentence, put that sentence in a TC and in Impacto. Do not soften it.

## Gate

Ask when a needed fact would be written into a BR, an AC, or an Entonces, and the outcome disappears without it.

Do not ask when the fact can live in `No incluye` and `Supuestos` while every stated rule stays intact. Do not ask a fact a source already answered.

Assume only after the user says to assume or accepts a proposed assumption. Then the assumption is a Supuesto, never a BR.

Split when the prompt has two outcomes a person could receive on their own. Use the first pattern that fits: separate workflow, separate operation, business-rule variation that is its own outcome, or a data set that could ship later. Do not split quality of the same outcome (channel, locale, accessibility). Do not split by screen layer. Two origins of the same result are one story.

## Ask block

Stop after this block. Questions in the user's language. One batch.

```markdown
## Preguntas para cerrar la historia

No escribo la historia todavía. Cada respuesta evita inventar una regla o un resultado.

1. {pregunta}
   Bloquea: {BR, AC o TC que quedaría inventado}
   Si hay que asumir: {supuesto, en una oración}
```

## Split block

Stop after this block.

```markdown
## Historias separadas

El prompt tiene más de un resultado que se puede entregar solo. Indica cuál hay que escribir.

1. {Como / Quiero / Para, en una línea}
   Queda afuera: {el otro resultado}
```

## Write each section

Title: the outcome, not the component.

Historia:

- Como: a person in a situation. Not a system, role title alone, or "usuario". Use a persona from `context` when one fits; propose a new one, never write it into the file.
- Quiero: the capability. Not the implementation plan.
- Para: an outcome that person recognizes outside the ticket.

Contexto de Uso: trigger, what stays, what changes, and the source of each visible datum. Name the owner of a shared template when the prompt names one.

Alcance: `Incluye` is what this story accepts. `No incluye` is a neighbor the prompt touched and this story does not change, including the failure behavior of a boundary the prompt did not define.

Reglas de negocio:

- One constraint. Present tense. No "when the user clicks".
- No color, pixel, font, component, or copy layout.
- "The same result from A or B" is one rule. "If A then X; if B then Y" is two rules.
- An exception is its own BR.
- Do not restate an AC. If a sentence only exists to lock a color, it is an AC.

Criterios de Aceptación:

- Pass or fail by observation.
- Quote copy the prompt already wrote. Do not invent copy.
- Line `Confirma:` lists BR ids, or `diseño acordado` when the prompt fixed a visual and no rule exists.
- Do not write an AC for a design the prompt did not fix.
- A rule wider than this story stays a BR. The AC confirms only the part `Incluye` owns, and says so.

Escenarios de prueba:

- Key examples, not one paraphrase per AC.
- Every AC id appears in some `Cubre`. Every BR id appears in some `Confirma` that a TC covers.
- Entonces quotes the value. "The correct date" is invalid.
- Dates, counts, and lengths: one example on the stated value, one on the boundary where the displayed value changes (month, year, zero, maximum). Do not invent what the product does past that boundary.
- Use names and addresses from the prompt. A fixture the prompt did not give is prefixed `Dato de ejemplo:` inside Contexto.
- Separate rows for a different actor, origin, variant, channel, or boundary. Merge rows that add no new fact.

Impacto. Walk this list. One row per hit. Skip a lens the prompt does not touch.

1. Other entry points of the same artifact.
2. A downstream step the prompt says stays.
3. A datum another surface also shows.
4. A named variant, ticket, or template owner.
5. Channels or clients the prompt names.
6. A secret, a sum of money, or a permission: who else can see it or do it.
7. A sentence produced by composing two rules, when that sentence is awkward or redundant.

Evidencia starts with `Observado:`, `Observado en código: <path>`, `Observado en contexto:` or `Supuesto:`. No invented system. A `code` surface is only one the agent read at that path. If the prompt names no shared surface, one row: `Ninguna superficie compartida nombrada` | `No hay efecto que afirmar` | `Observado: el prompt no nombra otra superficie`.

Supuestos: every fact the story uses that no source stated (including accepted assumptions), including sample fixtures and excluded boundaries. `- Ninguno.` when the story uses only stated facts.

## Identifiers

Keep the user's shapes: `BR100`, `AC 100`, `TC-01`.

- Group by outcome, in the order the story meets it. First group `100`, then `200`.
- Number BR and AC of that outcome inside the hundred, packed, no gaps.
- Next outcome takes the next hundred even if this one used three ids.
- TC ids are sequential in tester order: happy path, rules, variants, boundaries, quality.
- Do not renumber for taste.

## Checklist

Run before returning. Fix the story. Do not return the checklist.

- Headings match `assets/template.md`, in order, none added.
- Como is a person in a situation. Para is an outcome that person recognizes.
- Each BR is one declarative constraint, with no visual and no step sequence.
- Each AC is pass/fail, has `Confirma:`, and quotes only copy from the prompt.
- Each BR id is in a `Confirma`. Each AC id is in a `Cubre`.
- No Entonces introduces a datum, date, or behavior absent from a BR, an AC, or a labeled `Dato de ejemplo`.
- A numeric or date rule has a boundary example, and the undefined side is in `No incluye`.
- Impacto was walked. Secrets, money, and permissions are rows when the story shows them.
- Supuestos lists every unstated fact actually used.
- Prose is in the user's language. Headings were not translated.
