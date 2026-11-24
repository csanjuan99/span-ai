import { duration } from "@/lib/metrics"
import type { Statement } from "@/lib/types"

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
      result[item.stage] = duration(seconds)
    }
    previous = at
  }
  return result
}
