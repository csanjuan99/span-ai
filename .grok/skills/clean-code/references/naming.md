# Naming and Meaning Over Cleverness

Load this when a name needs a comment to be understood, when code is dense or abbreviated, or when reviewing identifiers in new code.

## The core idea: meaning over cleverness

Code is read far more often than it is written — by reviewers, by teammates, by you in six months. Every minute saved writing a clever one-liner is paid back many times over in confused reading. Optimize for the next reader's comprehension, not for keystrokes or for showing off what the language can do.

```ts
// Before: clever, dense, hostile to the reader
const r = u.filter(x => x.a && !x.d).map(x => x.id);

// After: the intent is on the surface
const activeUserIds = users
  .filter(user => user.isActive && !user.isDeleted)
  .map(user => user.id);
```

The "after" is longer and that is fine. Cost prevented: the reader no longer has to mentally execute the code to discover what `r`, `u`, `x`, `a`, and `d` mean.

When you reach for cleverness — a bit-twiddling trick, a dense reduce, a regex — leave a short comment explaining the *why*, or reconsider whether the clever version earns its cost.

## Naming

Names are the densest documentation you have. A good name removes the need for a comment.

**Intention-revealing.** The name should answer why it exists, what it does, and how it is used. If it needs a comment to explain, it is the wrong name.

```ts
// Before
const d = 30; // days until expiry
// After
const daysUntilExpiry = 30;
```

**Avoid abbreviations, encodings, and noise.** Spell words out (`message`, not `msg`; `index`, not `idx` unless idiomatic in a tight loop). Skip Hungarian-style type prefixes (`strName`, `iCount`) — modern tooling and types already tell you the type, and the prefix lies the moment the type changes. Drop noise words: `theUser`, `userData`, `userInfo`, `userObject` all usually just mean `user`.

**Pronounceable and searchable.** `generationTimestamp` beats `genymdhms` — you can say it in a code review. Single letters and bare numbers are nearly unsearchable; prefer a named constant you can grep for over a literal buried in code.

**Consistent vocabulary.** Pick one word per concept and stick to it. Don't mix `fetch`, `get`, and `retrieve` for the same operation, or `User` and `Customer` for the same entity. Inconsistency forces readers to wonder whether the difference is meaningful.

**Names reflect their level of abstraction.** High-level code reads in domain terms (`processPayment`); low-level helpers can be more mechanical (`appendChecksumByte`). A class named `OrderManager` whose methods are `doStuff` and `handleIt` mixes a domain noun with contentless verbs.

**Booleans read as predicates.** Prefix with `is`, `has`, `should`, `can` so conditions read like sentences: `if (user.isVerified)`, `if (cart.hasItems)`. Avoid negatives in the name (`isNotReady` makes `!isNotReady` a puzzle).

**Functions are verbs, classes are nouns.** `calculateTotal()`, `sendEmail()` for functions; `Invoice`, `EmailService` for classes/types. A function named `data()` or a class named `Manage` signals the abstraction wasn't thought through.

```ts
// Before
class Mgr {
  proc(u: any, f: boolean) { /* ... */ }
}
// After
class SubscriptionService {
  renew(subscription: Subscription, options: RenewalOptions) { /* ... */ }
}
```

Tradeoff: don't over-engineer names. Loop counters `i`, `j` and a lambda's `x` in a tiny, obvious scope are fine — short scope tolerates short names. Reserve the effort for names that travel.
