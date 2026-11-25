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
  { key: "failed", title: "Failed", hint: "the run stopped and needs a look" },
  { key: "running", title: "Running", hint: "nothing needed" },
  { key: "done", title: "Done", hint: "" },
]

/** Which group a run belongs to: a hold waits for a person, a stop failed, the rest follow the verdict. */
export function groupOf(run: SpanHead): GroupKey {
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

/** The runs under the chosen filter, grouped, newest first inside a group. Empty groups are left out. */
export function groupRuns(runs: SpanHead[], filter: DeskFilter): RunGroup[] {
  return groups
    .filter((group) => shown[filter].includes(group.key))
    .map((group) => ({
      ...group,
      runs: runs
        .filter((run) => groupOf(run) === group.key)
        .sort((a, b) => b.id.localeCompare(a.id)),
    }))
    .filter((group) => group.runs.length > 0)
}

/** How many runs each filter would show. */
export function deskCounts(runs: SpanHead[]): Record<DeskFilter, number> {
  const count = (keys: GroupKey[]) =>
    runs.filter((run) => keys.includes(groupOf(run))).length
  return {
    all: runs.length,
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

/** "merge · conflict in 3 files": the stage a run is at and why it waits. */
export function pillOf(run: SpanHead): string {
  const group = groupOf(run)
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
