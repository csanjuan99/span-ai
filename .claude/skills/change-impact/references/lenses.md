# Systemic lenses

Walk these seven lenses for every change. The stack does not add or remove a lens. A screen and a service answer the same question with different evidence.

A **shared surface** is any item below that two behaviors both rely on. One shared surface is enough to leave Ripple None.

| Lens | Question | Shared when | UI evidence | Service evidence |
| --- | --- | --- | --- | --- |
| Behavior | What can a person or a caller observe afterward? | Two behaviors promise outcomes that depend on the same rule | A control, message, route, or empty state changes meaning | A response, error, or side effect changes meaning |
| Contract | What is the shape, the defaults, the errors, and the legal order of inputs and outputs? | A reader still interprets the old shape | Props, form payload, route params, stored client object | Request, response, event payload, schema, error code |
| State | What survives the call? | Two behaviors read or write the same record, key, or cache | Store, local storage, URL, cache entry | Row, document, file, cache, already-sent message |
| Flow | Who triggers this, and what does this trigger? | A caller, subscriber, or next step assumes the old path | Navigation, submit handler, parent callback | Caller, queue consumer, webhook, job |
| Time | Does correctness depend on order, retries, freshness, or "eventually"? | One side assumes a write is visible, unique, or ordered | Double submit, stale cache, optimistic UI | Retry, race, idempotency key, async projection |
| Control | Who is allowed, and which flag or config selects the path? | Two behaviors share a permission, flag, tenant, or locale | Feature flag, role gate, i18n key | Auth scope, tenant column, env flag |
| Structure | Which other modules must change for this one to stay correct? | A change forces edits far from the behavior | Many screens edit one rule | Many handlers edit one rule |

## How to record a lens

For each lens write one of:

- `clear` — you looked, and this change does not affect it. Name what you looked at.
- `surface: <id>` — name the identifier another behavior can share.
- `unknown` — you could not look. On a surface this change writes or modifies, `unknown` finishes as High.

## Kind of change

Decide the kind from the lenses, after the walk:

| Kind | Meaning | Typical ripple |
| --- | --- | --- |
| Create | New behavior, new contract. Existing callers keep their old meaning. | The risk is what the new behavior reads or writes, not the new files. |
| Extend | A new variant. Current callers who ignore the unknown keep their old meaning. | If an existing caller cannot ignore the unknown, it is Modify. |
| Modify | Meaning, timing, shape, or stored data of an existing behavior changes. | Direct ripple. This is the day-30 case. |

Adding an optional field is Extend only when every current reader ignores it safely. A reader that treats absence as a default, or that persists the object back, makes it Modify.

## Boundary test

The boundary holds when you can name what is outside it and why that outside cannot observe the change.

These do not hold the boundary:

- "Different folder" or "different service" while both use the same stored record or payload.
- "Different team owns it."
- "The types still compile." Compilation does not protect a changed meaning, default, or order.

A long structure list is shotgun surgery: the rule is smeared across the system. Report it as a shared surface and isolate it. Do not plan to edit every file as the mitigation.
