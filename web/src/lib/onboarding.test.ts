import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

import {
  dryLook,
  needsOnboarding,
  readOnboarding,
  saveOnboarding,
  savedAgo,
  stageGroups,
} from "@/lib/onboarding"
import { stages } from "@/lib/types"

beforeEach(() => localStorage.clear())
afterEach(() => vi.restoreAllMocks())

describe("onboarding state", () => {
  it("starts at step 1 with nothing skipped or done", () => {
    expect(readOnboarding()).toEqual({
      step: 1,
      skipped: false,
      done: false,
      folder: "",
      agent: "",
      savedAt: 0,
    })
  })

  it("remembers the step and the folder, so the flow can be resumed", () => {
    saveOnboarding({ step: 2, folder: "/work/api" })
    expect(readOnboarding()).toMatchObject({
      step: 2,
      skipped: false,
      done: false,
      folder: "/work/api",
    })
    saveOnboarding({ step: 3 })
    expect(readOnboarding().folder).toBe("/work/api")
  })

  it("keeps the step inside the three steps", () => {
    saveOnboarding({ step: 9 })
    expect(readOnboarding().step).toBe(3)
    saveOnboarding({ step: 0 })
    expect(readOnboarding().step).toBe(1)
  })

  it("ignores a stored value that is not the state", () => {
    localStorage.setItem("span:onboarding", "not json")
    expect(readOnboarding().step).toBe(1)
    localStorage.setItem(
      "span:onboarding",
      JSON.stringify({ step: "two", skipped: "yes" })
    )
    expect(readOnboarding()).toEqual({
      step: 1,
      skipped: false,
      done: false,
      folder: "",
      agent: "",
      savedAt: 0,
    })
  })

  it("survives storage that refuses to work", () => {
    vi.spyOn(Storage.prototype, "getItem").mockImplementation(() => {
      throw new Error("blocked")
    })
    vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => {
      throw new Error("blocked")
    })
    expect(readOnboarding().step).toBe(1)
    expect(saveOnboarding({ step: 2 })).toBe(false)
  })
})

describe("needsOnboarding", () => {
  it("is true on a first visit", () => {
    expect(needsOnboarding()).toBe(true)
  })
  it("is false once skipped or finished", () => {
    saveOnboarding({ skipped: true })
    expect(needsOnboarding()).toBe(false)
    localStorage.clear()
    saveOnboarding({ done: true })
    expect(needsOnboarding()).toBe(false)
  })
  it("stays true while the flow is only half done", () => {
    saveOnboarding({ step: 2 })
    expect(needsOnboarding()).toBe(true)
  })
})

describe("dryLook", () => {
  it("has one row for every stage the engine walks, in order", () => {
    expect(dryLook.map((row) => row.stage)).toEqual([...stages])
  })
  it("says what each stage touches and when it asks", () => {
    for (const row of dryLook) {
      expect(row.does).not.toBe("")
      expect(row.touches).not.toBe("")
      expect(row.asks).not.toBe("")
    }
  })
})

describe("saving", () => {
  it("stamps the time of the last save", () => {
    const before = Date.now()
    saveOnboarding({ step: 2 })
    expect(readOnboarding().savedAt).toBeGreaterThanOrEqual(before)
  })

  it("remembers the agent that was picked", () => {
    saveOnboarding({ agent: "codex" })
    expect(readOnboarding().agent).toBe("codex")
  })
})

describe("savedAgo", () => {
  const now = 1_000_000_000
  it("says nothing when it was never saved", () => {
    expect(savedAgo(0, now)).toBe("")
  })
  it("counts minutes, then hours, then days", () => {
    expect(savedAgo(now - 20_000, now)).toBe("Saved just now")
    expect(savedAgo(now - 12 * 60_000, now)).toBe("Saved 12 min ago")
    expect(savedAgo(now - 3 * 3_600_000, now)).toBe("Saved 3 h ago")
    expect(savedAgo(now - 2 * 86_400_000, now)).toBe("Saved 2 d ago")
  })
})

describe("stageGroups", () => {
  it("holds every stage the engine walks, once, in order", () => {
    expect(stageGroups.flatMap((group) => group.stages)).toEqual([...stages])
  })
})
