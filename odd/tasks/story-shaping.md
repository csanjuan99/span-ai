# story-shaping

Build a user story inside Span through a guided interview (Example Mapping), keep the product
context the interview needs, seal the story only when nothing is left open, and create it in a
safe place in Jira. Branch `feat/story-shaping`. Delivery strategy: `ask-on-risk`. Test runner:
`go test ./...` and, for `web/`, `npm test` (vitest).

## Specs

S1. Shaping session. "Entonces me surgio la idea de directamente desde Span poder construir estas
historias, pero mediante un proceso de afinacion y curacion muy detallado, es decir extender y
mejorar dicha skill en en un flujo parecio a impeccable donde impeccable me acompaña haciendo
preguntas para construir archivos .md de producto y contexto que permiten luego tomar decisiones,
en este caso me lo imagino similar para lograr la mayor certeza a la hora de diseñar una historia
de usuario."
A person starts a session from an idea and a project folder (optionally one repo inside it). Each
turn the agent reads the idea, the answers so far, the product context and the repository, and
returns the current example map plus at most 3 new questions. The person answers, or accepts a
question's proposed assumption. Sessions persist under `<workspace>/.span/shapes/<id>/`.

S2. Better skill. "Es que tu talvez investigando en internet patrones o estandares pueda hacer que
cuando se use la skill el resultado sea aun mejor por eso te comente el proceso de hacer preguntas
y demas finalmente la skill lo que produce es un resultado."
`harness/skills/user-story/` is the Span copy of the user-story skill (v2). It keeps the template
and its headings verbatim, and adds: an interview mode driven by Example Mapping (story, rules,
examples, questions; the loop "give me an example / what happens if / is that always true / is
that the same rule"), Three Amigos lenses (business, development, testing), and evidence from
four sources, each fact labeled: prompt, answer, code (`path`), context. "Never invent" stays.

S3. Product context. "construir archivos .md de producto y contexto que permiten luego tomar
decisiones". The interview reads `<workspace>/span/context/product.md` when it exists (personas,
glossary, known surfaces). The agent may propose additions; a proposal is written to the file only
when the person accepts it. Nothing writes the file on its own.

S4. Certainty gate. A story is sealed only when every question is answered or its assumption was
accepted by the person (accepted assumptions become `Supuestos`, never rules). When the map shows
more than one outcome the person could receive separately, the turn proposes a split instead of a
story. Sealing stores the story markdown and a sha256 digest; a sealed session does not change.

S5. Safe place in Jira. "Para esto tambien busco que la historia se cree en Jira y se pueda indicar
si va al backlog, a en curso o en to do." Clarified by: "Mi proyecto es Kanban, pero a lo que me
refiero es que solo quiero que se cree en un lugar seguro; ahi si lo dejo a tu criterio".
Decision: the only place Span creates a story is the backlog of the project's Kanban board.
- The board is the single Kanban board of the project key, or the board id the person gives. More
  than one Kanban board and no id: refuse with the list of boards.
- The backlog is enabled only when the board configuration's column named "Backlog" has statuses.
  An empty "Backlog" column is a placeholder: refuse with code `jira_backlog_disabled` and create
  nothing.
- Create unassigned, issue type Story, label `span`, description as ADF.
- Read the created issue back. If its status is not a backlog status, transition it to one; if no
  transition reaches one, report `jira_not_in_backlog` with the key (the issue exists and is
  reported, never hidden).
- The key is stored in the session; asking again for the same session returns that key and
  creates nothing.
- Span never sets "In Progress" or "To Do" at creation. Moving the story out of the backlog is the
  person's or the run's job.

S6. Confirmation. Creating in Jira is visible to others: the page shows a preview (project, board,
backlog statuses, summary, description) and the create call requires `confirm: true`.

S7. Hand-off to a run. A sealed story starts a run through the existing `POST /quickstart` with
`story` = the sealed markdown and `issue` = the Jira key when there is one.

S8. Read-only interview. The interview agent must not change the repository. Span records
`git status --porcelain` of the repo before and after each turn; a difference fails the turn with
code `shape_repo_changed` naming the changed paths. Span does not revert them.

S9. HTTP contract (all JSON, errors as the existing `writeErr` shape):
- `POST /shapes` `{workspace, repo?, idea, agent?, model?}` → 201 Session (runs the first turn).
- `GET /shapes?workspace=` → `[Session]`; `GET /shapes/{id}?workspace=` → Session.
- `POST /shapes/{id}/answers` `{workspace, answers:[{question, text}], assume:[question]}` →
  Session (runs the next turn).
