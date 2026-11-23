---
name: bounded-contexts
description: "Trigger: reorganizar carpetas, nueva feature, estructura de proyecto, folder structure, where does this file go, bounded context, single file context, dónde va esta constante/tipo/util, file naming, suffix convention. Frontend-only: decides where code lives in modules/ vs shared/ by reuse scope, and which file each symbol belongs in. Not for backends or APIs."
license: "Proprietary"
metadata:
  author: "Carlos San Juan"
  version: "2.1.0"
  category: "architecture"
---

## Activation Contract

Load this skill when creating a new folder structure, placing a new file, adding a feature or page, deciding which file a constant, type, util, schema, or style belongs in, reorganizing legacy code, or migrating toward clean architecture in a frontend project (React, React Native/Expo, Next.js, Remix, Vite). Do not load for backend services, APIs, or infrastructure code.

This skill answers two questions in order: **Bounded Contexts** decides which folder code lives in; **Single File Context** decides which file it lives in once that folder is chosen.

## Hard Rules

- Frontend only: apply to UI codebases; backend and infrastructure repos are out of scope.
- Root layout is exactly `src/modules/` (domain features) and `src/shared/` (cross-module resources). Framework entry folders such as Expo `app/` stay at root and are not moved.
- Placement is decided by reuse scope, never by file type alone: ask "is this used by more than one module?" before choosing a folder.
- Every page is a folder with an `index.{tsx,vue,...}` entry: `modules/<module>/pages/<page>/index.tsx`.
- Components under `shared/components/` live in a folder per component; never leave loose component files at the `components/` root.
- Single File Context: a file holds exactly one kind of context, declared by its suffix (`<context>.constants.ts`, `<context>.types.ts`, `<context>.utils.ts`, ...). A symbol that is not part of the file's own kind is extracted to its own file.
- `index.{ts,tsx}` is either a component/page entry or a pure barrel of `export` lines. It never declares constants, types, interfaces, enums, schemas, or helper functions.
- Never create a generic bucket file (`constants/index.ts`, `types.ts`, `utils.ts`) that groups unrelated symbols by kind alone; name every file after the context it serves.
- Document with JSDoc above symbols; never inline comments.
- Services and adapters are object singletons (`export const authService = { ... }`). Constants use `export const`; types use `export type` or `export interface`.
- Create only the `shared/` subfolders the project actually needs.

## Decision Gates

| Question | Answer | Location |
| --- | --- | --- |
| Used in more than one module? | Yes | `shared/<type>/` |
| Used in more than one module? | No | `modules/<module>/` and continue |
| Used in more than one page of that module? | Yes | `modules/<module>/shared/<type>/` |
| Used in more than one page of that module? | No | `modules/<module>/pages/<page>/` |

`<type>` folders for `shared/` (root or per module):

| Content | Folder |
| --- | --- |
| React contexts | `contexts/` |
| Utility functions | `utils/` |
| Adapters (DB, storage) | `adapters/` |
| Services (API, auth) | `services/` |
| i18n | `i18n/` |
| Global constants | `constants/` |
| Global styles | `styles/` |
| Images, fonts, icons | `assets/` |
| Generic UI components | `components/<component>/` |
| Reusable hooks | `hooks/` |
| Data models | `models/` |
| Global types | `types/` |
| Error classes | `errors/` |

Naming:

| Kind | Format | Example |
| --- | --- | --- |
| Code files | kebab-case + suffix | `location.service.ts`, `theme.constants.ts` |
| UI components | kebab-case folder, PascalCase file | `profile-navbar/ProfileNavbar.tsx` |
| Contexts | PascalCase | `AuthContext.tsx` |
| Hooks | `use` prefix | `use-auth.ts` |

## Single File Context

Once the Decision Gates give you a folder, this decides the file.

Extraction gate — before writing a symbol inside a component or entry file, ask in order:

1. Is it part of this component's own rendering logic? If no → extract.
2. Would it still make sense with the component deleted? If yes → extract.
3. Could another file plausibly import it? If yes → extract.
4. Is it a literal used once, inside the JSX it belongs to? If yes → it may stay.

Extracted symbols go to `<context>.<kind>.ts` in the same folder, where `<context>` is the module, page, or component name:

| Content | Suffix |
| --- | --- |
| Constants, keys, magic numbers | `.constants.ts` |
| Types and interfaces | `.types.ts` |
| Enums | `.enums.ts` |
| Pure helper functions | `.utils.ts` |
| Services / adapters | `.service.ts`, `.adapter.ts` |
| Validation schemas, validators, type guards | `.schemas.ts`, `.validators.ts`, `.guards.ts` |
| DTOs, mappers, models | `.dto.ts`, `.mappers.ts`, `.model.ts` |
| Queries, mutations, stores, selectors | `.queries.ts`, `.mutations.ts`, `.store.ts`, `.selectors.ts` |
| Errors, events | `.errors.ts`, `.events.ts` |
| Styles, animations, config, routes | `.styles.ts`, `.animations.ts`, `.config.ts`, `.routes.ts` |
| Mocks, fixtures, tests | `.mocks.ts`, `.fixtures.ts`, `.test.ts` |

React contexts (`AuthContext.tsx`), hooks (`use-auth.ts`), components (`profile-navbar/ProfileNavbar.tsx`), and page entries (`index.tsx`) keep the naming from the table above.

Granularity is one context per file, not one symbol per file: `auth.constants.ts` with six auth constants is correct. For the full suffix catalog, worked before/after examples, and the split-further rule, load `references/single-file-context.md`.

## Execution Steps

1. Identify the item to place (page, component, hook, service, type) and list every module and page that uses it.
2. Walk the Decision Gates table top-down to get the target folder; apply the naming table.
3. Apply Single File Context: run the extraction gate over every symbol in the file, and name each resulting file `<context>.<kind>.ts`. Leave `index.{ts,tsx}` as an entry or a barrel only.
4. If the project has an existing structure or framework entry folder, load `references/structure.md` for the full tree, `shared/components/` layout, and Expo/Next.js/Remix/Vite notes.
5. If migrating an existing codebase, load `references/migration-dutties-apollo.md` as the worked example and produce an equivalent current-to-new mapping table before moving files.
6. Update imports after any move; the migration itself is a separate task from placement guidance.

## Output Contract

- Target path for each item, with the gate answer that justified it.
- For each file, the symbols it may contain and any symbol extracted out of it, with its new `<context>.<kind>.ts` file.
- For migrations: a mapping table (current path, new path, note) and ordered steps.
- Any placement that does not fit the gates, flagged for human decision.

## References

- `references/single-file-context.md` — load when you need the full suffix catalog, before/after examples, the `index.ts` rule, or the split-further guidance.
- `references/structure.md` — load when you need full `shared/`, `modules/`, or `shared/components/` tree examples or per-framework layout notes.
- `references/migration-dutties-apollo.md` — load when migrating an existing project; project-specific worked example with mapping table and result tree.
