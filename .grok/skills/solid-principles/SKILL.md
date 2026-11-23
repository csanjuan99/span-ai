---
name: solid-principles
description: "Trigger: SOLID, SRP, OCP, LSP, ISP, DIP, 'principios SOLID', or symptoms: class edited for unrelated reasons, growing switch, instanceof checks, fat interfaces, untestable logic. Diagnose and apply the minimal SOLID refactor."
license: "Proprietary"
metadata:
  author: "Carlos San Juan"
  version: "1.0.0"
  category: "design"
---

## Activation Contract

Load when the user names a SOLID principle (English or Spanish), asks whether a design follows SOLID, or describes a Decision Gates symptom while writing, reviewing, or refactoring modular code. For naming or style requests, use `clean-code` instead.

## Hard Rules

- Treat each principle as a lens for a concrete pain, not a checkbox; apply it only when the symptom is present in the code under review.
- Name the cost the refactor prevents (collisions, regressions, untestable logic), never the principle's authority.
- Prefer the smallest refactor: extract a class or module, a handler map, a role interface, or constructor injection.
- Never introduce an abstraction with a single implementation unless a test seam or second implementation is imminent and stated.
- A small script or prototype does not need five interfaces; match rigor to lifespan and blast radius.
- Load only the reference file for the diagnosed principle.

## Decision Gates

| Symptom | Principle | Refactor | Load |
| --- | --- | --- | --- |
| Module changes for unrelated reasons; mixes rules with persistence or I/O | SRP | Extract class or module per reason to change | `references/srp.md` |
| Same `switch` or `if` chain edited for every new case | OCP | Polymorphism or handler/strategy map | `references/ocp.md` |
| Subtype needs `instanceof`, throws "not supported", or weakens the base contract | LSP | Fix hierarchy; prefer composition or separate types | `references/lsp.md` |
| Implementers stub methods; clients depend on unused members | ISP | Split into role interfaces | `references/isp.md` |
| Logic instantiates concrete infrastructure; untestable without real DB or network | DIP | Depend on an interface; inject the implementation | `references/dip.md` |

| Do not apply when |
| --- |
| Throwaway script, spike, or one-off migration |
| One stable implementation and no test seam needed |
| Team convention conflicts: raise it as a team decision, never drive-by |

## Execution Steps

1. Read the code or diff; list observable symptoms, not principle names.
2. Match each symptom to one Decision Gates row and load only that reference file.
3. Confirm the cost is real: name the change or test that is painful today. If none, report "No SOLID issue".
4. Propose the minimal refactor with a before/after sketch from the reference's pattern.
5. Note interplay with other principles only when the refactor triggers a second symptom (for example, DIP extraction creating a fat interface).
6. Report per the Output Contract; cite `references/sources.md` only when asked for provenance.

## Output Contract

Per finding, return:
- Principle and file or symbol location.
- Symptom observed (one sentence).
- Cost prevented (one sentence).
- Minimal refactor: before/after sketch, at most 15 lines.
- Risks or tradeoffs.

If no symptom holds, return "No SOLID issue" with the reason.

## References

- `references/srp.md` — load when a module has multiple reasons to change.
- `references/ocp.md` — load when adding a case means editing existing conditionals.
- `references/lsp.md` — load when subtypes break base-type expectations.
- `references/isp.md` — load when interfaces force unused members on clients.
- `references/dip.md` — load when high-level logic depends on concrete infrastructure.
- `references/sources.md` — load when citing primary sources for a principle.