- `POST /shapes/{id}/context` `{workspace, accept:[proposal]}` → Session.
- `POST /shapes/{id}/seal` `{workspace}` → Session; 409 `shape_not_ready` listing open questions.
- `GET /shapes/{id}/jira/preview?workspace=&project=&board=` → Preview.
- `POST /shapes/{id}/jira` `{workspace, project, board?, confirm}` → Session with `jira`;
  400 `confirm_required`; 409 `jira_backlog_disabled`; 409 `shape_not_sealed`.

Session:
```json
{
  "id": "...", "workspace": "...", "repo": "...", "idea": "...", "agent": "...", "model": "...",
  "status": "interviewing | ready | split | sealed",
  "map": {
    "rules": [{"id": "R1", "text": "...", "source": {"kind": "prompt|answer|code|context", "ref": "..."}}],
    "examples": [{"id": "E1", "rule": "R1", "text": "...", "source": {}}],
    "questions": [{"id": "Q1", "text": "...", "blocks": "...", "assume": "...",
                   "status": "open | answered | assumed", "answer": "..."}]
  },
  "split": [{"story": "...", "leftOut": "..."}],
  "proposals": [{"id": "P1", "section": "...", "text": "...", "status": "proposed | accepted"}],
  "turns": [{"n": 1, "at": "RFC3339", "asked": ["Q1"]}],
  "story": "markdown", "digest": "sha256 hex",
  "jira": {"key": "...", "url": "...", "status": "..."}
}
```
Preview: `{"project", "board": {"id", "name"}, "backlogStatuses": [...], "summary", "description"}`.

Non-goals (named in the conversation, not part of this feature): voice conversation, the
multi-agent daily meeting, moving issues between Jira columns.

## Tasks

| ID | Spec | Route | Status | Commit |
| --- | --- | --- | --- | --- |
| T1 | S2, S3 | delegated (parallel, Sonnet) | done | f92e053 |
| T2 | S1, S3, S4, S8 | delegated (parallel, Sonnet) | done | 511787e |
| T3 | S5 | delegated (parallel, Sonnet) | done | db44787, 27b7584 |
| T4 | S1, S5, S6, S7, S9 | delegated (parallel with T5, Sonnet) | done | 4eba00b, 8b0ea8e |
| T5 | S1, S4, S6, S7, S9 | delegated (parallel with T4, Sonnet) | done | 840ec41 |
| T6 | all | inline | done | ef433d5 |

- T1 `harness/skills/user-story/`: skill v2 (interview mode, sources, product context).
- T2 `internal/shape/`: session store, example map merge, gate, seal, turn engine with an
  injectable agent runner, read-only guard, product context accept.
- T3 `internal/jira/`: board discovery, backlog statuses, create, read back, transition into the
  backlog, error codes.
- T4 `internal/serve/` and `cmd/span/`: the S9 endpoints wired to T2 and T3.
- T5 `web/`: shaping page (start, answer, assume, accept context, seal, Jira preview and confirm,
  start run).
- T6 docs: README, USAGE, CHANGELOG, harness/README; seam check `go test ./...` and web tests.

Route evidence: the user asked explicitly for Sonnet agents. T1, T2 and T3 are independent units
with disjoint edit surfaces, launched together. T4 and T5 share only the S9 contract written here,
so they run in parallel after T2 and T3.

## Log

L1. User request, verbatim (idea 1 of 3): "El proceso mediante el cual se construye una historia
es que yo corro una Skill user-story dentro del proyecto que modificare y luego esa historia se la
paso a Span. Entonces me surgio la idea de directamente desde Span poder construir estas historias,
pero mediante un proceso de afinacion y curacion muy detallado, es decir extender y mejorar dicha
skill en en un flujo parecio a impeccable donde impeccable me acompaña haciendo preguntas para
construir archivos .md de producto y contexto que permiten luego tomar decisiones, en este caso me
lo imagino similar para lograr la mayor certeza a la hora de diseñar una historia de usuario."

L2. 2026-11-24, user: "Para responderte no es que algo falle. Es que tu talvez investigando en
internet patrones o estandares pueda hacer que cuando se use la skill el resultado sea aun mejor
por eso te comente el proceso de hacer preguntas y demas finalmente la skill lo que produce es un
resultado. Para esto tambien busco que la historia se cree en Jira y se pueda indicar si va al
backlog, a en curso o en to do. Eso es lo que quiero explorar."

L3. 2026-11-24, user: "Mi proyecto es Kanban, pero a lo que me refiero es que solo quiero que se
cree en un lugar seguro; ahi si lo dejo a tu criterio"

L4. 2026-11-24, user: "Si, armá el documento de feature ODD y luego actua como orquestador y
ejecuta agentes con Sonnet"

