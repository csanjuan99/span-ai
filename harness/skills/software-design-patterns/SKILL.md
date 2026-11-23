---
name: software-design-patterns
description: "Trigger: design pattern, Factory, Builder, Singleton, Adapter, Decorator, Facade, Strategy, Observer, Command, State, 'patrón de diseño', or a problem they solve (swap algorithms, notify many objects). Select and apply the 23 GoF patterns."
license: "Proprietary"
metadata:
  author: "Carlos San Juan"
  version: "2.0.0"
  category: "design"
---

## Activation Contract

Use when the user names a GoF pattern, asks which fits, wants to refactor toward one, or describes a problem a pattern solves without naming it. Not for folder structure (bounded-contexts) or readability (clean-code).

## Hard Rules

- Apply a pattern only when its specific problem is present in the code, never because it is available.
- Prefer a plain function, `switch`, or small class when simpler and unlikely to change.
- Do not abstract preemptively. Wait for real or clearly imminent variation.
- Load only the one category reference the problem needs.
- Treat Singleton as a likely anti-pattern. Justify global state before proposing it.
- Examples in TypeScript; concepts language-agnostic.

## Decision Gates

| Problem is about | Category | Load |
|---|---|---|
| Making objects (which class, how built, how many) | Creational | `references/creational.md` |
| Arranging existing objects into structures | Structural | `references/structural.md` |
| How objects behave and communicate | Behavioral | `references/behavioral.md` |

| Symptom | Pattern |
|---|---|
| Concrete class unknown until runtime; subclass decides | Factory Method |
| Interchangeable families of products used together | Abstract Factory |
| Telescoping constructor; many optional parts | Builder |
| Creation expensive or concrete class unknown; need copy | Prototype |
| Exactly one shared resource must exist | Singleton |
| Integrating an API whose interface does not match | Adapter |
| Class explodes across two independent dimensions | Bridge |
| Tree structure; clients should ignore leaf vs branch | Composite |
| Add responsibilities dynamically, in combinations | Decorator |
| Complex subsystem needs one simple entry point | Facade |
| Huge number of similar objects; memory is bottleneck | Flyweight |
| Lazy loading, access control, caching, remote stand-in | Proxy |
| Several handlers may process a request; decouple sender | Chain of Responsibility |
| Undo/redo, queuing, logging of actions | Command |
| Uniform traversal regardless of storage | Iterator |
| Web of objects referencing each other | Mediator |
| Snapshots/undo without exposing internals | Memento |
| One change must broadcast to many subscribers | Observer |
| `if`/`switch` on a status field everywhere | State |
| Choose one of several algorithms at runtime | Strategy |
| Variants share a procedure, differ in steps | Template Method |
| Many unrelated operations over stable element classes | Visitor |
| Repeatedly interpret a simple, well-defined language | Interpreter |

## Execution Steps

1. Restate the design problem in one sentence. Load `references/overview.md` only if the concept is unclear.
2. Pick the category from the first table, then the candidate pattern from the second.
3. Load only that category's reference and read only the candidate's section.
4. Check Applicability, Pitfalls, and Relations. Switch if a related pattern fits better.
5. Draft the simplest version that relieves the observed pain.
6. Name the plain-code alternative rejected and why, or return "no pattern needed".

## Output Contract

Return:
- Chosen pattern or "no pattern needed", with the reason tied to the observed problem.
- Minimal TypeScript sketch.
- Tradeoffs and pitfalls for this application.
- Related patterns considered and why not chosen.

## References

- `references/overview.md` — concept, value, overuse risks, category guide, one-line intents. Load when concept or category is unclear.
- `references/creational.md` — Factory Method, Abstract Factory, Builder, Prototype, Singleton. Load for creation problems.
- `references/structural.md` — Adapter, Bridge, Composite, Decorator, Facade, Flyweight, Proxy. Load for composition problems.
- `references/behavioral.md` — Chain of Responsibility, Command, Iterator, Mediator, Memento, Observer, State, Strategy, Template Method, Visitor, Interpreter. Load for interaction problems.
