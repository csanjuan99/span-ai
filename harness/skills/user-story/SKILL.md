---
name: user-story
description: "Trigger: historia de usuario, user story, criterios de aceptación, business rules, casos de prueba, afinar una historia. Builds one detailed story in a fixed BR, AC, and TC format, or interviews first."
license: "Proprietary"
metadata:
  author: "Carlos San Juan"
  version: "2.0.0"
  category: "product"
---

## Activation Contract

Load when the user asks for a user story, historia de usuario, acceptance criteria, business rules, or test cases, or when Span drives a shaping interview. The input may be one sentence or a full spec. This skill writes the story. It does not implement it.

Two modes: **write** (default, one-shot) and **interview** (Span drives it turn by turn; see `references/interview.md`).

## Hard Rules

- On a write, return `assets/template.md` filled, headings verbatim, in that order. Do not add or rename a section.
- A business rule is one declarative constraint. No colors, sizes, fonts, components, or step order. An exception is another rule.
- An acceptance criterion is one observable pass/fail. Copy and visuals already fixed go here, each with `Confirma:`.
- A test case is a key example: concrete data in Dado / Cuando / Entonces, traced with `Cubre`. It adds no fact the rules and criteria do not state.
- Never invent a rule, offset, copy, color, actor, or expected result. A missing fact is a question, a `No incluye`, or a `Supuesto`.
- Every fact comes from one of four sources and carries its label: `prompt`, `answer` (the person answered a question), `code` (a repository path you read, read-only), `context` (`span/context/product.md`). No label, no fact. No sample story exists.
- An accepted assumption is a `Supuesto`, never a BR.
- One story, one user outcome. Quality of that same outcome stays inside it.
- Write the prose in the user's language. Keep the headings verbatim.
- Ask or split means stop. Do not attach a draft.
- Never write a file or change the repository. Product context additions are proposals only; the skill never writes `span/context/product.md`.

## Decision Gates

| Situation | Action |
| --- | --- |
| A rule, criterion, or expected result would need a fact no source gave, and dropping it removes the outcome | Ask, using `references/procedure.md`. Stop. |
| That fact can be excluded without changing a stated rule | Write. Put it in `No incluye` and `Supuestos`. |
| Two outcomes the user could receive separately | Propose the split. Stop. |
| The user told you to assume, or accepted a proposed assumption | Write. Assumptions are `Supuestos`, never rules. |
| The sources already state the facts | Write. Do not ask them back. |
| Span drives the turn | Interview mode: `references/interview.md`. |

## Execution Steps

1. Read `assets/template.md` and `references/procedure.md`. In interview mode also read `references/interview.md`.
2. Read `span/context/product.md` when present. Read repository paths only to confirm what exists, never to decide what the story should be.
3. List facts stated, each labeled. List facts a rule, criterion, or expected result would still need.
4. Apply the gate.
5. On a write, fill every section and run the procedure checklist.
6. Return only the output for that gate.

## Output Contract

- Write: the filled template only. No preamble.
- Ask: the question block only. Split: the slice list only.
- Interview mode: JSON only, matching the schema in Span's prompt. Span's prompt owns that schema; this skill defines none.
- A section with nothing to say still has `Ninguno` or one explicit row.

## References

- `assets/template.md` — required output. Placeholders only.
- `references/procedure.md` — load on every run.
- `references/interview.md` — load in interview mode.
- `references/sources.md` — load when asked why the format is this way.
