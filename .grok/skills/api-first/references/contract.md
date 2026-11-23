# The contract

Fill `assets/contract-template.md` from this file. The contract is the caller's view of one operation. An operation is a command, a query, an event, or a job. The wire protocol is not the contract.

## Required

These block implementation when the value is `unknown`.

| Field | Ready when |
| --- | --- |
| Consumer | A real caller and the job they are finishing. "The frontend" is enough only if that client is the one you can see in the repo. "Anyone" is `unknown`. |
| Operation | A name the caller would use, plus one sentence of the outcome. |
| Input | Every field the caller sends. Required or optional. Type. Unit, currency, or precision where a bare number would be ambiguous. Default stated, or `none`. |
| Output | Every field the caller receives on success, and what they may rely on. |
| Errors | Every failure the caller is expected to handle. A stable code, the condition, and whether retrying is safe. Include invalid input and not allowed when those can happen. |
| Side effects | The state that changes, or `none`. |
| Auth | Who may call and what they present. `Public, no credential` is complete. A guess is not. |
| Compatibility | What stays stable, and what would break a caller. |
| Examples | One success pair and one error pair whose fields exist in Input, Output, and Errors. |
| Boundary | The module that owns the contract types, and the internal names that must not appear in them. |

## Friendly

A contract is friendly when a caller who has never seen the repo can code a client from it. Before status Ready, all of these are true:

- No field is a table name, a column, an ORM field, or an internal type name.
- No error carries a stack trace, a query, or an internal exception name.
- Absent and null are not both used for the same meaning. State which one.
- A list names its maximum page size, or the operation returns one object.
- The success example and the error example use only fields declared above.

Rename in the contract first. Do not rename by leaking the internal model and documenting it.

## Conditional

Omit the mechanism, and write `Not applicable` plus the reason, when the row does not apply.

| Add this | When |
| --- | --- |
| Idempotency | The caller may retry an operation that writes. Name the token field and say a replay does not apply the effect twice. |
| Pagination | The output is a collection that can grow. Name the request token, the response token, the item list, and the maximum page size. The caller stops when the response token is absent. |

Do not add a version field because a guide says so. Record a version only when this repo already versions this capability, or when Compatibility says the change would break an existing caller. A new capability with no existing caller does not need a version.

## What this skill does not add

URI style, status-code catalogs, hypermedia, SDK generation, and a new specification language. Those belong to the repo's sibling, or they stay out.

## Mapping

The boundary accepts the contract input and returns the contract output or a contract error. Storage and domain types stay behind that mapping. One sibling shows where the mapping lives. Follow that file. Do not put the mapping inside the domain rule, and do not return the stored record.
