import { describe, expect, it } from "vitest"

import { UnauthorizedError } from "@/lib/api"
import { barWidth, duration, loadFailure, metricsOf, percent } from "@/lib/metrics"

describe("metricsOf", () => {
  it("reads anything that is not the reply as no runs", () => {
    for (const body of [null, "x", 3, [], { runs: "no", stages: 4 }]) {
      const got = metricsOf(body)
      expect(got.runs.total).toBe(0)
      expect(got.stages).toEqual([])
      expect(got.recent).toEqual([])
    }
  })

  it("keeps what is well formed and drops what is not", () => {
    const got = metricsOf({
      runs: { total: 2, done: 1, firstTry: -4 },
      stages: [{ stage: "frame", runs: 2, medianSeconds: 30, p90Seconds: "x" }, 7],
      recent: [{ id: "r1", agents: ["build pi", 3] }],
    })
    expect(got.runs).toMatchObject({ total: 2, done: 1, firstTry: 0 })
    expect(got.stages).toHaveLength(1)
    expect(got.stages[0]).toMatchObject({ stage: "frame", median: 30, p90: 0 })
    expect(got.recent[0]?.agents).toEqual(["build pi"])
  })
})

describe("formatting", () => {
  it("writes a time the way a person reads it", () => {
    expect(duration(0)).toBe("—")
    expect(duration(45)).toBe("45s")
    expect(duration(200)).toBe("3m 20s")
    expect(duration(3900)).toBe("1h 05m")
  })

  it("gives a percent only when there is something to divide by", () => {
    expect(percent(2, 3)).toBe("67%")
    expect(percent(0, 0)).toBe("—")
  })

  it("never draws an empty bar for a measured value", () => {
    expect(barWidth(0, 10)).toBe(0)
    expect(barWidth(1, 1000)).toBe(2)
    expect(barWidth(10, 10)).toBe(100)
  })
})

describe("loadFailure", () => {
  it("tells a server that needs its token apart from one that failed", () => {
    expect(loadFailure(new UnauthorizedError())).toMatch(/needs its access token/)
    expect(loadFailure(new Error("boom"))).toBe("boom")
    expect(loadFailure("x")).toBe("the metrics could not load")
  })
})
