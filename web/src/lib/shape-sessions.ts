import { clockTime } from "@/lib/clock"
import type { ShapeSession } from "@/lib/types"

/** One state of a session, which is also a filter chip of the list. */
export type ShapeState = ShapeSession["status"]

/** The words each state is shown with. */
export const stateLabels: Record<ShapeState, string> = {
  interviewing: "Interviewing",
  ready: "Ready",
  split: "Split",
  sealed: "Sealed",
}

/** The filter chips, in the order the design shows them. */
export const listedStates: readonly ShapeState[] = [
  "interviewing",
  "ready",
  "split",
  "sealed",
]

/** The short name of a session: what the server says, else the tail of its id. */
export function sessionRef(session: ShapeSession): string {
  if (session.ref !== undefined && session.ref !== "") {
    return session.ref
  }
  const tail = session.id.slice(session.id.lastIndexOf("-") + 1)
  return `SES-${tail.toUpperCase()}`
}

/** The heading of a session: its story heading, else the first line of the idea. */
export function sessionTitle(session: ShapeSession): string {
  if (session.title !== undefined && session.title !== "") {
    return session.title
  }
  return session.idea.trim().split("\n")[0] ?? ""
}

/** How many sessions each chip counts; `all` is every session in the list. */
export function stateCounts(
  sessions: ShapeSession[]
): Record<"all" | ShapeState, number> {
  const counts = {
    all: sessions.length,
    interviewing: 0,
    ready: 0,
    split: 0,
    sealed: 0,
  }
  for (const session of sessions) {
    counts[session.status] += 1
  }
  return counts
}

/** True when the person has something to do in the session: an answer, a review or a retry. */
export function needsYou(session: ShapeSession): boolean {
  if (session.status === "sealed") {
    return false
  }
  return (
    (session.failure ?? null) !== null ||
    (session.openQuestions ?? 0) > 0 ||
    session.status === "ready" ||
    session.status === "split"
  )
}

const minute = 60_000
const hour = 60 * minute
const day = 24 * hour

function plural(n: number, unit: string): string {
  return `${n} ${unit}${n === 1 ? "" : "s"} ago`
}

/** When a session last changed, in the words of the design: "2 min ago", "yesterday". */
export function ageLabel(iso: string | undefined, now: Date): string {
  const at = iso === undefined ? Number.NaN : Date.parse(iso)
  if (Number.isNaN(at)) {
    return ""
  }
  const elapsed = Math.max(0, now.getTime() - at)
  if (elapsed < minute) {
    return "just now"
  }
  if (elapsed < hour) {
    return `${Math.floor(elapsed / minute)} min ago`
  }
  if (elapsed < day) {
    return `${Math.floor(elapsed / hour)} h ago`
  }
  const days = Math.floor(elapsed / day)
  if (days === 1) {
    return "yesterday"
  }
  if (days < 14) {
    return days < 7 ? plural(days, "day") : "1 week ago"
  }
  return plural(Math.floor(days / 7), "week")
}

/** The agent and model of a session, as "claude-code · Auto" shows them. */
export function agentModel(session: ShapeSession): string {
  const agent = session.agent ?? ""
  const model = session.model ?? ""
  return `${agent === "" ? "Auto" : agent} · ${model === "" ? "Auto" : model}`
}

/** What the row of a session offers, and whether it is the main action. */
export function rowAction(session: ShapeSession): {
  label: string
  variant: "default" | "outline" | "ghost"
} {
  if (session.failure !== null && session.failure !== undefined) {
    return { label: "Resume", variant: "outline" }
  }
  switch (session.status) {
    case "interviewing":
      return { label: "Continue", variant: "outline" }
    case "ready":
      return { label: "Review", variant: "default" }
    case "split":
      return { label: "Review split", variant: "outline" }
    case "sealed":
      return { label: "Open", variant: "ghost" }
  }
}

/** What a stopped interview kept, one line each, for the "What was kept" list. */
export function keptLines(session: ShapeSession): string[] {
  const lines: string[] = []
  if (session.rules.length > 0 || session.examples.length > 0) {
    lines.push(
      `${session.rules.length} ${session.rules.length === 1 ? "rule" : "rules"} and ${session.examples.length} ${session.examples.length === 1 ? "example" : "examples"}, each with its source`
    )
  }
  const answered = session.questions.filter(
    (q) => q.status === "answered"
  ).length
  const assumed = session.questions.filter((q) => q.status === "assumed").length
  if (answered + assumed > 0) {
    lines.push(
      `${answered + assumed} ${answered + assumed === 1 ? "question" : "questions"}: ${answered} answered, ${assumed} assumed`
    )
  }
  lines.push("Your idea and the files it read")
  return lines
}

const resetPattern = /\bresets?\s+(?:at\s+)?(\d{1,2}(?::\d{2})?\s*(?:am|pm)?)/i

/** When the agent says its limit resets, if its message says so ("resets at 18:00"). */
export function resetHint(message: string): string {
  return resetPattern.exec(message)?.[1]?.trim() ?? ""
}

/** The cause of a stopped interview in one sentence, for the failure band. */
export function failureCause(
  failure: NonNullable<ShapeSession["failure"]>,
  fallbackAgent: string
): string {
  const agent = failure.agent !== "" ? failure.agent : fallbackAgent
  const when = clockTime(failure.at)
  const at = when === "" ? "" : ` at ${when}`
  switch (failure.kind) {
    case "usage_limit": {
      const reset = resetHint(failure.message)
      return `${agent} reported a usage limit${at}.${reset === "" ? "" : ` It resets at ${reset}.`}`
    }
    case "repo_changed":
      return `${agent} changed the repository during a read-only turn${at}. Span stopped the interview.`
    case "error":
      return `${agent} stopped${at}.`
  }
}
