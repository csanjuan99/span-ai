import { describe, expect, it } from "vitest"

import { metaTime } from "@/lib/run-header"

const now = new Date(2026, 10, 25, 14, 49)
const stamp = (hours: number, minutes: number) =>
  new Date(2026, 10, 25, hours, minutes).toISOString()

describe("metaTime", () => {
  it("says when a working run began and for how long, without 'ago'", () => {
    expect(
      metaTime({
        state: "running",
        startedAt: stamp(14, 8),
        lastStamp: "",
        now,
      })
    ).toBe("started 14:08 · 41 min")
  })

  it("says when a held run was held and how long ago", () => {
    expect(
      metaTime({ state: "hold", startedAt: "", lastStamp: stamp(14, 36), now })
    ).toBe("held 14:36 · 13 min ago")
  })

  it("says since when a run waits for a person", () => {
    expect(
      metaTime({
        state: "waiting",
        startedAt: "",
        lastStamp: stamp(14, 19),
        now,
      })
    ).toBe("waiting since 14:19 · 30 min ago")
  })

  it("says when a stopped run stopped", () => {
    expect(
      metaTime({ state: "stop", startedAt: "", lastStamp: stamp(14, 36), now })
    ).toBe("stopped 14:36 · 13 min ago")
  })

  it("names only the clock time of a finished run", () => {
    expect(
      metaTime({ state: "done", startedAt: "", lastStamp: stamp(14, 36), now })
    ).toBe("finished 14:36")
  })

  it("says how long a finished run took", () => {
    expect(
      metaTime({
        state: "done",
        startedAt: stamp(13, 3),
        lastStamp: stamp(14, 36),
        now,
      })
    ).toBe("finished 14:36 · took 1 h 33 min")
  })

  it("says nothing when the stamp is unknown", () => {
    expect(metaTime({ state: "hold", startedAt: "", lastStamp: "", now })).toBe(
      ""
    )
  })
})
