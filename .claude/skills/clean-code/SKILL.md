---
name: clean-code
description: "Trigger: clean this up, make it readable, review this, refactor, limpiar código, hacer legible; naming, functions, comments, error handling, duplication, smells. Applies Clean Code defaults with cost-based tradeoffs to TS/JS and beyond."
license: "Proprietary"
metadata:
  author: "Carlos San Juan"
  version: "2.0.0"
  category: "code-quality"
---

## Activation Contract

Use when writing new code, reviewing a diff, or refactoring for readability and maintainability, even when the user only says "clean this up", "make it readable", "review this", or "refactor" without naming Clean Code. Examples are TypeScript/JavaScript; rules are language-agnostic. For SOLID, load `../solid-principles/SKILL.md`.

## Hard Rules

- Optimize for the next reader: meaning over cleverness; never justify a rule by authority, only by the cost it prevents.
- Names are intention-revealing, searchable, and consistent (one word per concept); booleans read as predicates; functions are verbs, classes are nouns.
- One function does one thing at one level of abstraction; describe it without "and".
- No flag arguments: a boolean that switches behavior means two functions.
- Guard clauses first; the happy path stays flat and un-indented.
- Never swallow errors; include what was attempted and identifiers in every error message.
- Delete dead and commented-out code; delete comments that restate the code; keep only comments that explain why.
- Abstract on the third repetition, not the first (rule of three); DRY is about knowledge, not similar text.
- Team convention beats personal preference; match the existing style and tooling.
- When a rule and clarity conflict, clarity wins, and state the reason.

## Decision Gates

| Smell | Refactor | Load |
|---|---|---|
| Long function | Extract Function until each does one thing | `references/functions.md` |
| Large class | Extract Class along responsibility lines | `references/smells.md` |
| Long parameter list | Introduce a parameter object | `references/functions.md` |
| Duplicated code | Extract shared function/module after the third repeat | `references/principles.md` |
| Feature envy | Move Method to the data owner | `references/smells.md` |
| Primitive obsession | Introduce a value type (`Email`, `Money`) | `references/smells.md` |
| Shotgun surgery | Gather the scattered responsibility into one place | `references/smells.md` |
| Divergent change | Split so each part has one reason to change | `references/smells.md` |
| Comments as deodorant | Rename or refactor, then delete the comment | `references/comments-and-formatting.md` |
| Dead code | Delete it | `references/comments-and-formatting.md` |
| Magic numbers/strings | Name them as constants | `references/smells.md` |

| Situation | Action |
|---|---|
| Splitting scatters one coherent thought | Leave the function whole |
| Codebase follows a different consistent style | Match it; raise changes as a team decision |
| Throwaway script, one-off migration, prototype | Skip the full treatment; match rigor to lifespan |
| Hot path needs a dense implementation | Keep it and document why |

## Execution Steps

1. Read the target code and classify the task: new code, diff review, or refactor.
2. Apply the Hard Rules directly; do not load references for rules already satisfied.
3. For each smell present, load only the reference named in the Decision Gates table.
4. Load `references/naming.md` for unclear identifiers, `references/error-handling.md` for try/catch or null returns, `references/tests-and-boundaries.md` for tests or third-party wrappers, `references/pragmatism.md` before breaking a rule.
5. Draft each change with the cost it prevents; drop any finding whose cost you cannot state.

## Output Contract

Return a findings list; for each item give:
- Rule or smell name.
- Location (file and line or symbol).
- Cost prevented (concrete reader or maintenance cost, never "the book says").
- Proposed change (before/after or a one-line refactor).
State explicitly when a rule was deliberately not applied and why.

## References

- `references/naming.md` — names, meaning over cleverness.
- `references/functions.md` — size, arguments, flags, side effects, nesting.
- `references/comments-and-formatting.md` — comment value, dead code, layout.
- `references/error-handling.md` — exceptions, null, fail-fast, error context.
- `references/principles.md` — DRY/KISS/YAGNI; pointer to the SOLID skill.
- `references/smells.md` — smell-to-refactor catalogue.
- `references/tests-and-boundaries.md` — AAA, FIRST, boundary wrappers.
- `references/pragmatism.md` — when to break a rule.
