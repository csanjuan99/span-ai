# Impact: <behavior name>

**Verdict:** Proceed | Pin first | Stop
**Ripple:** None | Low | Medium | High
**Shared surface:** <contract, state, flow, control, or timing — or "none">
**Delay:** Immediate | Next change | Later — <when a missed ripple would show up>

## 1. Change

<One sentence: the behavior this change adds, alters, or removes.>

- **Kind:** Create | Modify | Extend
- **Holds constant:** <the existing behavior that must stay true>

## 2. Boundary

- **Inside:** <behaviors this change owns>
- **Outside:** <behaviors that must stay as they are>
- **Why this boundary:** <what the other side cannot observe, or "the surface is shared">

## 3. Inventory

| Element | Action | What changes for a caller or a reader |
| --- | --- | --- |
| <name> | Create, Modify, or Extend | <observable difference> |

## 4. Ripple map

| Neighbor behavior | Connection | What can break | Evidence |
| --- | --- | --- | --- |
| <behavior, or "none found"> | <lens: contract, state, flow, time, control, structure> | <the failed outcome> | Observed or Assumption |

Searched: <identifiers, names, and places you actually looked>.

## 5. Invariants

1. <Testable sentence. "Behavior X still <outcome> when <condition>.">

## 6. Interaction

<Earlier behaviors that already use this surface. Or "none found" plus the search.>

<Any second change bundled into this edit. Or "none".>

## 7. Score

<One sentence that quotes the matching row in `references/procedure.md`.>

## 8. Mitigation

- **Isolate:** <how the neighbor stops sharing the surface, or "not needed">
- **Pin:** <the check, placed on the shared surface, that fails if the invariant breaks; or "already present: <where>">
- **Shrink:** <the one behavior this change keeps>
- **Rollback:** <the one step that returns the previous behavior>

## 9. Verification

| Invariant | Check | Result |
| --- | --- | --- |
| <n> | <existing or new check> | pending, pass, fail, or not run |

## 10. Handoff

- Do not break: <invariant numbers and the sentence>
- Surface identifiers to search next time: `<id>`, `<id>`
- Pin lives at: <path or check name, or "not in the repo yet">
