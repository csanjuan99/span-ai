# Functions

Load this when a function does more than one thing, mixes abstraction levels, takes many or boolean arguments, hides side effects, or nests deeply.

**Small, and doing one thing.** A function should do one thing, do it well, and do only it. The practical test: can you describe what it does in one sentence without "and"? Smaller functions are easier to name, test, and reuse. "Small" is a guide, not a line-count law — a flat 40-line switch that does one thing can be clearer than splitting it five ways.

**One level of abstraction per function.** Don't mix high-level policy ("charge the customer") with low-level mechanics (string formatting, array indexing) in the same body. Mixing levels forces the reader to constantly shift gears. Extract the lower level into a named helper.

**Few arguments.** Zero to two is ideal; three is a smell worth a second look; more than three usually wants a parameter object. Each argument is something the caller must understand and supply correctly, and long positional lists invite mistakes (`createUser("Ana", "Lopez", true, false, null)` — what are those booleans?).

```ts
// Before: positional soup
function createUser(first, last, isAdmin, sendEmail, referrer) { /* ... */ }
createUser("Ana", "Lopez", true, false, null);

// After: a named options object — call sites self-document
interface CreateUserInput {
  firstName: string;
  lastName: string;
  isAdmin?: boolean;
  sendWelcomeEmail?: boolean;
  referrer?: string;
}
function createUser(input: CreateUserInput) { /* ... */ }
createUser({ firstName: "Ana", lastName: "Lopez", isAdmin: true });
```

**No flag arguments.** A boolean parameter that switches behavior means the function does two things; split it. The call site `render(true)` tells the reader nothing.

```ts
// Before
function render(isPreview: boolean) {
  if (isPreview) { /* preview path */ } else { /* full path */ }
}
render(true);

// After
function renderPreview() { /* preview path */ }
function renderFull() { /* full path */ }
renderPreview();
```

**No hidden side effects.** A function named `getUser` should not also create a session. Surprises that aren't in the name are where bugs hide. If a function must mutate or perform I/O, let the name say so (`loadAndCacheUser`).

**Command-query separation.** A function either *does* something (command, returns void/throws) or *answers* something (query, returns a value), not both. `if (set("username", "ana"))` is a riddle — does `set` test existence or perform assignment? Keep them separate so callers never have to guess.

**Guard clauses over deep nesting.** Handle the edge cases up front and return early; let the happy path flow down the left margin, un-indented. Deep nesting forces the reader to hold many conditions in their head at once.

```ts
// Before: the real work is buried four levels deep
function getPayrate(employee) {
  if (employee) {
    if (employee.isActive) {
      if (employee.contract) {
        return employee.contract.rate;
      } else {
        throw new Error("No contract");
      }
    } else {
      throw new Error("Inactive");
    }
  } else {
    throw new Error("No employee");
  }
}

// After: guards first, happy path last and flat
function getPayrate(employee) {
  if (!employee) throw new Error("No employee");
  if (!employee.isActive) throw new Error("Inactive");
  if (!employee.contract) throw new Error("No contract");
  return employee.contract.rate;
}
```
