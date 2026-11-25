import { describe, expect, it, vi } from "vitest"

import { runContextOf } from "@/lib/run-commands"
import { runCommandFor } from "@/lib/help"

describe("runContextOf", () => {
  it("offers only the actions the run can take now", () => {
    const context = runContextOf("SPAN-482", { resume: vi.fn(), diff: vi.fn() })
    expect(context?.label).toBe("SPAN-482")
    expect(context?.commands.map((command) => command.id)).toEqual(["resume", "diff"])
  })

  it("is null when nothing can be done, so the palette shows no empty group", () => {
    expect(runContextOf("SPAN-482", {})).toBeNull()
  })

  it("binds each action to its own key, Shift+S for stop", () => {
    const stop = vi.fn()
    const context = runContextOf("SPAN-482", { stop, resume: vi.fn() })
    runCommandFor(context, "S", true)?.run()
    expect(stop).toHaveBeenCalledOnce()
    expect(runCommandFor(context, "s", false)).toBeUndefined()
  })

  it("gives every command a distinct key", () => {
    const all = Object.fromEntries(
      ["resume", "note", "pause", "stop", "rewind", "transcript", "diff", "copySummary"].map((name) => [name, vi.fn()])
    )
    const keys = (runContextOf("x", all)?.commands ?? []).map((command) => `${command.shift === true}${command.key}`)
    expect(new Set(keys).size).toBe(keys.length)
    expect(keys).toHaveLength(8)
  })
})
