# Interface Segregation Principle (ISP)

Load this when implementers stub or throw in methods they do not need, or when clients depend on interface members they never call.

## Definition

"Clients should not be forced to depend on methods they do not use" (Robert C. Martin, 2000). A fat interface couples every client to every member; a change for one client forces recompilation, re-testing, or dummy implementations for all others. Prefer several small, client-specific (role) interfaces over one general-purpose interface.

## Violation signals

- Implementations contain `throw new Error("not supported")` or empty method bodies.
- A test double must implement a dozen members to satisfy one call.
- Changing a method used by one consumer breaks the build for unrelated consumers.
- Interface names are generic (`IService`, `Repository`) and members span reading, writing, reporting, and administration.

## Refactor

1. List each client and the members it actually calls.
2. Group members by client role: reader, writer, notifier, administrator.
3. Declare one interface per role; a class may implement several.
4. Change each client's parameter type to the narrowest role interface it needs.
5. Delete stubbed members from implementations that no longer need them.

## Example

```ts
type Order = { id: string; total: number };

// Before: every consumer and every fake depends on all five members.
interface OrderStore {
  find(id: string): Order | undefined;
  list(): Order[];
  save(order: Order): void;
  delete(id: string): void;
  exportCsv(): string;
}
class ReadOnlyOrderStoreBefore implements OrderStore {
  constructor(private readonly rows: Map<string, Order>) {}
  find(id: string): Order | undefined { return this.rows.get(id); }
  list(): Order[] { return [...this.rows.values()]; }
  save() { throw new Error("not supported"); }
  delete() { throw new Error("not supported"); }
  exportCsv() { throw new Error("not supported"); }
}

// After: role interfaces; the report only needs to read.
interface OrderReader { find(id: string): Order | undefined; list(): Order[] }
interface OrderWriter { save(order: Order): void; delete(id: string): void }
interface OrderExporter { exportCsv(): string }

class ReadOnlyOrderStore implements OrderReader {
  constructor(private readonly rows: Map<string, Order>) {}
  find(id: string): Order | undefined { return this.rows.get(id); }
  list(): Order[] { return [...this.rows.values()]; }
}
function buildReport(orders: OrderReader): number { return orders.list().length; }
```

A fake for `buildReport` now implements two methods, and the compiler proves the report never mutates orders.

## Tradeoffs and when not to apply

- Too many one-method interfaces fragment the model; stop at the roles clients actually have.
- If every client uses every member, the interface is already cohesive; do not split it.
- In TypeScript, structural typing lets you declare a narrow parameter type without changing the implementation; use that before reorganizing classes.
- Do not segregate purely to shrink test doubles when a lightweight fake of the whole interface is cheap.

## Related

- `lsp.md`: "not supported" stubs are simultaneously ISP and LSP violations.
- `dip.md`: the abstraction a high-level module depends on should be a role interface, sized for that module.
