import { duration } from "@/lib/metrics"
import type { Statement } from "@/lib/types"

/** What a person looks up after a run: which agents worked and how long it took. */
export interface RunDetails {
  agents: string[]
  time: { started: string; took: string } | null
}

function clock(date: Date): string {
  return `${date.getHours().toString().padStart(2, "0")}:${date.getMinutes().toString().padStart(2, "0")}`
}

/**
 * Reads the end-of-run facts from the ledger and the agent choices. The time runs from the
 * first stamped statement to the last, which is when Span wrote them, not when a person
 * started or stopped looking.
 */
export function runDetails(statements: Statement[], stageAgents: Record<string, string> | undefined): RunDetails {
  const stamps = statements
    .map((item) => new Date(item.time ?? ""))
    .filter((date) => !Number.isNaN(date.getTime()))
  const first = stamps[0]
  const last = stamps[stamps.length - 1]
  return {
    agents: [...new Set(Object.values(stageAgents ?? {}))],
    time:
      first === undefined || last === undefined || stamps.length < 2
        ? null
        : { started: clock(first), took: duration((last.getTime() - first.getTime()) / 1000) },
  }
}
