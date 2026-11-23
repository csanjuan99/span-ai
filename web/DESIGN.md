# Span page

A terminal session, not a dashboard. The structure comes from gentle-shell: a prompt, hairline rules, and a status line. The colors are herdr's default Catppuccin Mocha.

## Scene

Night, one pane. The screen is the light. The person running Span is already in a terminal.

## Color

| Token | Hex | Role |
| --- | --- | --- |
| background | `#11111b` | Crust. The pane. |
| card | `#1e1e2e` | A nested record, the patch, the transcript. |
| foreground | `#cdd6f4` | Body text. |
| muted | `#a6adc8` | Secondary text. |
| primary | `#cba6f7` | The live prompt and the current stage. |
| pass | `#a6e3a1` | `advance` and `pass`. |
| hold | `#f9e2af` | `hold` and `unverified`. |
| destructive | `#f38ba8` | `stop` and `fail`. |
| border | `#313244` | Rules and control outlines. |

One accent carries the prompt. Green, yellow, and red are verdicts, the same jobs they have in a shell.

## Type

The host monospace: `ui-monospace`, SF Mono, Menlo, Consolas. Ligatures are off. Headings stay the same face. Hierarchy is the `$` prompt, weight, and color.

## Controls

Square. A primary command is an outline in mauve. A secondary command is an outline in the rule color. Inputs are flat fields on the crust. Tabs are words on a rule. The current one is mauve.

## Chrome

A title bar names the path. The pane scrolls. A timeline band sits above the status line and records decisions, not the agent transcript. A status line stays at the bottom: on the desk it says `desk`; on a run it says the stage, the verdict, and whether the socket is open.
