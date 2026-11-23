---
name: code-quality
description: "Trigger: writing or reviewing TypeScript, code quality, strict typing, no any. Enforces observable structure, type safety, and typecheck gates."
license: "Proprietary"
metadata:
  author: "Carlos San Juan"
  version: "3.1.0"
  category: "code-quality"
  risk: "safe"
  date_added: "2026-06-25"
---

## Activation Contract

Load when writing, reviewing, or refactoring TypeScript, or when the user asks for code quality, strict typing, or no `any`. Apply these rules above generic preferences when the project adopts this skill.

## Hard Rules

1. Use guard clauses, no `else` after `return`, and at most two conditions per `if`; extract a named predicate for three or more.
2. Give each unit one responsibility. Handlers orchestrate, services hold business logic, repositories access data, and adapters call external APIs. Split descriptions containing “and” and functions above about 40 lines.
3. Add no explicit or implicit `any`. Start untrusted values as `unknown`, then validate or narrow. Type boundaries, exports, and relevant callbacks; allow clear, safe local inference. Keep one authoritative contract per semantic concept and boundary; adapters validate and map external DTOs into internal domain contracts instead of sharing unvalidated wire shapes across layers.
4. Add no type assertions, non-null assertions, double assertions, or `@ts-ignore`. A localized exception requires a documented reason and runtime-validated invariant. Allow `as const` only as a reported literal-modeling choice, never boundary proof. Allow `@ts-expect-error` only with a reason and a test or gate proving necessity.
5. Use `interface` for intentionally open object contracts; use `type` for unions, tuples, mapped/conditional types, aliases, and objects that must not merge. Avoid accidental declaration merging. Never default to enums: decide among a discriminated union, `as const` object, and string enum; avoid numeric/heterogeneous enums. Prove closed variants exhaustive with `never`.
6. Run the repository typecheck or `tsc --noEmit`; lint and tests are not substitutes. Prefer `strict: true`; inspect `noUncheckedIndexedAccess`, `exactOptionalPropertyTypes`, `noImplicitOverride`, `noFallthroughCasesInSwitch`, and `useUnknownInCatchVariables`. Never change `tsconfig` without authorization and compatibility evidence.
7. Put rationale-focused JSDoc above every exported function, class, type, hook, service, adapter, and repository. Delete comments that merely restate code.

## Decision Gates

| Situation | Required action |
| --- | --- |
| External, parsed, or dynamic input | Use `unknown`; validate or narrow before access |
| Object contract | Use `interface` only when openness is intentional; otherwise choose the appropriate `type` form |
| Closed variants or named runtime values | Apply the modeling matrix; record enum/runtime rationale and `never` exhaustiveness |
| Typing escape | Remove it, or report its location, reason, invariant, runtime proof, and gate |
| Cross-layer contract change | Identify the authoritative owner, adapters, and nearby consumers before editing; preserve boundary translation |
| Merge, codemod, or bulk type change | Stop unless ownership and semantic equivalence are proven; inspect affected consumers after the change |
| Missing strict option | Report the gap; edit config only with explicit authority |
| Typecheck missing or failing | Return non-pass; never substitute lint or tests |

## Execution Steps

1. Inspect repository scripts and the effective TypeScript configuration before editing.
2. Apply `references/strict-typescript.md`; load `references/examples.md` only for structural examples.
3. Run the repository typecheck, or `tsc --noEmit` when no equivalent exists.
4. When contracts, guards, adapters, or state variants change, run focused tests and inspect nearby consumers for semantic regressions.
5. Read back changed files and search them for forbidden escapes.

## Output Contract

Return:
- `typecheck`: exact command and observed result.
- `strictness`: effective options, gaps, and any authorized config change.
- `escapes`: `none`, or locations with reason, invariant/runtime proof, and gate.
- `modeling`: non-obvious modeling decisions and exhaustiveness evidence.
- `quality`: guard-clause, responsibility, boundary/export/callback typing, and JSDoc findings.
- `regression`: focused tests, nearby consumers inspected, and observed results; use `N/A` only when no relevant contract changed.
- `status`: `pass`, `partial`, or `blocked`; never `pass` after a failed typecheck or unjustified escape.

## References

- `references/strict-typescript.md` — mandatory type-safety matrix, gates, escape policy, and evidence criteria.
- `references/examples.md` — optional structural examples.