L5. Research. The current skill already rests on 3C, INVEST, the Business Rules Manifesto, ISTQB
CT-AT and Specification by Example; the gap is the conversation, not the format. Example Mapping
(Matt Wynne) maps one to one: rule = BR, example = TC, question = open item; many open questions
means not ready, many rules means split. Jira: create cannot be trusted to land in a given status,
so create, read back, then transition. On a company-managed Kanban board the configuration always
returns a "Backlog" column; it has statuses only when the backlog is enabled
(community.developer.atlassian.com/t/61416). This is community evidence, to be confirmed against
the live instance.

L6. Decision: safe place = Kanban backlog (S5). "In Progress" is set by a run, never at creation,
so Jira does not show work nobody is doing.

L7. The pre-commit review hook (Gentleman Guardian Angel, rules in AGENTS.md) reviews every
staged file and also flagged pre-existing code in touched files. Each unit took one to five
rounds; fixes stayed inside each unit's surface. Lesson: stage only the unit's own paths, and a
failed commit leaves files staged that the next commit's review then includes.

L8. Parent fixes, each with RED then GREEN: `CreateInBacklog` returns the created issue along
with `jira_not_in_backlog`, so the key is never lost (27b7584); `jira.ProjectKey` normalizes the
key once; a Jira reply with no key and no error answers 400 instead of panicking; a relative
shaping workspace resolves against the server folder like `/quickstart`, because the page
starts with "." and every shaping call failed with 400 (8b0ea8e).

L9. Seam check on 8b0ea8e/ef433d5: `go vet ./...` clean; `go test ./...` all packages pass except
`internal/worktree` TestManyRunsCanCreateTheirTreesAtTheSameTime once ("failed to read
.git/worktrees/.../commondir"), which then passed 5 of 5; that package has no change since
3d7d1f1, so it is a pre-existing flake (follow-up). Web: typecheck, lint, 263 tests, build pass.

L10. Risk: item 2 (the S8 read-only guard around the interview agent). Independent verifier
launched over the whole feature from baseline 3d7d1f1. Not yet confirmed against a live Jira:
the Kanban backlog column shape (L5).

L11. Verifier: S2-S7, S9 PASS; two blockers. B1, the guard missed commits, checkouts, resets,
branches and tags on a clean tree; B2, the session repo could point outside the workspace. One
correction (9be388e): the guard also compares HEAD, the symbolic branch and every ref, and a
repo outside the workspace (links followed) is refused with `shape_repo_outside` before any agent
run. Recheck limited to B1 and B2: both FIXED. Follow-ups, not blockers: gitignored files and
files outside the repo are not compared (documented in USAGE); another session saving under a
non-ignored `.span/` during a turn can read as a change; a board id from another project is
accepted; if Jira creates the issue and saving the key fails, the error carries no key; locks are
per process. Pre-existing flake: internal/worktree TestManyRunsCanCreateTheirTreesAtTheSameTime.

L12. RDD: assess over 3d7d1f1..HEAD with item 2 escalated gave high / review_due. The user
granted the review (lenses risk, resilience, reliability), but START stopped with
`lens_context_budget_exceeded` (47 files, 6104 lines): no authority was created. The review
needs smaller candidates. Plan: at publish time, slice the branch into chained PRs (strategy
`ask-on-risk` asks for the chain strategy) with the shape package and its guard fix as their
own slice, and review each slice. Per-commit assessment was not run after each work unit; it
should have been, and that is why the open slice became the whole branch.

L13. 2026-11-24, user: "en cadena sobre una rama feature en la que estamos trabajando". Chain
strategy `feature-branch-chain`, one slicing pass by unit, rebuilt locally by cherry-pick (final
tree identical to the old feat/story-shaping head): 01-skill 360, 02-jira 751, 03-shape 1702,
04-serve 990, 05-web 2077, 06-docs 232 changed lines. Slices 02-05 exceed 400: each is one
behavior with its tests (tests are 40-50% of each), so they are `size:exception` candidates.
Assessed per slice against its parent: 01, 02, 04, 05 medium (under_budget, no review due), 06
passive, 03 high (item 2 escalated). 03 reviewed with consent (risk, resilience, reliability):
approved and acknowledged, lineage review-0b2a943542aa7234. Nine non-blocking findings as
follow-ups: proposal section/text not sanitised before it is written into product.md; Answer is
not retry-safe after a failed turn; AcceptProposals appends before saving, so a retry after a
failed save appends twice; status can read ready with a stale story when the next turn fails;
no lock in internal/shape (the HTTP layer serializes per session, the package does not);
gitignored and outside-repo paths are not guarded. Push, the tracker PR and the child PRs are
still the user's decision.
