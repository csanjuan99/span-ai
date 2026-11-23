# Code Quality — Never/Always Examples

Load this when you need a concrete before/after pattern for the structural rules.

## Rule 1 — Guard Clauses Over Nested `if`

**Never:**
```typescript
if (condition) {
  if (otherCondition) {
    // actual work
  } else {
    return error()
  }
} else {
  return error()
}
```

**Always:**
```typescript
if (!condition) return error()
if (!otherCondition) return error()
// actual work
```

Maximum **two conditions** per `if`. Three or more → extract a named predicate.

Extract complex boolean checks into named predicates.

No `else` after a `return`.

**Never:**
```typescript
if (user.active && user.role === 'admin' && user.tenantId === tenantId) { ... }
```

**Always:**
```typescript
const isAuthorizedAdmin = user.active && user.role === 'admin'
if (!isAuthorizedAdmin || user.tenantId !== tenantId) return forbidden()
```

## Rule 2 — Single Responsibility

Layer contract:

- **Handler** (HTTP, Lambda, controller): orchestrates only — parse → validate → call service → return response. No business logic.
- **Service**: business logic only. No persistence access, no network calls.
- **Repository**: data access only (the database). No business logic.
- **Adapter**: external API calls only. No business logic.

If you can describe a function with "and", split it. Max ~40 lines per function; if longer, extract named helpers.

## Rule 3 — Never `any`

| Instead of | Use |
|-----------|-----|
| `any` | `unknown` + type guard / narrowing |
| `any[]` | `unknown[]` or specific typed array |
| `Record<string, any>` | `Record<string, unknown>` or a proper type |

**Narrowing pattern:**
```typescript
function process(input: unknown): string {
  if (typeof input !== 'string') throw new ValidationError('Expected string')
  return input.trim()
}
```

If the type comes from an external API response, use `unknown` at the boundary and narrow before use.

## Rule 4 — JSDoc Over Inline Comments

Document **above** symbols, never beside code.

**Never:**
```typescript
// Check if user has permission
if (user.role === 'admin') { ... }

const x = computeValue() // returns processed result
```

**Always:**
```typescript
/**
 * Returns the processed result from the raw input.
 * Uses X algorithm because Y constraint.
 */
function computeValue(): ProcessedResult { ... }
```

- Every exported function, class, type, hook, service, adapter, and repository gets a JSDoc block.
- The JSDoc explains **why**, not what (names explain what).
- Inline `//` comments are only for non-obvious workarounds: platform quirks, business invariants, known limitations.
