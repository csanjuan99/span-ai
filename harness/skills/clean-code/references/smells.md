# Code Smells and Their Refactors

Load this when a smell from the Decision Gates table is present and you need the reasoning and the matching refactor.

Smells are hints, not proofs. Each points to a likely improvement; confirm the cost is real before acting.

- **Long function** — does too much. → Extract Function until each does one thing.
- **Large class** — too many responsibilities. → Extract Class along responsibility lines (see SRP in `../../solid-principles/SKILL.md`).
- **Long parameter list** — too many positional args. → Introduce a parameter object; pass a whole object instead of its fields.
- **Duplicated code** — same knowledge in several places. → Extract a shared function/module (respecting the rule of three).
- **Feature envy** — a method uses another object's data more than its own. → Move Method to the class that owns the data.
- **Primitive obsession** — strings/numbers standing in for domain concepts (`string` for an email, `number` for money). → Introduce a small value type (`Email`, `Money`) that centralizes validation and behavior.
- **Shotgun surgery** — one change forces edits across many files. → Gather the scattered responsibility into one place.
- **Divergent change** — one module edited for many unrelated reasons. → Split it so each part has a single reason to change (the SRP cure; the inverse of shotgun surgery).
- **Comments as deodorant** — a comment explaining confusing code. → Refactor or rename so the comment becomes unnecessary, then delete it.
- **Dead code** — unreachable or unused. → Delete it; version control remembers.
- **Magic numbers/strings** — unexplained literals. → Name them as constants (`const MAX_RETRIES = 3`) so they're searchable and self-documenting.

```ts
// Magic number → named constant
// Before
if (attempts > 3) throw new Error("too many");
// After
const MAX_LOGIN_ATTEMPTS = 3;
if (attempts > MAX_LOGIN_ATTEMPTS) throw new Error("too many");
```
