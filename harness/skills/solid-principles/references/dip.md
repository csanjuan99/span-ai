# Dependency Inversion Principle (DIP)

Load this when business logic instantiates concrete infrastructure, or when a unit cannot be tested without a real database, network, or third-party SDK.

## Definition

"High-level modules should not depend on low-level modules. Both should depend on abstractions. Abstractions should not depend on details. Details should depend on abstractions" (Robert C. Martin, 2000). The direction of source-code dependency is inverted relative to the flow of control: the policy owns the interface, and the infrastructure implements it.

## Violation signals

- `new SendGridClient()`, `new PrismaClient()`, or `fetch(...)` inside a domain or application service.
- Unit tests spin up containers, hit sandboxes, or mock module imports to test a business rule.
- Swapping a provider (email, storage, payment) requires editing the logic that uses it.
- Import graph: `domain/` imports from `infrastructure/`.

## Refactor

1. Declare the interface next to the high-level module, named for what the policy needs (`EmailSender`), not for the vendor.
2. Make the concrete class implement that interface in the infrastructure layer.
3. Inject the implementation through the constructor (or a function parameter). No framework is required.
4. Wire concrete instances in one composition root (the application entry point or a `main`/bootstrap file).
5. In tests, pass a hand-written fake that implements the interface.

## Example

```ts
declare class SendGridClient {
  constructor(apiKey: string);
  send(to: string, body: string): Promise<void>;
}
type User = { email: string };
declare const apiKey: string;

// Before: the service is welded to a concrete vendor client.
class NotificationServiceBefore {
  private sendgrid = new SendGridClient(apiKey);
  notify(user: User, body: string) { return this.sendgrid.send(user.email, body); }
}

// After: the policy owns the abstraction; the detail is injected.
interface EmailSender {
  send(to: string, body: string): Promise<void>;
}
class NotificationService {
  constructor(private readonly sender: EmailSender) {}
  notify(user: User, body: string) { return this.sender.send(user.email, body); }
}
class SendGridEmailSender implements EmailSender {
  private client = new SendGridClient(apiKey);
  send(to: string, body: string) { return this.client.send(to, body); }
}

// Composition root (main.ts): the only place that names the vendor.
const notifications = new NotificationService(new SendGridEmailSender(apiKey));
```

The business rule is now testable with a three-line fake, and changing vendors edits one class plus one line in the composition root.

## Tradeoffs and when not to apply

- An interface with one implementation and no test that uses a fake is pure ceremony; add it when the seam is needed, not before.
- Do not invert dependencies on stable, ubiquitous primitives (the standard library, `Date`, `Math`); invert on volatile details (vendors, databases, network).
- DI containers add indirection; constructor injection plus a composition root is enough for most codebases.
- Avoid leaking vendor types through the interface (`send(msg: SendGridMessage)`); that keeps the dependency pointing the wrong way.

## Related

- `isp.md`: size the injected interface for the consuming module, not for the vendor's full API.
- `srp.md`: extracting infrastructure classes is usually the step that reveals where DIP applies.
- `ocp.md`: injected abstractions are the extension points that keep policy closed to modification.
