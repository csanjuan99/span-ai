# Worked example

The same system failure, two surfaces. Day 30 is the moment to run this skill. Day 31 is what happens when the session only checks behavior B.

## The story

Day 1, behavior X ships: a finished export can be read back as ready. The system stores a job record `{ status: ready }` and a screen renders Ready. X writes no channel.

Day 30, the request is behavior B: when a job finishes, notify a channel. The session reuses the job record, changes `ready` to `complete`, and makes `channel` required.

Day 31, export breaks. The screen still waits for `ready`. Old jobs have no channel. The fix lands on X.

B's tests pass on day 30 if they only assert that a notification is sent.

## Filled analysis on day 30

The block below is the analysis this skill returns. Headings stay exactly like this in the reply.

```markdown
# Impact: Notify a channel when a job finishes

**Verdict:** Stop
**Ripple:** High
**Shared surface:** job record (`status`, `channel`)
**Delay:** Next change — export looks broken the first time a notified job is read back. With no pin, that can be a later session.

## 1. Change

When a job finishes, a channel is notified.

- **Kind:** Modify
- **Holds constant:** A finished export can be stored and read back as ready with no channel.

## 2. Boundary

- **Inside:** notification when a job finishes
- **Outside:** export's ready state and any job written without a channel
- **Why this boundary:** it does not hold. Both behaviors read and write the same job record.

## 3. Inventory

| Element | Action | What changes for a caller or a reader |
| --- | --- | --- |
| job.status | Modify | `ready` becomes `complete` |
| job.channel | Modify | absence becomes illegal |
| notification | Create | a new send when a job finishes |

## 4. Ripple map

| Neighbor behavior | Connection | What can break | Evidence |
| --- | --- | --- | --- |
| Export reads as ready | state, contract | finished export never matches `ready` | Observed |
| Jobs stored without a channel | state | old and new exports fail the required field | Observed |

Searched: `status`, `ready`, `channel`, export tests, job writers.

## 5. Invariants

1. Behavior Export still reads a finished export as ready when the job has no channel.

## 6. Interaction

Export (day 1) already owns `job.status = ready`.

Second change bundled: a stored-shape change (`channel` required) inside the notification feature.

## 7. Score

High. The change Modifies a stored shape and a status value that Export reads, and no pin exists.

## 8. Mitigation

- **Isolate:** B writes its own notification record and does not change `job.status` or require `channel` on the job. Export's contract stays. The score drops to Low because B is then a Create that does not write X's state.
- **Pin:** only if the product truly wants one shared status. Then this change must declare Export as inside the boundary, update the invariant on purpose, and add a check that a finished export with no channel still reads as ready. That is a different change than "add notification".
- **Shrink:** notification only. Leave the job schema alone.
- **Rollback:** revert the notification change. A required `channel` column written in place onto existing jobs has no one-step rollback, so it cannot ship inside this change.

## 9. Verification

| Invariant | Check | Result |
| --- | --- | --- |
| 1 | Export-finishes-without-channel-reads-ready | not run — edit blocked |

## 10. Handoff

- Do not break: invariant 1, finished export reads as ready with no channel
- Surface identifiers to search next time: `job.status`, `ready`, `channel`
- Pin lives at: not in the repo yet
```

## Same shape on a screen

Behavior X enables Pay when `cart.total > 0`. Behavior B, "free with a promo", writes `cart.total = 0` for a valid free order. Pay disappears on orders the business wants to accept.

Shared surface: `cart.total` (one field, two meanings: merchandise total and amount due). Verdict Stop.

Isolation that drops the score: B writes `cart.discount` and leaves `total` alone. Pay reads amount due, which considers both, in a declared follow-up if the rule must change. B's own tests for "discount applied" do not protect Pay.
