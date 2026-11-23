# Skills

Portable agent skills following the [Agent Skills](https://agentskills.io) open standard. Each skill is a directory whose name equals the `name` in its `SKILL.md` frontmatter. `SKILL.md` is a thin runtime contract; detail lives in `references/`, executables in `scripts/`.

| Skill | Category | Load when |
| --- | --- | --- |
| `bounded-contexts/` | architecture | Frontend only: deciding where a file lives, creating folder structure, migrating to modules/shared |
| `clean-code/` | code-quality | Writing, reviewing, or refactoring for readability (naming, functions, comments, errors, smells) |
| `solid-principles/` | design | Diagnosing and refactoring SRP/OCP/LSP/ISP/DIP violations |
| `software-design-patterns/` | design | Choosing or applying one of the 23 GoF patterns |
| `code-quality/` | code-quality | TypeScript quality rules: guard clauses, single responsibility, never any, JSDoc |
| `cost-sentinel/` | ops | Read-only AWS cost spot-check after a deploy or suspected runaway |
| `change-impact/` | architecture | Before building or changing a behavior: what it creates, modifies, or extends, and which other behaviors can break |
| `api-first/` | architecture | Before building a backend capability a third party can call: write the consumer contract first, in the repo's own language and style |

## Conventions

- Frontmatter: `name`, one-line quoted `description` starting with `Trigger:`, `license`, `metadata` (strings only).
- Body sections in order: Activation Contract, Hard Rules, Decision Gates, Execution Steps, Output Contract, References.
- Body budget: target under 450 words. Examples and background go to `references/`; load only the file the decision gate names.
- Artifacts are written in English; trigger phrases keep their Spanish variants inside `description`.

## Install

Symlink or copy a skill directory into your harness's skills path, for example `~/.claude/skills/<name>`, `.claude/skills/<name>`, or the equivalent for Codex, OpenCode, Cursor, or Gemini CLI. Then refresh the registry if your tooling has one (for example `gentle-ai skill-registry refresh`).
