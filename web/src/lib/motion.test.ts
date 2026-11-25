import { afterEach, beforeEach, describe, expect, it } from "vitest"

import {
  applyMotion,
  motionReduced,
  readMotion,
  saveMotion,
} from "@/lib/motion"

function system(reduce: boolean) {
  window.matchMedia = ((query: string) => ({
    matches: reduce && query.includes("reduce"),
    media: query,
    addEventListener: () => undefined,
    removeEventListener: () => undefined,
  })) as unknown as typeof window.matchMedia
}

describe("motion", () => {
  beforeEach(() => {
    localStorage.clear()
    applyMotion("system")
    system(false)
  })
  afterEach(() => applyMotion("system"))

  it("follows the system when nothing was chosen", () => {
    expect(readMotion()).toBe("system")
    expect(motionReduced()).toBe(false)
    system(true)
    expect(motionReduced()).toBe(true)
  })

  it("holds still when the person asks, whatever the system says", () => {
    expect(saveMotion("reduced")).toBe(true)
    expect(readMotion()).toBe("reduced")
    expect(motionReduced()).toBe(true)
    expect(document.getElementById("span-motion")).not.toBeNull()
  })

  it("puts the page back when the choice goes back to the system", () => {
    saveMotion("reduced")
    saveMotion("system")
    expect(document.getElementById("span-motion")).toBeNull()
    expect(motionReduced()).toBe(false)
  })

  it("adds the rule once", () => {
    applyMotion("reduced")
    applyMotion("reduced")
    expect(document.querySelectorAll("#span-motion")).toHaveLength(1)
  })
})
