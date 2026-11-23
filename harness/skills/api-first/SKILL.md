---
name: api-first
description: "Trigger: api first, api-first, api friendly, contrato, terceros, contract first. Defines the consumer contract before backend code, in the repo's language and style."
license: "Proprietary"
metadata:
  author: "Carlos San Juan"
  version: "1.0.0"
  category: "architecture"
---

## Activation Contract

Load when building or changing a backend capability another party can call: a client, another service, a partner, or a job. A private helper with no outside caller is out of scope. If the caller is unclear, treat the capability as public.

API First: the consumer contract exists before the implementation. API Friendly: a caller can use that contract without reading the implementation.

## Hard Rules

- Contract before code. An `unknown` required field blocks implementation. Do not invent the consumer, the auth rule, or an error.
- Shape it from the caller's job. Tables, ORM entities, and internal types stay behind the boundary.
- A stranger can code a client from the contract: names, inputs, outputs, errors, auth, side effects, one success example, and one error example.
- Use the carrier the repo already commits. Do not add OpenAPI, GraphQL, gRPC, Smithy, or another framework. With none, use boundary types in the repo's language.
- Copy wire, errors, names, and layers from one sibling. Do not restyle the codebase.
- Load another skill only when the command names it, and only for the code behind the contract. No names means load none.
- With the repo's test runner, check the success example and the error example. With no runner, say the checks were not run.

## Decision Gates

| Situation | Action |
| --- | --- |
| Private helper, no outside caller | Do not write a public contract. |
| Any required field is `unknown` | Status Blocked. Do not implement. |
| Contract complete, carrier or sibling exists | Status Ready. Implement in that shape. |
| Contract complete, no carrier and no sibling | Status Ready. Boundary types only. Wire is Assumption. |
| Command names skills | Apply them after the contract, in the order named. |
| Command names none | Load none. Local style is not a skill. |
| Two named skills conflict | The one named last wins. Record it. |

## Execution Steps

1. Name the consumer and the job, or stop with that field `unknown`.
2. Follow `references/adapt.md` for language, carrier, one sibling, and named skills.
3. Fill `assets/contract-template.md` from `references/contract.md`.
4. On Blocked, return the template and stop.
5. Map at the boundary, then implement behind it in the sibling's style. Named skills apply here.
6. Run the two example checks and append pass, fail, or not run.

## Output Contract

Return the filled template before any code, then the files changed and the two check results.

## References

- `assets/contract-template.md` — required output.
- `references/contract.md` — load while filling the template.
- `references/adapt.md` — load on every run.
- `references/example.md` — load when the request starts from an internal function or a table.
- `references/sources.md` — load only when asked why a rule exists.
