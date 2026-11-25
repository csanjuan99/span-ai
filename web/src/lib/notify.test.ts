import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

import {
  attentionCount,
  badgeOf,
  defaultPrefs,
  loadPrefs,
  notifyEvents,
  savePrefs,
  titleOf,
  toastCopy,
  wantsToast,
} from "@/lib/notify"
import { head } from "@/test/heads"

const waiting = head({ id: "a", issue: "SPAN-488", verdict: "hold", stage: "shape", reason: "3 questions" })
const failed = head({ id: "b", issue: "SPAN-482", verdict: "stop", stage: "verify" })
const running = head({ id: "c", issue: "SPAN-495", verdict: "running" })
const done = head({ id: "d", issue: "SPAN-468", verdict: "done", stage: "note" })

describe("notifyEvents", () => {
  it("says nothing on the first read, so opening the desk does not announce the backlog", () => {
    expect(notifyEvents(null, [waiting, failed, done])).toEqual([])
  })

  it("announces a run that moved into waiting, failed or done", () => {
    const before = [{ ...waiting, verdict: "running" }, { ...failed, verdict: "running" }, { ...running }, { ...done, verdict: "running" }]
    const events = notifyEvents(before, [waiting, failed, running, done])
    expect(events.map((event) => [event.kind, event.run.id])).toEqual([
      ["needs", "a"],
      ["failed", "b"],
      ["finished", "d"],
    ])
  })

  it("does not repeat a run that stays where it was", () => {
    expect(notifyEvents([waiting, failed, done], [waiting, failed, done])).toEqual([])
  })

  it("announces a run that appears already waiting", () => {
    expect(notifyEvents([running], [running, waiting]).map((event) => event.kind)).toEqual(["needs"])
  })

  it("does not announce a run that left a group for running", () => {
    expect(notifyEvents([waiting], [{ ...waiting, verdict: "running" }])).toEqual([])
  })
})

describe("attentionCount, badgeOf and titleOf", () => {
  it("counts the runs a person has to act on", () => {
    expect(attentionCount([waiting, failed, running, done])).toBe(2)
  })

  it("is idle when nothing needs a person", () => {
    expect(badgeOf([running, done], null)).toBe("")
    expect(titleOf([running, done], null)).toBe("Span")
  })

  it("shows how many need me and names the first", () => {
    expect(badgeOf([waiting, { ...waiting, id: "z", issue: "SPAN-1" }], null)).toBe("2")
    expect(titleOf([waiting, { ...waiting, id: "z", issue: "SPAN-1" }], null)).toBe("(2) SPAN-488 needs you · Span")
  })

  it("shows ! for a failure when no run waits", () => {
    expect(badgeOf([failed, running], null)).toBe("!")
    expect(titleOf([failed, running], null)).toBe("(!) SPAN-482 failed · Span")
  })

  it("shows a check for a run that just finished, and a waiting run still wins", () => {
    expect(badgeOf([running], done)).toBe("✓")
    expect(titleOf([running], done)).toBe("(✓) SPAN-468 done · Span")
    expect(badgeOf([waiting], done)).toBe("1")
  })

  it("names a run by its title or id when it has no story key", () => {
    expect(titleOf([head({ id: "x", verdict: "hold", title: "Add healthz" })], null)).toBe("(1) Add healthz needs you · Span")
    expect(titleOf([head({ id: "x", verdict: "hold" })], null)).toBe("(1) x needs you · Span")
  })
})

describe("preferences", () => {
  beforeEach(() => localStorage.clear())
  afterEach(() => vi.restoreAllMocks())

  it("starts with toasts and the tab on, system off, and only in the background", () => {
    const prefs = loadPrefs()
    expect(prefs).toEqual(defaultPrefs)
    expect(prefs.events.needs).toEqual({ toast: true, tab: true, system: false })
    expect(prefs.background).toBe(true)
  })

  it("saves and reads back", () => {
    const next = { ...defaultPrefs, background: false, events: { ...defaultPrefs.events, finished: { toast: false, tab: true, system: true } } }
    expect(savePrefs(next)).toBe(true)
    expect(loadPrefs()).toEqual(next)
  })

  it("keeps the defaults for what a damaged value lacks", () => {
    localStorage.setItem("span:notify", JSON.stringify({ background: false, events: { needs: { toast: false } } }))
    const prefs = loadPrefs()
    expect(prefs.background).toBe(false)
    expect(prefs.events.needs).toEqual({ toast: false, tab: true, system: false })
    expect(prefs.events.failed).toEqual(defaultPrefs.events.failed)
    localStorage.setItem("span:notify", "{not json")
    expect(loadPrefs()).toEqual(defaultPrefs)
  })

  it("says it could not save when storage is blocked", () => {
    vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => {
      throw new Error("blocked")
    })
    expect(savePrefs(defaultPrefs)).toBe(false)
  })

  it("holds a toast back while the tab is in front, when the person asked for that", () => {
    expect(wantsToast(defaultPrefs, "needs", true)).toBe(false)
    expect(wantsToast(defaultPrefs, "needs", false)).toBe(true)
    expect(wantsToast({ ...defaultPrefs, background: false }, "needs", true)).toBe(true)
    const off = { ...defaultPrefs, events: { ...defaultPrefs.events, needs: { toast: false, tab: true, system: false } } }
    expect(wantsToast(off, "needs", false)).toBe(false)
  })
})

describe("toastCopy", () => {
  it("says who needs me and where", () => {
    expect(toastCopy({ kind: "needs", run: waiting })).toEqual({ title: "SPAN-488 needs you", detail: "shape · 3 questions", action: "Review" })
  })
  it("says a stopped run failed, with where it stopped", () => {
    expect(toastCopy({ kind: "failed", run: failed })).toEqual({ title: "SPAN-482 failed", detail: "verify · failed", action: "Open" })
  })
  it("names a finished run by what it was about", () => {
    expect(toastCopy({ kind: "finished", run: { ...done, title: "Validate codes" } })).toEqual({
      title: "SPAN-468 is done",
      detail: "Validate codes",
      action: "Summary",
    })
  })
})
