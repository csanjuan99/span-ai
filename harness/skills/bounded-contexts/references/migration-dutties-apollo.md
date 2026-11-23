# Migration Guide: dutties-apollo

Load this when migrating an existing project to the bounded-context layout; it is a project-specific worked example (Expo / React Native) showing the current tree, the mapping table, the resulting tree, and the steps.

## Current structure

```
src/
├── app/           # Expo Router
├── components/    # components
├── constants/     # theme.ts
├── core/          # auth, db, http
├── features/      # account, tasks, sync
├── hooks/         # use-theme, use-color-scheme
└── types/         # global types
```

## Migration mapping

| Current | New | Notes |
| --- | --- | --- |
| `src/app/` | `src/app/` | Keep as is |
| `src/components/` | `src/shared/components/` | Move all |
| `src/hooks/` | `src/shared/hooks/` | Move all |
| `src/constants/theme.ts` | `src/shared/constants/theme.constants.ts` | Rename |
| `src/types/` | `src/shared/types/` | Move all |
| `src/core/auth/` | `src/shared/services/auth.service.ts` | Consolidate into a singleton |
| `src/core/db/` | `src/shared/adapters/` | Move adapters |
| `src/core/http/` | `src/shared/services/http.service.ts` | Consolidate into a singleton |
| `src/features/account/` | `src/modules/account/` | Convert to module |
| `src/features/tasks/` | `src/modules/tasks/` | Convert to module |
| `src/features/sync/` | `src/modules/sync/` | Convert to module |

## Resulting structure

```
src/
├── app/
│   ├── _layout.tsx
│   ├── index.tsx
│   └── explore.tsx
├── shared/
│   ├── adapters/
│   │   ├── db.adapter.ts
│   │   └── token-store.adapter.ts
│   ├── services/
│   │   ├── auth.service.ts
│   │   └── http.service.ts
│   ├── constants/
│   │   └── theme.constants.ts
│   ├── components/
│   │   ├── ui/
│   │   │   └── collapsible/
│   │   │       └── Collapsible.tsx
│   │   ├── animated-icon/
│   │   │   └── AnimatedIcon.tsx
│   │   ├── themed-view/
│   │   │   └── ThemedView.tsx
│   │   ├── themed-text/
│   │   │   └── ThemedText.tsx
│   │   ├── external-link/
│   │   │   └── ExternalLink.tsx
│   │   ├── app-tabs/
│   │   │   └── AppTabs.tsx
│   │   ├── web-badge/
│   │   │   └── WebBadge.tsx
│   │   └── hint-row/
│   │       └── HintRow.tsx
│   ├── hooks/
│   │   ├── use-theme.ts
│   │   └── use-color-scheme.ts
│   └── types/
│       └── css.d.ts
├── modules/
│   ├── account/
│   │   ├── pages/
│   │   │   └── profile/
│   │   │       └── index.tsx
│   │   └── shared/
│   │       ├── presentation/
│   │       │   ├── account-banner/
│   │       │   │   └── AccountBanner.tsx
│   │       │   ├── account-sheet/
│   │       │   │   └── AccountSheet.tsx
│   │       │   └── AuthContext.tsx
│   │       └── domain/
│   │           └── session.ts
│   ├── tasks/
│   │   ├── pages/
│   │   │   ├── today/
│   │   │   │   └── index.tsx
│   │   │   └── task-form/
│   │   │       └── index.tsx
│   │   └── shared/
│   │       ├── presentation/
│   │       │   └── use-today-tasks.ts
│   │       ├── domain/
│   │       │   └── task.ts
│   │       └── data/
│   │           └── task-repository.ts
│   └── sync/
│       └── shared/
│           └── sync-engine.ts
```

> Review needed: the `presentation/`, `domain/`, and `data/` subfolders under `modules/<module>/shared/` in the tree above are project-specific and are not in the `shared/<type>` folder list.

## Suggested migration steps

1. **Create structure** — create the `shared/` and `modules/` folders.
2. **Move shared** — constants, types, hooks, generic components.
3. **Move core** — consolidate into services and adapters.
4. **Convert features** — create modules with `pages/` and `shared/`.
5. **Update imports** — fix paths across the whole project.

> Note: the migration is a process independent of the skill. The skill only guides the structure.

## Example: creating a new page in dutties-apollo

1. **Used in more than one module?**
   - No → goes in `modules/[domain]/pages/[page]/`.

2. **Used in more than one page of the same module?**
   - Yes → goes in `modules/[domain]/shared/`.
   - No → goes directly in `modules/[domain]/pages/[page]/`.

3. **Example**: new "settings" page in `account`:

   ```
   modules/account/pages/settings/index.tsx
   ```

   If it needs a hook shared with other `account` pages:

   ```
   modules/account/shared/hooks/use-account-settings.ts
   ```
