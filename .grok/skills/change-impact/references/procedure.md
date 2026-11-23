# Trace and score

## Before the trace

Search the repository for:

1. The identifiers of each candidate surface (field, event, route, flag, permission, copy key, status value).
2. The neighbor behavior's name in tests, docs, and `# Impact:` notes.
3. An existing pin: a check whose name states the neighbor invariant.

Record the search in the ripple map. A trace you did not run is `unknown`, not `clear`.

Use a call or dependency index when the repo has one. Then still search for state, time, and control. Those do not show up as imports.

## Starting set and estimated set

The **starting set** is every element this change will Create, Modify, or Extend.

The **estimated set** starts from the starting set. For each element, add a hop when you find any of these:

- a caller or entry point (screen, handler, job, parent)
- a callee whose contract this change relies on
- another reader or writer of the same state
- a publisher or subscriber of the same message
- another producer or consumer of the same payload
- the same permission, flag, route, or copy key
- a check that encodes the current behavior

Stop a branch only when you can say why the next element cannot observe the change. "Different file" is not a stop.

Each row is `Observed` (seen in code, test, or doc in this session) or `Assumption` (inferred from the request, not from evidence).

## Change interaction

After the estimated set exists, ask: which behavior that already shipped uses the same surface?

That behavior is X in the day-1 / day-30 story, even if this task is named B and you will not open X's files. Its invariant goes in section 5.

If the edit also renames, reformats, rewrites a schema, or "cleans up" a neighbor, list that as a second change in section 6. Score the bundle at least one step higher than the feature alone, and High if the second change has no separate rollback.

## Score

Use the highest row that matches. A later row does not apply once an earlier one does.

| Ripple | When |
| --- | --- |
| High | Any one of these is true. The change Modifies a contract or stored shape that another behavior reads. It makes two behaviors share mutable state they did not share. Order, retry, permission, or a flag now couples them and no pin exists. A surface this change writes or modifies is `unknown`. A second change is bundled and cannot be rolled back alone. |
| Medium | None of the High conditions. This change writes or modifies a shared surface, and at least one invariant on that surface has no pin yet. |
| Low | None of the above. Every shared surface is listed. This change writes no shared state and leaves existing contracts alone, or every invariant it can falsify already has a pin. One step rolls it back. |
| None | No shared surface. Behavior, contract, state, flow, time, and control stay as they are. A format, comment, or rename with evidence qualifies. |

Reading a stable contract without writing it does not raise the score. The ripple comes from what this change can falsify.

## Verdict

| Ripple | Verdict | Edit |
| --- | --- | --- |
| None or Low | Proceed | Allowed after the analysis is in the reply |
| Medium | Pin first | Allowed once the pin in section 8 is part of the same change |
| High | Stop | Not allowed. Propose one isolation or one pin, then wait |

No-code requests: a Modify is High. A Create is High when you cannot see the readers of the state it writes. A Create that only reads a contract it does not change, and writes nothing shared, is Low with every hop marked Assumption.
