# Single Responsibility Principle (SRP)

Load this when a module keeps being edited for unrelated reasons, or when business rules, persistence, and I/O live in the same unit.

## Definition

"A module should have one, and only one, reason to change" (Robert C. Martin, Agile Software Development: PPP, 2002). Martin later restated it as "a module should be responsible to one, and only one, actor" (Clean Architecture, 2017), where an actor is a group of stakeholders who request changes. The unit is the module or class, not the function; "does one thing" at function level is a Clean Code rule, not SRP.

## Violation signals

- Two unrelated changes (a schema migration and a pricing rule) collide in the same file and the same pull request.
- A class name ends in `Manager`, `Handler`, or `Service` and its methods span math, database access, and email or HTTP.
- Tests for one concern require setting up infrastructure for another (a pricing test needs a database).
- Git history shows the file touched by different teams for different tickets.

## Refactor

1. Group methods by the actor that would request their change (finance, operations, infrastructure).
2. Extract each group into its own class or module with a name that states its responsibility.
3. Keep the original as a thin coordinator only if callers need a single entry point; otherwise update callers directly.
4. Verify each extracted unit can be tested without the others' dependencies.

## Example

```ts
// Before: invoice math AND persistence AND email in one class.
class Invoice {
  calculateTotal(): number { /* pricing rules */ }
  saveToDb(): void { /* SQL */ }
  sendEmail(): void { /* SMTP */ }
}

// After: each class has one reason to change.
class Invoice {
  calculateTotal(): number { /* pricing rules */ }
}
class InvoiceRepository {
  save(invoice: Invoice): void { /* SQL */ }
}
class InvoiceMailer {
  send(invoice: Invoice): void { /* SMTP */ }
}
```

A pricing change now edits `Invoice` only; a mailer swap edits `InvoiceMailer` only; neither change risks the other.

## Tradeoffs and when not to apply

- Over-splitting creates many tiny classes that must be read together; if the pieces always change together, they share one reason to change and belong together.
- A cohesive 200-line module with a single actor is not an SRP violation because of size alone.
- For scripts and prototypes, the extraction cost exceeds the collision cost. Skip it.
- If the "actor" boundary is unclear, wait for the second unrelated change before splitting; the split will then be guided by evidence.

## Related

- `ocp.md`: after extraction, new variants of a responsibility often want an extension point.
- `dip.md`: the extracted infrastructure class (repository, mailer) is the natural place for an interface and injection.
