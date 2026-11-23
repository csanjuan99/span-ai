import { describe, expect, it } from "vitest"

import { coveredByReplay, frameGate, reconnectDelay } from "@/lib/reconnect"

const middle = () => 0.5

describe("reconnectDelay", () => {
  it("doubles from half a second", () => {
    expect(reconnectDelay(0, middle)).toBe(500)
    expect(reconnectDelay(1, middle)).toBe(1000)
    expect(reconnectDelay(2, middle)).toBe(2000)
    expect(reconnectDelay(4, middle)).toBe(8000)
  })

  it("caps at ten seconds", () => {
    expect(reconnectDelay(5, middle)).toBe(10000)
    expect(reconnectDelay(60, middle)).toBe(10000)
    expect(reconnectDelay(Number.MAX_SAFE_INTEGER, middle)).toBe(10000)
  })

  it("jitters by at most a fifth either way", () => {
    expect(reconnectDelay(0, () => 0)).toBe(400)
    expect(reconnectDelay(0, () => 1)).toBe(600)
    expect(reconnectDelay(9, () => 0)).toBe(8000)
    expect(reconnectDelay(9, () => 1)).toBe(12000)
    for (let attempt = 0; attempt < 20; attempt += 1) {
      const delay = reconnectDelay(attempt)
      expect(delay).toBeGreaterThanOrEqual(400)
      expect(delay).toBeLessThanOrEqual(12000)
    }
  })

  it("treats a negative or broken attempt as the first", () => {
    expect(reconnectDelay(-3, middle)).toBe(500)
    expect(reconnectDelay(Number.NaN, middle)).toBe(500)
  })

  it("is never negative even with a broken random source", () => {
    expect(reconnectDelay(0, () => -100)).toBeGreaterThanOrEqual(0)
  })
})

describe("frameGate", () => {
  it("resets on a replay", () => {
    expect(frameGate(9, { type: "replay", seq: 3 })).toBe("reset")
    expect(frameGate(9, { type: "replay" })).toBe("reset")
  })

  it("applies a frame from a server that does not number them", () => {
    expect(frameGate(4, { type: "cursor" })).toBe("apply")
  })

  it("skips a frame the replay already covered", () => {
    expect(frameGate(4, { type: "cursor", seq: 4 })).toBe("skip")
    expect(frameGate(4, { type: "cursor", seq: 2 })).toBe("skip")
  })

  it("applies the next frame", () => {
    expect(frameGate(4, { type: "cursor", seq: 5 })).toBe("apply")
  })

  it("reports a gap when frames went missing", () => {
    expect(frameGate(4, { type: "cursor", seq: 7 })).toBe("gap")
  })
})

describe("coveredByReplay", () => {
  it("drops a transcript frame that ends at or before the replay's offset", () => {
    expect(coveredByReplay(16, { type: "transcript", offset: 16 })).toBe(true)
    expect(coveredByReplay(16, { type: "transcript", offset: 4 })).toBe(true)
  })

  it("keeps a transcript frame past the offset", () => {
    expect(coveredByReplay(16, { type: "transcript", offset: 20 })).toBe(false)
  })

  it("keeps frames without an offset and frames that are not transcript", () => {
    expect(coveredByReplay(16, { type: "transcript" })).toBe(false)
    expect(coveredByReplay(16, { type: "cursor", offset: 3 })).toBe(false)
  })
})
