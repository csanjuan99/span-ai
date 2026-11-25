import { describe, expect, it } from "vitest"

import {
  filterCommands,
  isTyping,
  nextChord,
  paletteCommands,
  runCommandFor,
  runKeys,
  runPaletteCommands,
  shortcuts,
} from "@/lib/help"

describe("paletteCommands", () => {
  it("lists only what the app does: the pages, the theme and the shortcut sheet", () => {
    const ids = paletteCommands().map((command) => command.id)
    expect(ids).toEqual([
      "go-desk",
      "go-settings",
      "go-metrics",
      "go-shaping",
      "theme-system",
      "theme-light",
      "theme-dark",
      "shortcuts",
    ])
  })

  it("gives every command a title, a hint and a group", () => {
    for (const command of paletteCommands()) {
      expect(command.title).not.toBe("")
      expect(command.hint).not.toBe("")
      expect(command.group).not.toBe("")
    }
  })
})

describe("filterCommands", () => {
  const all = paletteCommands()

  it("keeps everything for an empty query", () => {
    expect(filterCommands(all, "  ")).toHaveLength(all.length)
  })

  it("matches the title or the hint, ignoring case", () => {
    expect(
      filterCommands(all, "SETTINGS").map((command) => command.id)
    ).toEqual(["go-settings"])
    expect(
      filterCommands(all, "every key").map((command) => command.id)
    ).toEqual(["shortcuts"])
  })

  it("says nothing matches with an empty list", () => {
    expect(filterCommands(all, "zzz")).toEqual([])
  })
})

describe("shortcuts", () => {
  it("lists the keys the page really answers", () => {
    expect(
      shortcuts.flatMap((group) => group.items.map((item) => item.label))
    ).toEqual([
      "Command palette",
      "This sheet",
      "Go to Desk",
      "Go to Settings",
      "Close a panel",
    ])
  })
})

describe("isTyping", () => {
  it("is true for a field and false for the page", () => {
    expect(isTyping(document.createElement("input"))).toBe(true)
    expect(isTyping(document.createElement("textarea"))).toBe(true)
    const editable = document.createElement("div")
    editable.setAttribute("contenteditable", "true")
    expect(isTyping(editable)).toBe(true)
    expect(isTyping(document.createElement("button"))).toBe(false)
    expect(isTyping(null)).toBe(false)
  })
})

describe("nextChord", () => {
  it("g then d goes to the desk, g then s to settings", () => {
    expect(nextChord("", "g")).toEqual({ pending: "g", go: "" })
    expect(nextChord("g", "d")).toEqual({ pending: "", go: "/" })
    expect(nextChord("g", "s")).toEqual({ pending: "", go: "/settings" })
  })

  it("forgets the chord on any other key", () => {
    expect(nextChord("g", "x")).toEqual({ pending: "", go: "" })
    expect(nextChord("", "d")).toEqual({ pending: "", go: "" })
  })
})

describe("run context", () => {
  const ctx = {
    label: "SPAN-482",
    commands: [
      {
        id: "resume",
        title: "Resume",
        hint: "again",
        key: "r",
        run: () => undefined,
      },
      {
        id: "stop",
        title: "Stop",
        hint: "ends",
        key: "s",
        shift: true,
        run: () => undefined,
      },
    ],
  }
  it("finds a command by key and Shift, never the other way", () => {
    expect(runCommandFor(ctx, "R", false)?.id).toBe("resume")
    expect(runCommandFor(ctx, "s", false)).toBeUndefined()
    expect(runCommandFor(ctx, "S", true)?.id).toBe("stop")
    expect(runCommandFor(null, "r", false)).toBeUndefined()
  })
  it("draws Shift as a key cap and names the group after the run", () => {
    expect(runKeys(ctx.commands[1]!)).toEqual(["⇧", "S"])
    const lines = runPaletteCommands(ctx)
    expect(lines.map((line) => line.group)).toEqual([
      "This run · SPAN-482",
      "This run · SPAN-482",
    ])
    expect(runPaletteCommands(null)).toEqual([])
  })
})
