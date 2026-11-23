# Boundaries and Tests

Load this when writing or reviewing tests, or when wrapping a third-party library or external service.

Test code is real code: it is read, maintained, and trusted, so it deserves the same care as production code. Sloppy tests rot, get disabled, and stop protecting you.

**One concept per test.** Each test verifies one behavior. Multiple unrelated assertions in one test make a failure ambiguous — you can't tell at a glance what broke. (One *concept* may legitimately need a few assertions; the point is one reason to fail.)

**Readable: Arrange–Act–Assert.** Structure each test so setup, the action, and the verification are visually distinct. A test is also documentation of how the unit is meant to behave; make that story easy to read.

**FIRST.** Good unit tests are **F**ast (so people actually run them), **I**ndependent (no ordering or shared-state coupling), **R**epeatable (same result every run, no reliance on clock, network, or random), **S**elf-validating (a clear pass/fail, no manual log-reading), and **T**imely (written alongside the code, while the design is still malleable).

```ts
// Readable, one concept, self-validating
test("applies the loyalty discount to members", () => {
  const cart = new Cart([{ price: 100 }]);          // Arrange
  const total = cart.totalFor({ isMember: true });  // Act
  expect(total).toBe(90);                           // Assert
});
```

At **boundaries** (third-party libraries, external services), wrap the foreign API behind a thin interface you own. It keeps their churn from rippling through your code and gives you a seam to fake in tests.
