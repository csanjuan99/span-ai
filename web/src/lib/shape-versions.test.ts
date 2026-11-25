import { describe, expect, it } from "vitest"

import {
  defaultPair,
  revisionDiff,
  sideRows,
  tallyText,
  type StoryLine,
} from "@/lib/shape-versions"

const line = (
  kind: StoryLine["kind"],
  at: number,
  textValue: string
): StoryLine => ({
  kind,
  old: kind === "add" ? 0 : at,
  new: kind === "del" ? 0 : at,
  text: textValue,
})

describe("revisionDiff", () => {
  it("reads items with their source and survives a broken reply", () => {
    const diff = revisionDiff({
      from: { id: "r3", n: 3 },
      to: { id: "r4", n: 4, current: true },
      rules: { changed: 1 },
      items: [
        {
          kind: "rule",
          id: "R3",
          change: "changed",
          before: "a",
          after: "b",
          source: { kind: "answer", ref: "A2" },
        },
        { kind: "nonsense", id: "X", change: "other" },
      ],
      story: [{ kind: "del", old: 13, text: "x" }, { kind: "?" }],
    })
    expect(diff.rules.changed).toBe(1)
    expect(diff.to.current).toBe(true)
    expect(diff.items[0].source).toBe("answer · A2")
    expect(diff.items[1]).toMatchObject({
      kind: "rule",
      change: "changed",
      source: "",
    })
    expect(diff.story[1].kind).toBe("ctx")
    expect(revisionDiff(null).items).toEqual([])
  })
})

describe("sideRows", () => {
  it("puts unchanged lines on both sides and pairs a removed line with the added one", () => {
    const rowsOut = sideRows([
      line("ctx", 12, "same"),
      line("del", 13, "old rule"),
      line("add", 13, "new rule"),
      line("add", 15, "only added"),
      line("ctx", 16, "tail"),
    ])
    expect(rowsOut).toHaveLength(4)
    expect(rowsOut[1].left?.text).toBe("old rule")
    expect(rowsOut[1].right?.text).toBe("new rule")
    expect(rowsOut[2].left).toBeNull()
    expect(rowsOut[2].right?.text).toBe("only added")
    expect(rowsOut[3].left).toBe(rowsOut[3].right)
  })
})

describe("tallyText and defaultPair", () => {
  it("says what changed in words", () => {
    expect(tallyText({ added: 0, removed: 0, changed: 0 })).toBe("no change")
    expect(tallyText({ added: 2, removed: 1, changed: 0 })).toBe(
      "2 added · 1 removed"
    )
  })
  it("picks the last two revisions", () => {
    const rev = (id: string) => ({
      n: 0,
      id,
      at: "",
      actor: "",
      label: "",
      digest: "",
      status: "",
      current: false,
    })
    expect(defaultPair([rev("r1"), rev("r2"), rev("r3")])).toEqual({
      before: "r2",
      after: "r3",
    })
    expect(defaultPair([rev("r1")])).toEqual({ before: "r1", after: "r1" })
    expect(defaultPair([])).toEqual({ before: "", after: "" })
  })
})
