# Span harness

These skills are the house style. Two people using Span should produce code that reads as one author.

Each skill is a directory under `skills/` with a thin `SKILL.md`. Detail stays in `references/` and is read only when that skill's trigger matches the change. Do not paste every skill into a prompt.

Span resolves this directory next to the `span` binary, from `SPAN_HARNESS`, or by walking up from the working directory. A project can pin the absolute path in `span.yaml` as `harness`.

The build prompt lists each skill's name, its trigger, and the path of `SKILL.md`. The agent opens that file only when the trigger matches the change. Reference files stay unread until the skill names one. Bodies are not pasted into the prompt.
