---
name: change-impact
description: "Trigger: impacto, análisis de impacto, blast radius, ripple effect, regresión, qué se rompe, change impact. Maps creates, modifies, extends, and which other behaviors can break."
license: "Proprietary"
metadata:
  author: "Carlos San Juan"
  version: "1.0.0"
  category: "architecture"
---

## Activation Contract

Load before creating, modifying, or extending a behavior, in a UI or a service, and when asked what a change will affect. A read-only impact question ends at the analysis.

Prevent this: behavior X ships, a later change to B shares a surface with X, and X breaks after B's own checks pass.

## Hard Rules

- A feature is a behavior plus the invariant that keeps it true.
- Label every touched element Create, Modify, or Extend. A change of meaning, timing, shape, or stored data is Modify.
- Trace every Modify and Extend. "No impact" requires the search you ran. Label each hop Observed or Assumption. Do not invent one.
- A shared contract, record, flow, permission, flag, or timing is one cell. An unfinished trace on a surface this change writes or modifies is High.
- One behavior per change. A drive-by refactor, rename, or schema rewrite is a second change.
- The verdict follows the ripple score. High means do not edit.
- Fill every section of `assets/analysis-template.md`. Empty means `none found` plus the search. Write the prose in the user's language. Keep the headings verbatim.

## Decision Gates

| Situation | Action |
| --- | --- |
| Impact question only | Return the analysis. Do not edit. |
| None or Low | Proceed. Implement, then run the named checks. |
| Medium | Pin first. Add the neighbor pin, then implement. |
| High, or a written surface you could not trace | Stop. Do not edit. |
| No code, and the change is Modify or writes untraced state | High. Stop. Hops are Assumption. |
| Format, comment, or rename with no behavior change | None. Still return the template. |

Score with `references/procedure.md`.

## Execution Steps

1. State one behavior and its kind: Create, Modify, or Extend.
2. Walk `references/lenses.md`. Name the shared surface, or why there is none.
3. Search those identifiers, the neighbor's name, and any pin or `# Impact:` note. Build the starting set and the estimated set from `references/procedure.md`.
4. Write each neighbor invariant as a sentence a check could fail.
5. Score, set the verdict from the gate, and for Medium or High take one move from `references/mitigations.md`.
6. Fill `assets/analysis-template.md`. After an allowed edit, append pass, fail, or not run under Verification.

## Output Contract

Return the filled template before any edit. Do not replace it with a summary. Save the note only when the repo already keeps impact notes. The pin is what the next session can find.

## References

- `assets/analysis-template.md` — required output. Fill every section.
- `references/lenses.md` — load on every run.
- `references/procedure.md` — load to trace and score.
- `references/mitigations.md` — load for Medium or High.
- `references/example.md` — load when the change touches a behavior that already shipped.
- `references/concept.md` — load when asked what this practice is.
- `references/sources.md` — load when asked for sources.
