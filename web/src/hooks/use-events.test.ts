import { describe, expect, it } from "vitest"

import { journalState } from "@/hooks/use-events"

describe("journalState", () => {
  it("reads events and the chain verdict", () => {
    const got = journalState({ events: [{ seq: 1, time: "t", actor: "person", type: "stop", detail: { stage: "qa", n: 3 } }], intact: false, brokenAt: 4 })
    expect(got.events).toEqual([{ seq: 1, time: "t", actor: "person", type: "stop", detail: { stage: "qa", n: "" } }])
    expect([got.intact, got.brokenAt]).toEqual([false, 4])
  })

  it("reads anything unexpected as an empty intact journal", () => {
    expect(journalState(null)).toEqual({ events: [], intact: true, brokenAt: 0 })
    expect(journalState({ events: ["x", 5] }).events).toEqual([])
  })
})
