import type { ShapePreview, ShapeSession, ShapeSource } from "@/lib/types"

/** Where the person is in a session. `jira` is the preview of the issue, opened from a sealed story. */
export type SessionStage =
  | "interview"
  | "questions"
  | "map"
  | "split"
  | "context"
  | "seal"
  | "sealed"
  | "jira"

/** What the person already did on this page. The server only knows the session's status. */
export interface SessionProgress {
  /** The example map was read and confirmed. */
  reviewed?: boolean
  /** The product-context notes were settled. */
  contextDone?: boolean
  /** The Jira preview is open. */
  jira?: boolean
}

/** The settled and open counts of the questions. */
export function questionCounts(session: ShapeSession) {
  const count = (status: string) =>
    session.questions.filter((question) => question.status === status).length
  return {
    open: count("open"),
    answered: count("answered"),
    assumed: count("assumed"),
  }
}

/** How the product-context notes stand. */
export function contextCounts(session: ShapeSession) {
  const count = (status: string) =>
    session.proposals.filter((proposal) => proposal.status === status).length
  return {
    accepted: count("accepted"),
    proposed: count("proposed"),
    skipped: count("skipped"),
  }
}

/** The screen a session shows now: the server's status, then how far the person has got. */
export function sessionStage(
  session: ShapeSession,
  progress: SessionProgress
): SessionStage {
  if (session.status === "sealed") {
    return progress.jira === true ? "jira" : "sealed"
  }
  if (session.status === "interviewing") {
    return questionCounts(session).open > 0 ? "questions" : "interview"
  }
  if (progress.reviewed !== true) {
    return "map"
  }
  if (session.status === "split") {
    return "split"
  }
  if (contextCounts(session).proposed > 0 && progress.contextDone !== true) {
    return "context"
  }
  return "seal"
}

/** One step of the rail. */
export interface SessionStep {
  name: string
  place: "done" | "current" | "waiting"
  /** The words under the name: "done", "you are here", "later" or "optional". */
  note: string
}

const stepNames = [
  "Idea",
  "Interview",
  "Questions",
  "Examples",
  "Split",
  "Context",
  "Seal",
  "Jira",
] as const

const optionalSteps = new Set<string>(["Split", "Context", "Jira"])

const stageIndex: Record<SessionStage, number> = {
  interview: 1,
  questions: 2,
  map: 3,
  split: 4,
  context: 5,
  seal: 6,
  sealed: 7,
  jira: 7,
}

/** The eight steps of a session, with where the person is. */
export function sessionSteps(
  session: ShapeSession,
  stage: SessionStage
): SessionStep[] {
  const at = stageIndex[stage]
  return stepNames.map((name, index) => {
    if (name === "Jira" && stage === "sealed" && session.jira !== null) {
      return { name, place: "done", note: session.jira.key }
    }
    if (index < at) {
      return { name, place: "done", note: "done" }
    }
    if (index === at && (stage !== "sealed" || name !== "Jira")) {
      return { name, place: "current", note: "you are here" }
    }
    return {
      name,
      place: "waiting",
      note: optionalSteps.has(name) ? "optional" : "later",
    }
  })
}

/** One line of "Before you seal". `ok` false is a warning, not a block. */
export interface SealCheck {
  id: "examples" | "sources" | "open" | "assumed" | "notes"
  ok: boolean
  title: string
  detail: string
}

function plural(count: number, one: string, many: string): string {
  return count === 1 ? one : many
}

/**
 * What sealing locks in, computed from the session. `pendingNotes` is how many context notes
 * are still proposed: they are dropped when the story is sealed.
 */
export function sealChecks(
  session: ShapeSession,
  pendingNotes: number
): SealCheck[] {
  const withExample = session.rules.filter((rule) =>
    session.examples.some((example) => example.rule === rule.id)
  ).length
  const items = [...session.rules, ...session.examples]
  const sourced = items.filter(
    (item) => item.source.kind === "prompt" || item.source.ref !== ""
  ).length
  const open = questionCounts(session).open
  const assumed = session.questions.filter(
    (question) => question.status === "assumed"
  )
  const checks: SealCheck[] = [
    {
      id: "examples",
      ok: withExample === session.rules.length,
      title: "Every rule has an example.",
      detail: `${withExample} of ${session.rules.length} rules`,
    },
    {
      id: "sources",
      ok: sourced === items.length,
      title: "Every item has a source.",
      detail: `${sourced} of ${items.length} items`,
    },
    {
      id: "open",
      ok: open === 0,
      title: "No open questions.",
      detail: `${open} open`,
    },
  ]
  if (assumed.length > 0) {
    checks.push({
      id: "assumed",
      ok: false,
      title: `${assumed.length} ${plural(assumed.length, "assumption", "assumptions")} will be sealed in`,
      detail: assumed
        .map((question) => `${question.id}: ${question.answer}`)
        .join(" · "),
    })
  }
  if (pendingNotes > 0) {
    checks.push({
      id: "notes",
      ok: false,
      title: `${pendingNotes} context ${plural(pendingNotes, "note", "notes")} not accepted yet`,
      detail: "It will be dropped",
    })
  }
  return checks
}

/** A source as the person reads it: an answer is named by its answer id. */
export function answerSource(
  session: ShapeSession,
  source: ShapeSource
): string {
  if (source.kind === "answer") {
    const question = session.questions.find((item) => item.id === source.ref)
    const ref = question?.answerId ?? ""
    return `answer · ${ref !== "" ? ref : source.ref}`.trim()
  }
  return source.ref === "" ? source.kind : `${source.kind} · ${source.ref}`
}

/** The path and line of a code source, or null when it cannot be opened. */
export function parseSourceRef(
  source: ShapeSource
): { path: string; line: number } | null {
  if (source.kind !== "code" || source.ref.trim() === "") {
    return null
  }
  const at = source.ref.lastIndexOf(":")
  const line = Number.parseInt(source.ref.slice(at + 1), 10)
  if (at > 0 && Number.isInteger(line) && line > 0) {
    return { path: source.ref.slice(0, at), line }
  }
  return { path: source.ref, line: 1 }
}

/** The request Span makes to Jira, as text. The description is sent as an Atlassian document. */
export function issuePayload(preview: ShapePreview): string {
  return JSON.stringify(
    {
      fields: {
        project: { key: preview.project },
        issuetype: { name: "Story" },
        summary: preview.summary,
        description: preview.description,
        labels: ["span"],
      },
    },
    null,
    2
  )
}

/** A label followed by the id it names, or the bare label when there is no id yet. */
export function labelled(label: string, id: string | undefined): string {
  return id ? `${label} · ${id}` : label
}
