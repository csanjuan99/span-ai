# Span page

A calm workbench with a terminal inside. Product surfaces (plans, questions, verdicts, forms, navigation) are soft and human, set in Geist. Code surfaces (commands, diffs, logs, transcripts) are precise and machine, set in a monospace face. They are one family: the same hues, radii, hairlines and status colors, so a code block reads as a deeper layer of the page, never a foreign object.

The system is defined in the Span AI design system and wired in `src/index.css`. Light is the first theme and lives on `:root`; Dark mirrors it on `.dark`. The theme follows the system until the person presses `d` or picks one.

## Color

Tokens are named in `src/index.css`; shadcn variables map onto them.

| Token | Light | Dark | Role |
| --- | --- | --- | --- |
| surface-canvas (`background`) | `#f6f6fb` | `#12121a` | The page. |
| surface-raised (`card`, `popover`) | `#ffffff` | `#1a1a26` | Panels, cards, inputs. |
| surface-sunken (`muted`, `secondary`) | `#eceef6` | `#0e0e15` | Tab bars, quiet wells. |
| surface-code | `#eeedf9` | `#0b0b12` | Ground of every code surface. |
| line (`border`) | `#e1e1ee` | `#2a2a3b` | Hairlines. Decorative only. |
| line-strong (`input`) | `#82829f` | `#6a6a88` | Edge of a control, so it keeps 3:1. |
| ink (`foreground`) | `#1a1a2e` | `#ececf6` | Body text. |
| ink-muted (`muted-foreground`) | `#575774` | `#a4a4be` | Secondary text. |
| accent (`primary`) | `#5b3fd4` | `#b8a0f8` | The next action and the current stage. |
| pass / hold / fail | `#16714a` / `#8f5200` / `#bf2b47` | `#7ddba6` / `#f0c76e` / `#f58ea0` | Advance, wait, stop. |

One accent. Pass, hold and fail are verdicts, each with a `-soft` fill. A status color never stands alone: pair it with a word, a mark (`+`, `>`, `·`, A/D/R/M) or a number (`exit 1`).

## Type

Geist for everything a person reads. Machine text (paths, branch names, stage ids, exit codes, commands, diffs) is Geist Mono, with the host monospace stack (`ui-monospace`, SF Mono, Menlo, Consolas) as fallback. Both ship with the app through `@fontsource-variable`, so nothing loads from a CDN. Ligatures are off in code.

## Shape

Radius scale from `--radius: 0.625rem`: 6 / 10 / 14 / 20 px. Controls and code blocks use 10, panels use 14, badges and stage markers are fully round. No square product controls; the terminal feel is carried by code surfaces and the `>`, `+`, `·` stage markers.

## Chrome

The shell still names the path and keeps a timeline band and a status line (`desk`, or stage, verdict and socket state on a run). Restyling the shell and components onto these tokens is tracked separately from the token foundation.
