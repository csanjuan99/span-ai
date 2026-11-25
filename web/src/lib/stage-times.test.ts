import { describe, expect, it } from "vitest"

import { stageDurations, stageSpan } from "@/lib/stage-times"
import type { Statement } from "@/lib/types"

function statement(stage: string, time: string, verdict = "advance"): Statement {
  return { stage, iteration: 1, verdict, reason: "", time, commands: [] }
}

describe("stageDurations", () => {
  it("measures a stage from the previous statement to its own", () => {
    const result = stageDurations([
      statement("intake", "2026-11-24T14:10:00Z"),
      statement("frame", "2026-11-24T14:25:00Z"),
      statement("build", "2026-11-24T14:31:30Z"),
    ])
    expect(result.frame).toBe("15m")
    expect(result.build).toBe("7m")
  })

  it("leaves the first stage without a duration, since nothing came before it", () => {
    expect(stageDurations([statement("intake", "2026-11-24T14:10:00Z")]).intake).toBeUndefined()
  })

  it("skips stages that did not advance and statements with no time", () => {
    const result = stageDurations([
      statement("intake", "2026-11-24T14:10:00Z"),
      statement("frame", "", "advance"),
      statement("build", "2026-11-24T14:20:00Z", "hold"),
    ])
    expect(result.frame).toBeUndefined()
    expect(result.build).toBeUndefined()
  })

  it("keeps the newest sealing of a stage that ran again", () => {
    const result = stageDurations([
      statement("intake", "2026-11-24T14:00:00Z"),
      statement("build", "2026-11-24T14:10:00Z"),
      statement("prove", "2026-11-24T14:12:00Z", "hold"),
      statement("build", "2026-11-24T14:20:00Z"),
    ])
    expect(result.build).toBe("8m")
  })
})

describe("stageSpan", () => {
  it("words a stage's time as the rail does, rounding minutes", () => {
    expect(stageSpan(46)).toBe("46s")
    expect(stageSpan(564)).toBe("9m")
    expect(stageSpan(120)).toBe("2m")
    expect(stageSpan(3900)).toBe("1h 05m")
  })
})
