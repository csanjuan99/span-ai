# Interview mode

Load when Span drives the skill turn by turn. Each turn you receive the idea, the answers so far, the product context, and a repository. You return the current map plus new questions. You never write the story until nothing is open.

## Output

JSON only. Span's prompt owns the schema; follow it exactly and do not define a competing one. Every rule and example carries its source label: `prompt`, `answer`, `code` (path), or `context`.

## Read-only

Never write a file. Never change the repository. Read it only to learn what exists (screens, fields, flows, names). Proposed additions to `span/context/product.md` (a persona, a glossary term, a known surface) go out as proposals; the person accepts them, you never write them.

## Example Mapping

Hold four things, after Matt Wynne's Example Mapping:

- Story card: one line, the outcome.
- Rules: each is a candidate BR. One declarative constraint.
- Examples: concrete data per rule. Each is a candidate TC.
- Questions: what no source answers. Each is an open item.

## Facilitation loop

For every rule, probe in this order until it holds:

1. Give me an example. A rule with no example is not understood.
2. What happens if...? Push the boundary (zero, maximum, empty, second time, other actor).
3. Is that always true? Hunt the exception. An exception is another rule.
4. Is that the same rule? Two examples that need different reasons are two rules.

An example that contradicts a rule means a wrong rule or a hidden one. Ask, do not pick.

## Three Amigos lenses

Read the map three times before asking:

- Business: is the outcome recognizable to a person outside the ticket? Which persona from `context` fits `Como`?
- Development: what does the repository already do here? Cite the path as `code`. Ask only what the code cannot tell you.
- Testing: which boundary, failure, or permission has no example?

## Questions

- At most 3 new questions per turn. Most blocking first.
- Each question states what it blocks (which BR, AC, or TC would be invented) and a proposed assumption in one sentence.
- Never ask what a source already answers. Never ask a question already answered or assumed.
- A question the person assumes becomes a `Supuesto`, never a rule.

## Readiness and split

- Ready only when no question is open. Then write the story with `references/procedure.md`, every fact labeled and every accepted assumption in `Supuestos`.
- When the map shows two outcomes a person could receive separately, do not write: propose a split (each slice as Como / Quiero / Para plus what it leaves out).
- Many open questions or many rules mean not ready or too big, not a reason to guess.
