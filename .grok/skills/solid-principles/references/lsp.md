# Liskov Substitution Principle (LSP)

Load this when a subtype needs `instanceof` checks, throws "not supported", or overrides that weaken what the base type promised.

## Definition

Barbara Liskov (1987) stated it as: if for each object `o1` of type `S` there is an object `o2` of type `T` such that programs defined in terms of `T` behave the same when `o1` is substituted for `o2`, then `S` is a subtype of `T`. In practice: objects of a supertype shall be replaceable with objects of its subtypes without breaking the program. Liskov and Wing (1994) made it behavioral: subtypes may not strengthen preconditions, weaken postconditions, or break invariants and history constraints of the supertype.

## Violation signals

- Callers check `instanceof` or a type tag before using a subtype, or catch exceptions only some subtypes throw.
- An override throws `NotSupportedError`, returns `null` where the base returned a value, or ignores an argument.
- A subtype silently changes side effects the base guaranteed (a `setWidth` that also changes height).
- Tests written against the base type fail when run against a subtype.

## Refactor

1. Write down the base type's contract: preconditions, postconditions, invariants.
2. For each subtype, find the clause it breaks.
3. Choose a fix: make the value immutable so the invariant cannot drift, split into separate types that share only the honored subset, or replace inheritance with composition.
4. Remove the `instanceof` branches from callers; if any must stay, the hierarchy is still wrong.

## Example

```ts
// Before: Square extends Rectangle and breaks the width/height contract.
class Rectangle {
  constructor(public width: number, public height: number) {}
  setWidth(w: number) { this.width = w; }
  area() { return this.width * this.height; }
}
class Square extends Rectangle {
  setWidth(w: number) { this.width = w; this.height = w; } // surprise
}
function grow(r: Rectangle) { r.setWidth(r.width + 1); }   // area math now wrong for Square

// After: immutable values with separate types; no substitution promise is made.
interface Shape { area(): number }
class ImmutableRectangle implements Shape {
  constructor(readonly width: number, readonly height: number) {}
  withWidth(w: number) { return new ImmutableRectangle(w, this.height); }
  area() { return this.width * this.height; }
}
class ImmutableSquare implements Shape {
  constructor(readonly side: number) {}
  area() { return this.side ** 2; }
}
```

Callers depend on `Shape`, which both types honor fully; the mutating contract that Square could not keep no longer exists.

## Tradeoffs and when not to apply

- Splitting a hierarchy can duplicate code; accept small duplication over a lie in the type system.
- Mathematical "is-a" is not behavioral "is-a"; model behavior, not taxonomy.
- Do not add defensive checks in the base to accommodate a bad subtype; fix the subtype or the hierarchy.
- With structural typing (TypeScript), a type can satisfy an interface by accident; still verify behavior with the base type's tests.

## Related

- `isp.md`: subtypes that throw "not supported" are often forced to by a fat interface.
- `ocp.md`: honored contracts are what let OCP extensions plug in without new conditionals.
