# Strict TypeScript Runtime Policy

Use this reference for every TypeScript authoring or review task governed by the `code-quality` skill.

## Typecheck Gate

1. Inspect `package.json`, workspace tooling, and the effective `tsconfig` chain.
2. Run the repository's dedicated typecheck script when it invokes TypeScript without emitting. Otherwise run `tsc --noEmit` with the repository's normal project selection.
3. Report the exact command and exit result. Passing lint, tests, transpilation, or bundling is not evidence that TypeScript typechecking passed.
4. Do not weaken compiler options, exclude files, add suppressions, or change project references merely to make the gate green.

Prefer `strict: true`. Inspect these options explicitly because they close gaps that `strict` alone may not cover:

| Option | Safety property |
| --- | --- |
| `noUncheckedIndexedAccess` | Indexed reads include `undefined` unless presence is proven |
| `exactOptionalPropertyTypes` | Optional absence is distinct from an explicit `undefined` value |
| `noImplicitOverride` | Overrides remain explicit when base classes change |
| `noFallthroughCasesInSwitch` | Accidental switch fallthrough is rejected |
| `useUnknownInCatchVariables` | Caught values require narrowing before use |

Treat missing options as findings, not automatic edit authority. Change `tsconfig` only when the requested scope authorizes it and the repository can absorb the stricter errors.

## Modeling Matrix

| Need | Default | Verification |
| --- | --- | --- |
| Extensible object contract | `interface` | Extension is intentional; declaration merging cannot surprise the module |
| Object contract that must stay closed | `type` alias | No consumer relies on declaration merging |
| Union, tuple, primitive alias, mapped or conditional type | `type` | The construct cannot be represented more clearly as an interface |
| Closed state or message variants | Discriminated union | Every member has one stable discriminant; consumers check exhaustiveness with `never` |
| Named runtime values with zero enum semantics | Object literal plus `as const` | Values and derived union type stay aligned |
| Runtime API that specifically benefits from enum identity | String enum | Runtime object is intentional and interoperability requirements are documented |

Do not impose “enums always.” Avoid numeric enums because reverse mappings and implicit numeric values are easy to misuse. Avoid heterogeneous enums because mixed value domains weaken reasoning.

Exhaustiveness pattern:

```typescript
function assertNever(value: never): never {
  throw new Error(`Unexpected variant: ${String(value)}`)
}
```

Call `assertNever` from the unreachable/default branch of a switch over a closed discriminated union.

## Contract Ownership and Translation

- Define one authoritative contract for each semantic concept within its owning boundary. Do not create parallel aliases whose meanings can drift.
- Treat transport DTOs, SDK responses, persistence rows, and domain models as different contracts when their semantics or trust levels differ.
- Validate external data at the adapter boundary, then map it into the internal domain contract. Do not pass an unvalidated wire shape through services merely because its fields currently resemble the domain model.
- When a contract crosses layers, make ownership and translation explicit. Shared types are appropriate only when the shared semantics are genuinely identical and validated at entry.

## Root-Cause Repair Workflow

1. Locate the authoritative contract and the first boundary where the value becomes inaccurate or untrusted.
2. Trace producers, adapters or guards, and nearby consumers before editing aliases or call sites.
3. Repair the owning contract or boundary translation instead of adding downstream assertions, duplicated optional fields, or broader unions that conceal the mismatch.
4. Run the dedicated typecheck, then run focused tests for the changed contract path and inspect nearby consumers.

## Bulk-Change and Merge Gate

Before a merge resolution, codemod, search-and-replace, or bulk type edit:

- identify the authoritative owner and every affected adapter or consumer;
- prove the replacement preserves semantic meaning, not only assignability or matching field names;
- stop and split the change when one mechanical transform cannot preserve those semantics;
- after the change, inspect nearby consumers and run focused regression tests plus the dedicated typecheck.

## Boundary and Escape Policy

- Type inputs from JSON, storage, environment variables, third-party SDKs, postMessage, and network responses as `unknown` until runtime validation or narrowing proves the domain type.
- Prefer schema validation or focused type guards at the boundary. Do not cast after parsing and call that validation.
- Add no explicit `any`, inferred implicit `any`, `as Type`, angle-bracket assertion, non-null `!`, double assertion, or `@ts-ignore` in changed code. Allow `as const` only for a deliberate literal-object modeling choice and report that choice; never use it as boundary validation.
- A rare assertion or non-null exception must be the smallest possible expression and include a nearby rationale naming the invariant plus the runtime check that establishes it. Report it under `escapes`; do not hide it behind a helper.
- Use `@ts-expect-error` only for an intentional negative contract or verified external typing defect. Add a reason and a test/typecheck gate that fails when the suppression is no longer needed.
- Type exported APIs and meaningful callback parameters/returns where inference would hide a contract. Keep local inference for obvious implementation details; redundant annotations are not safety.

## SDK and External Response Guidance

- Treat SDK and external responses as untrusted until runtime evidence establishes the required shape. Generated SDK types improve compile-time ergonomics but do not validate runtime data.
- Reject assertion-only fixes such as `response as DomainModel`. Validate the response and map it through an adapter into the authoritative domain contract.
- If an upstream typing defect forces a localized assertion, first validate the relevant runtime invariant, document the defect and containment, and keep the assertion out of domain code.

## Nullish and Semantic Validation

- Use `value == null` only when both `null` and `undefined` are invalid by design; otherwise check the intended state explicitly.
- Do not use generic falsy checks for values where `0`, `false`, or `""` are valid. Validate the semantic constraint, such as finite number, non-empty identifier, or explicit boolean presence.
- Model missing, empty, invalid, and unavailable states separately when consumers behave differently.

## Review Evidence

Before returning `pass`, verify all of the following in the changed scope:

- Dedicated TypeScript typecheck passed.
- No new explicit or implicit `any` exists.
- No unjustified assertion, non-null assertion, double assertion, or suppression exists.
- Every untrusted boundary validates or narrows `unknown` before use.
- External DTOs are validated and translated into their authoritative internal contracts; no assertion-only response fix remains.
- Closed variants have an exhaustive `never` proof.
- Every enum is deliberate; numeric or heterogeneous enums have explicit compatibility evidence.
- Compiler-option gaps and unauthorized configuration changes are reported.
- Contract, guard, adapter, and state-variant changes have focused regression results and nearby-consumer inspection evidence.
