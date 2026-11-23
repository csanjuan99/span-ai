# Contract: <operation name>

**Status:** Ready | Blocked
**Consumer:** <who calls, and the job they are doing>
**Carrier:** <openapi | graphql | proto | smithy | types | other> at `<path>`

## Operation

<Name in the caller's language. One sentence: what the caller gets done.>

## Input

| Field | Required | Type and unit | Default |
| --- | --- | --- | --- |
| <name> | yes or no | <type, unit, currency, or precision> | <value or none> |

## Output

| Field | Type and unit | Meaning for the caller |
| --- | --- | --- |
| <name> | <type> | <what the caller may rely on> |

## Errors

| Code | When | Retry |
| --- | --- | --- |
| <stable code> | <condition the caller can recognize> | yes or no |

## Side effects

<State that changes, or "none". One line per effect.>

## Auth

<Who may call, and what they must present. "Public, no credential" is a known answer. `unknown` blocks.>

## Compatibility

- Callers may rely on: <fields, codes, and ordering that stay stable>
- Breaking change: <what would force a caller to change>

## Idempotency

<Not applicable, and why. Or the token field and how long a replay is discarded.>

## Pagination

<Not applicable, and why. Or the page size bound, the request token, and the stop condition.>

## Examples

### Success

<Input the caller sends, and the output they receive.>

### Error

<Input the caller sends, the error code, and the output.>

## Wire

<How the caller reaches this operation. Copied from `<sibling path>`, or Assumption.>

## Boundary

<Where the contract types live, and where they map to internal types. Internal names that must not leak.>

## Companion skills

<none, or each named skill and the constraint it added. Conflicts: which skill won.>

## Checks

| Example | Check | Result |
| --- | --- | --- |
| Success | <name or "no runner"> | pending, pass, fail, or not run |
| Error | <name or "no runner"> | pending, pass, fail, or not run |
