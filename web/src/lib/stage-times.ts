import type { Statement } from "@/lib/types"

/** A stage's time the way the rail words it: 46s, 14m, 1h 05m. Minutes are rounded, as the design draws them. */
export function stageSpan(seconds: number): string {
  const whole = Math.round(seconds)
  if (whole < 60) {
    return `${whole}s`
  }
  if (whole < 3600) {
    return `${Math.round(whole / 60)}m`
  }
  return `${Math.floor(whole / 3600)}h ${String(Math.round((whole % 3600) / 60)).padStart(2, "0")}m`
}

/**
 * How long each sealed stage took, read from the stamps the engine already records: a stage
 * runs from the previous statement to its own. The first statement has nothing before it, so
 * it has no duration. A stage sealed more than once keeps its newest measurement.
 */
export function stageDurations(statements: Statement[]): Record<string, string> {
  const result: Record<string, string> = {}
  let previous = Number.NaN
  for (const item of statements) {
    const at = item.time === undefined || item.time === "" ? Number.NaN : Date.parse(item.time)
    if (Number.isNaN(at)) {
      continue
    }
    const seconds = (at - previous) / 1000
    if (item.verdict === "advance" && !Number.isNaN(previous) && seconds > 0) {
      result[item.stage] = stageSpan(seconds)
    }
    previous = at
  }
  return result
}
