# What this skill is

Load this when the user asks what the practice is called, or why the analysis exists.

## The practice

**Change impact analysis** is the maintenance activity of deciding what a change will touch and what else that touch can affect. Bohner and Arnold define it as identifying what to modify to accomplish a change, or identifying the potential consequences of a change.

Two sets keep the work honest:

- The **starting impact set** is what you intend to touch.
- The **estimated impact set** is everything that can observe that touch.

The **ripple effect** is the propagation of a modification into the rest of the system. Stability is resistance to that ripple.

## The failure this skill targets

A **change-interaction regression**: behavior X ships, a later change to behavior B shares a dependency with X, and X breaks even though B's own checks pass. The repair then lands on X, which was fine the day it shipped.

Böhme, Oliveira, and Roychoudhury studied this on GNU Coreutils. Regressions often survived for years, most of the long-standing ones came from interactions between changes, and tests that stressed each change alone caught only half of those interaction failures. The dependency between the changes is the defect path. Coverage of B is not a substitute.

A cousin concept is **feature interaction**: two behaviors that are each correct produce a wrong combined behavior. Use the same shared-surface trace for it.

## The system reading

Treat the change as an intervention in a system, not as an edit to a part.

- The **boundary** is the set of behaviors allowed to change.
- A **shared surface** is a stock or a flow that crosses that boundary: a contract, a stored record, a message, a permission, a flag, a timing assumption.
- The day-31 repair is **delayed feedback**. The leverage is to move the signal to the day of the change (a pin on the surface) and to stop two behaviors from sharing the stock (isolation).
- An agent session that optimizes B while leaving X's invariant unstated is local optimization. The part gets better and the system gets a regression.

Repeated sessions make this worse when each one leaves no pin. Complexity accumulates unless the invariant is maintained on purpose.

## Words that are not this practice

**Impact mapping** (Gojko Adzic) is a planning map from a goal to actors, impacts, and deliverables. It does not trace a code change. When someone says "impacto", decide from the request: a proposed code or behavior change uses this skill; a product-strategy map does not.

A blast-radius limit in AWS Well-Architected is the same idea at infrastructure grain: a failure, including a bad deploy, stays inside a boundary, and cells do not share state. This skill uses that idea on behaviors. It does not require AWS, and it is not a Well-Architected review.

## What the analysis refuses to do

- It does not approve an edit by listing files.
- It does not authorize a refactor of the neighbor.
- It does not replace a security review. The control lens can surface a permission issue; hand that off.
- It does not invent a dependency to look thorough. An unknown stays an unknown and raises the score.
