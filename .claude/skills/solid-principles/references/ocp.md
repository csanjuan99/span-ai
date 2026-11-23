# Open/Closed Principle (OCP)

Load this when adding a new case means editing the same `switch`, `if` chain, or enum-driven conditional every time.

## Definition

"Software entities (classes, modules, functions) should be open for extension, but closed for modification." Coined by Bertrand Meyer in Object-Oriented Software Construction (1988) and popularized by Robert C. Martin (2000), who reframed it around abstraction: new behavior is added by adding code that plugs into a stable abstraction, not by editing tested code.

## Violation signals

- The same conditional on a type tag (`kind`, `status`, `provider`) appears in several functions and grows with every feature.
- Adding a variant requires touching many files and re-testing unrelated branches.
- Bug fixes in the shared conditional cause regressions in cases that were not meant to change.
- Pull requests for "add X support" mostly consist of new `case` branches in existing files.

## Refactor

1. Identify the axis of variation (the tag the conditional dispatches on).
2. Define the stable abstraction: an interface, a function type, or a map key.
3. Move each branch into its own implementation (class, function, or module).
4. Replace the conditional with polymorphic dispatch or a handler map; new cases register themselves without editing the dispatcher.
5. Keep an explicit default or exhaustiveness check so unknown cases fail loudly.

## Example

```ts
type Method = "card" | "bank" | "wallet";

// Before: every new payment method edits this function.
function feeForBefore(method: Method, amount: number): number {
  switch (method) {
    case "card": return amount * 0.029 + 0.3;
    case "bank": return 0.8;
    // add "wallet" here... and in refund(), and in validate()
    default: throw new Error(`unsupported method: ${method}`);
  }
}

// After: the dispatcher is closed; adding "wallet" adds one entry.
type FeeRule = (amount: number) => number;
const feeRules: Record<Method, FeeRule> = {
  card: (amount) => amount * 0.029 + 0.3,
  bank: () => 0.8,
  wallet: (amount) => amount * 0.015,
};
function feeFor(method: Method, amount: number): number {
  return feeRules[method](amount);
}
```

`Record<Method, FeeRule>` makes the compiler report a missing entry when `Method` gains a member, so extension stays safe.

## Tradeoffs and when not to apply

- A conditional that has never changed is not a violation; OCP protects against churn you have observed, not churn you imagine.
- Polymorphism spreads one algorithm across many files. For two or three stable cases, a flat `switch` is easier to read.
- Full OCP (no modification ever) is unreachable; aim to close the parts that change most often.
- Do not build plugin registries "for later"; that is speculative generality (YAGNI).

## Related

- `srp.md`: a growing switch usually also mixes responsibilities per branch.
- `lsp.md`: the implementations you plug in must honor the abstraction's contract, or callers regain the conditionals.
- The `software-design-patterns` skill covers Strategy, State, and Chain of Responsibility, the usual OCP mechanisms.
