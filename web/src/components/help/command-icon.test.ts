import { describe, expect, it } from "vitest"

import { commandIcon, isDanger } from "./command-icon"

describe("commandIcon", () => {
  it("knows the palette commands and marks only stop as destructive", () => {
    expect(commandIcon("go-desk")).toBeDefined()
    expect(commandIcon("run-stop")).toBeDefined()
    expect(commandIcon("unknown")).toBeUndefined()
    expect(isDanger("run-stop")).toBe(true)
    expect(isDanger("run-resume")).toBe(false)
  })
})
