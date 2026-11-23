# Error Handling

Load this when reviewing try/catch blocks, error codes, null returns, input validation, or error messages.

Error handling matters, but it shouldn't drown the main logic. Separate the happy path from the failure handling so each stays readable.

**Prefer exceptions or a `Result` type over error codes.** Error codes force every caller to check and branch inline, tangling error handling into the logic and making it easy to forget a check.

```ts
// Before: error codes silently ignored, logic and errors tangled
const code = saveOrder(order);
if (code === -1) { /* ... */ }   // easy to forget; easy to mishandle

// After: throw, and handle at the boundary where you can do something
function saveOrder(order: Order): SavedOrder {
  if (!order.items.length) throw new EmptyOrderError(order.id);
  return repository.persist(order);
}
```

**Don't return or pass `null` casually.** A `null` that escapes a function becomes every caller's problem and the source of the next null-reference crash. Prefer returning an empty collection, a typed "not found" result, or throwing. If absence is a real state, model it explicitly (`User | undefined`) so the type system forces callers to handle it.

**Fail fast.** Validate inputs at the boundary and throw immediately with a clear message, rather than letting bad data flow deep into the system where the eventual failure is far from its cause.

**Don't swallow errors.** An empty `catch {}` discards the one piece of information that would let someone diagnose the problem. At minimum log with context; better, handle or rethrow. Swallowing turns a loud failure into a silent, much harder bug.

**Provide context in errors.** "Operation failed" tells no one anything. Include what was being attempted and the relevant identifiers: `Failed to charge order ${order.id}: ${cause.message}`. The error message is often the only clue the on-call engineer gets at 3am.
