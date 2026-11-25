import { describe, expect, it } from "vitest"

import { runDetails } from "@/lib/run-details"
import type { Statement } from "@/lib/types"

const at = (h: number, m: number) => new Date(2026, 10, 24, h, m).toISOString()
const sealed = (stage: string, time?: string): Statement => ({ stage, iteration: 1, verdict: "advance", reason: "", commands: [], ...(time === undefined ? {} : { time }) })

describe("runDetails", () => {
  it("names each agent once, in the order the stages use them", () => {
    const details = runDetails([], { frame: "claude-code", build: "codex", inspect: "claude-code" })
    expect(details.agents).toEqual(["claude-code", "codex"])
  })

  it("says when the run began and how long it took, from the first and last statement", () => {
    const details = runDetails([sealed("intake", at(14, 8)), sealed("build", at(14, 31)), sealed("note", at(15, 41))], {})
    expect(details.time).toEqual({ started: "14:08", took: "1h 33m" })
  })

  it("has no time when the ledger has fewer than two stamps", () => {
    expect(runDetails([sealed("intake", at(14, 8))], {}).time).toBeNull()
    expect(runDetails([sealed("intake"), sealed("frame")], {}).time).toBeNull()
  })

  it("says nothing about agents when the run recorded none", () => {
    expect(runDetails([], undefined).agents).toEqual([])
  })
})
