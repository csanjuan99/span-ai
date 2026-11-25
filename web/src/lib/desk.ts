import { tidy } from "@/lib/tidy"
import type { SpanHead } from "@/lib/types"

/** What the desk is filtered to. `needs` is waiting plus failed. */
export type DeskFilter = "all" | "needs" | "running" | "done"

/** The groups the desk lists, in the order a person should look at them. */
export type GroupKey = "waiting" | "failed" | "running" | "done"

export interface RunGroup {
  key: GroupKey
  title: string
  hint: string
  runs: SpanHead[]
}

const groups: { key: GroupKey; title: string; hint: string }[] = [
  {
    key: "waiting",
    title: "Waiting for me",
    hint: "answer, decide or resolve",
  },
  { key: "failed", title: "Failed", hint: "a check did not pass" },
  { key: "running", title: "Running", hint: "nothing needed" },
  { key: "done", title: "Done", hint: "" },
]

/** A held run whose check failed, worded the way the engine and the fixtures do. */
function checkFailed(run: SpanHead): boolean {
  return /\bexit \d+\b/i.test(run.reason) || /check failed/i.test(run.reason)
}

/**
 * Which group a run belongs to. A hold usually waits for a person. A hold that is only a
 * failed check sits with the failures, so the desk offers Resume instead of Review.
 */
export function groupOf(run: SpanHead): GroupKey {
  if (run.verdict === "hold" && run.need === "" && checkFailed(run)) {
    return "failed"
  }
  switch (run.verdict) {
    case "hold":
      return "waiting"
    case "stop":
      return "failed"
    case "done":
      return "done"
    default:
      return "running"
  }
}

const shown: Record<DeskFilter, GroupKey[]> = {
  all: ["waiting", "failed", "running", "done"],
  needs: ["waiting", "failed"],
  running: ["running"],
  done: ["done"],
}

/** A finished run stays on the desk this long; after that only the Done filter lists it. */
const doneVisibleDays = 30

function hiddenDone(run: SpanHead, now: Date): boolean {
  if (groupOf(run) !== "done") {
    return false
  }
  const moved = new Date(run.updatedAt)
  return (
    !Number.isNaN(moved.getTime()) &&
    now.getTime() - moved.getTime() > doneVisibleDays * 86_400_000
  )
}

/** The runs under the chosen filter, grouped, newest first inside a group. Empty groups are left out. */
export function groupRuns(
  runs: SpanHead[],
  filter: DeskFilter,
  now: Date
): RunGroup[] {
  return groups
    .filter((group) => shown[filter].includes(group.key))
    .map((group) => ({
      ...group,
      runs: runs
        .filter(
          (run) =>
            groupOf(run) === group.key &&
            (filter === "done" || !hiddenDone(run, now))
        )
        .sort((a, b) => b.id.localeCompare(a.id)),
    }))
    .filter((group) => group.runs.length > 0)
}

/** How many runs each filter would show. */
export function deskCounts(
  runs: SpanHead[],
  now: Date
): Record<DeskFilter, number> {
  const count = (keys: GroupKey[]) =>
    runs.filter((run) => keys.includes(groupOf(run))).length
  return {
    all: runs.filter((run) => !hiddenDone(run, now)).length,
    needs: count(shown.needs),
    running: count(shown.running),
    done: count(shown.done),
  }
}

/** The runs whose key, title, branch, label or id contain the text, ignoring case. */
export function searchRuns(runs: SpanHead[], query: string): SpanHead[] {
  const needle = query.trim().toLowerCase()
  if (needle === "") {
    return runs
  }
  return runs.filter((run) =>
    [run.issue, run.title, run.branch, run.label, run.id].some((field) =>
      field.toLowerCase().includes(needle)
    )
  )
}

function plural(count: number, one: string, many: string): string {
  return `${count} ${count === 1 ? one : many}`
}

/** What a held run waits for, counted: "conflict in 3 files", "2 questions", "1 decision". */
function needText(run: SpanHead): string {
  switch (run.need) {
    case "conflict":
      return `conflict in ${plural(run.count, "file", "files")}`
    case "questions":
      return plural(run.count, "question", "questions")
    case "decision":
      return plural(run.count, "decision", "decisions")
    default:
      return ""
  }
}

/** "merge · conflict in 3 files": the stage a run is at and why it waits. */
export function pillOf(run: SpanHead): string {
  const group = groupOf(run)
  const need = needText(run)
  if (group === "waiting" && need !== "") {
    return `${run.stage} · ${need}`
  }
  const why = tidy(run.reason).replace(/\s+/g, " ").trim()
  if (group === "running") {
    return `${run.stage} · running`
  }
  if (group === "done") {
    return `${run.stage} · done`
  }
  if (why !== "") {
    return `${run.stage} · ${why.length > 60 ? `${why.slice(0, 60).trimEnd()}…` : why}`
  }
  return `${run.stage} · ${group === "failed" ? "failed" : "waiting"}`
}

/** "16 min ago". Empty when the time is missing, unreadable or ahead of `now`. */
export function agoOf(stamp: string, now: Date): string {
  const then = new Date(stamp)
  if (
    stamp === "" ||
    Number.isNaN(then.getTime()) ||
    then.getTime() > now.getTime()
  ) {
    return ""
  }
  const minutes = Math.floor((now.getTime() - then.getTime()) / 60_000)
  if (minutes < 1) {
    return "just now"
  }
  if (minutes < 60) {
    return `${minutes} min ago`
  }
  const hours = Math.floor(minutes / 60)
  if (hours < 24) {
    return `${hours} h ago`
  }
  const days = Math.floor(hours / 24)
  return `${days} ${days === 1 ? "day" : "days"} ago`
}

/** The one thing a row offers: what to press, whether it is the main action, and where it goes. */
export interface RowAction {
  label: string
  primary: boolean
  to: string
}

const waitingLabels: Record<string, string> = {
  conflict: "Resolve",
  questions: "Answer",
  decision: "Decide",
}

/** Resolve a conflict, answer questions, decide, resume a failed run, open a running one, read a finished one. */
export function actionOf(run: SpanHead): RowAction {
  const group = groupOf(run)
  const to = `/spans/${run.id}`
  switch (group) {
    case "waiting":
      return {
        label: waitingLabels[run.need] ?? "Review",
        primary: run.need === "conflict",
        to,
      }
    case "failed":
      return { label: "Resume", primary: false, to }
    case "done":
      return { label: "Summary", primary: false, to: `${to}/summary` }
    default:
      return { label: "Open", primary: false, to }
  }
}

/** "started 12 min ago" for a running run, "2 h ago" for the others. Empty when the time is unknown. */
export function whenOf(run: SpanHead, now: Date): string {
  if (groupOf(run) === "running") {
    const started = agoOf(run.startedAt, now)
    return started === "" ? "" : `started ${started}`
  }
  return agoOf(run.updatedAt, now)
}
