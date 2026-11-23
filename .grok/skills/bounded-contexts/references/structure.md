# Bounded Context Structure

Load this when you need the full folder trees for `shared/` and `modules/`, the `shared/components/` rule, or framework-specific layout notes.

## Base layout

```
src/
├── modules/      # domain modules (features)
└── shared/       # resources shared across modules
```

## shared/ — shared resources

Folders are grouped by element type. Not all exist in every project; create only what the project needs.

```
shared/
├── contexts/          # React contexts (PascalCase)
├── utils/             # utility functions
├── adapters/          # adapters (DB, storage, etc.)
├── services/          # services (API, auth, etc.)
├── i18n/              # internationalization
├── constants/         # global constants
├── styles/            # global styles
├── assets/            # images, fonts, icons
├── components/        # generic components (PascalCase)
│   └── [component]/
│       └── [ComponentName].tsx
├── hooks/             # reusable hooks
├── models/            # data models
├── types/             # global types
└── errors/            # error classes
```

### shared/components/ — special structure

Components inside `shared/components/` always live in a folder named after the component:

```
shared/components/
├── ui/
│   └── collapsible/
│       └── Collapsible.tsx
├── themed-view/
│   └── ThemedView.tsx
├── themed-text/
│   └── ThemedText.tsx
└── profile-navbar/
    └── ProfileNavbar.tsx
```

Never leave loose component files at the root of `components/`. Component files are always PascalCase.



## modules/ — domain modules

Each module represents a business domain. It contains pages and resources shared between the pages of that module.

```
modules/[domain]/
├── pages/
│   └── [page]/
│       └── index.tsx
└── shared/
    ├── services/
    ├── contexts/
    ├── hooks/
    ├── components/
    └── ...
```

### pages/

Each page is a folder with its `index.{tsx,vue,etc}`:

```
modules/auth/pages/
├── login/
│   └── index.tsx
└── sign-up/
    └── index.tsx
```

### shared/ (per module)

Resources shared between pages of the same module:

```
modules/auth/shared/
├── services/
│   └── auth.service.ts
├── contexts/
│   └── AuthContext.tsx
└── hooks/
    └── use-auth.ts
```

## Naming conventions

| Kind | Format | Example |
| --- | --- | --- |
| Code files | `kebab-case` + suffix | `location.service.ts`, `theme.constants.ts` |
| UI components | kebab-case folder, `PascalCase` file | `profile-navbar/ProfileNavbar.tsx`, `themed-view/ThemedView.tsx` |
| Contexts | `PascalCase` | `AuthContext.tsx`, `ThemeContext.tsx` |
| Hooks | `kebab-case` file, `use` prefix | `use-theme.ts`, `use-auth.ts` |

Rules:

- JSDoc always; never inline comments.
- Services and adapters use a singleton object:
  ```ts
  export const authService = {
    login: async () => { ... },
  };
  ```
- Constants: `export const`.
- Types and interfaces: `export type` or `export interface`.

## Per framework

### Expo / React Native

Keep `app/` at the root (Expo Router entry points):

```
src/
├── app/              # Expo Router (do not touch)
├── shared/
│   ├── adapters/
│   ├── services/
│   └── components/
└── modules/
    └── [domain]/
```

### React Web (Next.js, Remix, Vite)

`shared/` may include `routes/`:

```
shared/
├── routes/
│   ├── index.ts
│   └── paths.ts
└── ...
```

The entry point imports from modules:

```ts
// shared/routes/index.ts
export const routes = {
  '/': 'modules/home/pages/index.tsx',
  '/tasks': 'modules/tasks/pages/list/index.tsx',
};
```
