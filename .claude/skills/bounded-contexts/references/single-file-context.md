# Single File Context

Load this when deciding what may live inside a file, or when naming a file by the kind of symbol it exports.

## The rule

A file holds exactly one kind of context. The file name declares that kind through its suffix, and the file exports nothing that belongs to another kind.

Bounded Contexts answers **where** code lives (which module, which page, which shared level). Single File Context answers **which file** it lives in once that folder is chosen. They compose: first walk the Decision Gates, then apply this.

## What this forbids

- Declaring a constant, type, interface, enum, schema, or helper function inside `index.ts` / `index.tsx`.
- Declaring them inside a component file when they are not part of that component's rendering logic.
- A `constants/index.ts` (or `types/index.ts`, `utils/index.ts`) that is a bucket of unrelated declarations grouped only by being "constants".
- A single `types.ts` at a folder root that accumulates every type of every feature under it.

## What this requires

Every extracted symbol goes to a file named `<context-name>.<kind>.ts`, where `<context-name>` is the module, feature, page, or component it belongs to — not a generic word.

```
❌ shared/constants/index.ts
   export const TIME_ALIVE = 500;
   export const MAX_RETRIES = 3;
   export const AUTH_TOKEN_KEY = 'token';

✅ modules/session/shared/session.constants.ts
   export const TIME_ALIVE = 500;

✅ modules/auth/shared/auth.constants.ts
   export const AUTH_TOKEN_KEY = 'token';

✅ shared/http/http.constants.ts
   export const MAX_RETRIES = 3;
```

```
❌ modules/profile/pages/settings/index.tsx
   type SettingsForm = { name: string };
   const FIELD_LIMIT = 40;
   const formatHandle = (h: string) => h.trim().toLowerCase();
   export default function Settings() { ... }

✅ modules/profile/pages/settings/
   ├── index.tsx                      // only the component
   ├── settings.types.ts              // SettingsForm
   ├── settings.constants.ts          // FIELD_LIMIT
   └── settings.utils.ts              // formatHandle
```

## Suffix catalog

Use the suffix that names the kind of symbol, in kebab-case, always `<context>.<kind>.ts`.

| Content | Suffix | Example |
| --- | --- | --- |
| Constants, magic numbers, keys | `.constants.ts` | `session.constants.ts` |
| Types and interfaces | `.types.ts` | `user.types.ts` |
| Enums (when kept apart from types) | `.enums.ts` | `order-status.enums.ts` |
| Pure helper functions | `.utils.ts` | `date.utils.ts` |
| API / domain services (singleton object) | `.service.ts` | `auth.service.ts` |
| Adapters (DB, storage, SDK boundaries) | `.adapter.ts` | `secure-storage.adapter.ts` |
| Validation schemas (zod, yup, valibot) | `.schemas.ts` | `sign-up.schemas.ts` |
| Standalone validators / predicates | `.validators.ts` | `password.validators.ts` |
| Type guards | `.guards.ts` | `user.guards.ts` |
| DTO ↔ domain translation | `.mappers.ts` | `invoice.mappers.ts` |
| DTO shapes from the wire | `.dto.ts` | `invoice.dto.ts` |
| Domain models | `.model.ts` | `user.model.ts` |
| Data-fetching queries / mutations | `.queries.ts`, `.mutations.ts` | `orders.queries.ts` |
| Client state store / slice | `.store.ts` | `cart.store.ts` |
| Derived state selectors | `.selectors.ts` | `cart.selectors.ts` |
| Error classes | `.errors.ts` | `http.errors.ts` |
| Event names / payloads | `.events.ts` | `analytics.events.ts` |
| Style objects (`StyleSheet`, styled) | `.styles.ts` | `profile-navbar.styles.ts` |
| Animation definitions | `.animations.ts` | `drawer.animations.ts` |
| Build/runtime configuration | `.config.ts` | `sentry.config.ts` |
| Route definitions | `.routes.ts` | `app.routes.ts` |
| Test doubles, fixtures | `.mocks.ts`, `.fixtures.ts` | `user.mocks.ts` |
| Tests | `.test.ts`, `.spec.ts` | `date.utils.test.ts` |

Exceptions that keep the existing Bounded Contexts naming:

| Kind | Format | Example |
| --- | --- | --- |
| React contexts | PascalCase | `AuthContext.tsx` |
| Hooks | `use-` prefix, no suffix | `use-auth.ts` |
| UI components | PascalCase inside kebab-case folder | `profile-navbar/ProfileNavbar.tsx` |
| Page entry | `index.{tsx,vue}` | `pages/login/index.tsx` |

If a kind is not listed, invent the suffix from the plain English name of the symbol kind, in kebab-case, and keep it consistent across the repo.

## What `index.ts` is allowed to be

Exactly two things:

1. **A component or page entry** — `index.tsx` holding one component and nothing else.
2. **A barrel** — only `export` / `export * from` lines, zero declarations.

An `index.ts` that declares anything is a Single File Context violation.

## Extraction gate

Before writing a symbol inside a component or entry file, ask in order:

1. **Is it part of this component's own rendering logic?** If no → extract.
2. **Would it still make sense with the component deleted?** If yes → extract.
3. **Could another file plausibly import it?** If yes → extract.
4. **Is it a literal used once, inside the JSX it belongs to?** If yes → it may stay.

A `useState` initial value, a locally derived variable, or a one-line handler that only touches component state stays. A timeout, a limit, a regex, a formatter, a shape, or a mapping leaves.

## How far to split

One context per file, not one symbol per file. `auth.constants.ts` holding six auth constants is correct; splitting it into six files is not the rule and creates its own noise.

Split a suffix file further only when it grows two clearly distinct contexts inside the same folder — then name each by its context (`login.constants.ts`, `recovery.constants.ts`), never by numbering or by size.

## Relation to known ideas

There is no established industry term named "Single File Context". It is the Single Responsibility Principle applied at file granularity, combined with the file-suffix conventions popularized by Angular style guides and NestJS. Treat the name as this codebase's vocabulary, and the rule as SRP for files.
