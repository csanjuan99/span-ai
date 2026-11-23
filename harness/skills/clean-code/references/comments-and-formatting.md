# Comments, Formatting, and Structure

Load this when deciding whether a comment earns its place, when you find commented-out or dead code, or when file layout and ordering hurt readability.

## Comments

The best comment is the one you didn't need to write because the code says it. A comment is a small failure to express the intent in code — sometimes a necessary one, but always weigh expressing it in the code first. Comments also rot: code changes, comments get forgotten, and a stale comment is worse than none because it lies.

**Good comments** earn their place: explaining *why* (the non-obvious rationale, a workaround for a known bug, a business rule), warnings of consequences, `TODO`/`FIXME` with enough context to act on, legal/license headers, and clarifying intent that the code genuinely can't carry.

```ts
// Good: explains a non-obvious WHY the code can't show
// Stripe rounds half-up; we round half-even to match the ledger system,
// otherwise reconciliation drifts by a cent on large batches.
const rounded = roundHalfEven(amount);
```

**Bad comments** are noise or lies: redundant restatements of the code, commented-out code (delete it — version control remembers), and misleading or outdated comments.

```ts
// Bad: says nothing the code doesn't
i++; // increment i
```

**Delete dead code.** Commented-out blocks and unreachable functions make readers wonder if they matter. They don't — version control has them if you ever need them back. Carrying them rots trust in the whole file.

## Formatting and structure

Consistent formatting lets readers focus on logic instead of re-parsing layout. Most of this should be automated so it never reaches review.

**Vertical density and distance.** Lines that are tightly related belong together; blank lines separate distinct thoughts. Declare variables close to where they're used, not in a clump at the top. Closely related concepts should be vertically near each other so the reader doesn't scroll to connect them.

**Caller above callee.** Order functions so the high-level entry point reads first and the helpers it calls follow below, like a newspaper: headline first, details after. The reader meets the overview before the mechanics.

**Consistent style via tooling.** Use Prettier (or equivalent) for formatting and ESLint (or equivalent) for rules, wired into CI and pre-commit. Style debates in review are a waste of human attention; let the machine decide so people review behavior. Match the existing project's configuration rather than imposing your own.
