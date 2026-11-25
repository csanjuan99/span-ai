import { describe, expect, it } from "vitest"

import { sinceLine } from "@/lib/since"

const at = (h: number, m: number) => new Date(2026, 10, 24, h, m)

describe("sinceLine", () => {
  it("says when and how long ago", () => {
    expect(sinceLine("held", at(14, 36).toISOString(), at(14, 52))).toBe("held 14:36 · 16 min ago")
  })

  it("says 'just now' under a minute and counts hours and minutes past an hour", () => {
    expect(sinceLine("held", at(14, 36).toISOString(), new Date(at(14, 36).getTime() + 20_000))).toBe("held 14:36 · just now")
    expect(sinceLine("waiting since", at(12, 30).toISOString(), at(14, 36))).toBe("waiting since 12:30 · 2 h 6 min ago")
    expect(sinceLine("held", at(13, 36).toISOString(), at(14, 36))).toBe("held 13:36 · 1 h ago")
  })

  it("names the day when it is not today", () => {
    const yesterday = new Date(2026, 10, 23, 22, 5)
    expect(sinceLine("held", yesterday.toISOString(), at(9, 0))).toBe("held yesterday 22:05")
    expect(sinceLine("held", new Date(2026, 10, 20, 8, 0).toISOString(), at(9, 0))).toBe("held 20 Nov 08:00")
  })

  it("stays empty when the time is unknown or in the future", () => {
    expect(sinceLine("held", "", at(14, 0))).toBe("")
    expect(sinceLine("held", "not a date", at(14, 0))).toBe("")
    expect(sinceLine("held", at(15, 0).toISOString(), at(14, 0))).toBe("")
  })
})
