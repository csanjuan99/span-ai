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
      ["resume", "note", "pause", "stop", "rewind", "transcript", "diff", "copySummary", "ledger", "implementation", "qa"].map((name) => [name, vi.fn()])
    )
    const keys = (runContextOf("x", all)?.commands ?? []).map((command) => `${command.shift === true}${command.key}`)
    expect(new Set(keys).size).toBe(keys.length)
    expect(keys).toHaveLength(11)
  })

  it("switches tabs with the keys 1, 2 and 3", () => {
    const ledger = vi.fn()
    const implementation = vi.fn()
    const qa = vi.fn()
    const context = runContextOf("SPAN-482", { ledger, implementation, qa })
    runCommandFor(context, "1", false)?.run()
    runCommandFor(context, "2", false)?.run()
    runCommandFor(context, "3", false)?.run()
    expect([ledger, implementation, qa].map((fn) => fn.mock.calls.length)).toEqual([1, 1, 1])
  })
})
