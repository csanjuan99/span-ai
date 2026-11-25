import { describe, expect, it } from "vitest"

import { commandIcon, isDanger } from "./command-icon"

describe("commandIcon", () => {
  it("knows the palette commands and marks only stop as destructive", () => {
    expect(commandIcon("go-desk", "size-4")).not.toBeNull()
    expect(commandIcon("run-stop", "size-4")).not.toBeNull()
    expect(commandIcon("unknown", "size-4")).toBeNull()
    expect(isDanger("run-stop")).toBe(true)
    expect(isDanger("run-resume")).toBe(false)
  })
})
