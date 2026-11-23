# Mitigations

Pick one move that drops the score. Do not list a catalog and leave the choice implicit. The move you pick is the one in section 8.

## Pin

A pin is a check that fails when a neighbor invariant fails. It is the feedback signal moved to the day of the change.

A pin qualifies when all of these hold:

- It asserts the neighbor's existing outcome, not the new feature's happy path.
- It sits on the shared surface (the record, the contract, the flag), as close to the write as the system allows.
- Its name includes the behavior it protects, so a later session can search it.
- It runs before this change is called done.

A check that only exercises B, while B and X share a surface, does not qualify. That is the isolated test that misses the interaction.

If a pin already exists, name it. Do not add a second one with the same assertion.

## Isolate

Isolation drops a High score when the new behavior stops writing the neighbor's surface.

- Give the new behavior its own state. It may read the neighbor through the current contract.
- Prefer Extend: a new branch, message, or entry whose unknown values current consumers ignore.
- If a current consumer cannot ignore the unknown, call the change Modify and keep the score.

A new folder, module, or service that writes the same record is not isolation. Cells that share state are one cell. A bad update then reaches every consumer of that state.

## Shrink

Keep one behavior in the change.

- Split a stored-shape change from the feature that wants it. The shape change stays High until every reader is in the estimated set and is either updated in that same change or pinned.
- Leave renames and reformats for their own change once behavior is untouched.
- Do not update the neighbor "while you are here" unless that update is the declared change and its invariant is rewritten on purpose.

## Roll back

Name one step that restores the previous behavior: revert this change, turn this flag off, or read the previous stored shape.

These do not count as rollback:

- "We will fix it forward."
- An in-place overwrite of existing stored data with no way to read the old shape.
- A schema change that current readers cannot survive for the length of a deploy.

If you cannot name the step, the score stays High.

## What does not drop the score

- Editing every file in a long structure list.
- Explaining the ripple in the analysis and then implementing the High change.
- Adding a test that only covers the new behavior.
- Splitting the work across two sessions so each session's diff looks small. The interaction remains. The pin or the isolation has to land with the change that creates the shared surface.
