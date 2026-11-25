import { describe, expect, it } from "vitest"

import { foldContext } from "@/lib/diff-fold"
import type { DiffLine } from "@/lib/diff"

const ctx = (n: number): DiffLine => ({
  kind: "ctx",
  oldNo: n,
  newNo: n,
  text: `c${n}`,
})
const add = (n: number): DiffLine => ({
  kind: "add",
  oldNo: null,
  newNo: n,
  text: `a${n}`,
})

describe("foldContext", () => {
  it("folds a long run of context between two changes and keeps three lines each side", () => {
    const lines = [
      add(1),
      ...Array.from({ length: 12 }, (_, i) => ctx(i + 2)),
      add(14),
    ]
    const segments = foldContext(lines)
    expect(segments.map((s) => `${s.kind}:${s.lines.length}`)).toEqual([
      "lines:1",
      "lines:3",
      "fold:6",
      "lines:3",
      "lines:1",
    ])
  })

  it("folds only the inner side of context at the start or end of a hunk", () => {
    const lines = [...Array.from({ length: 10 }, (_, i) => ctx(i + 1)), add(11)]
    expect(
      foldContext(lines).map((s) => `${s.kind}:${s.lines.length}`)
    ).toEqual(["fold:7", "lines:3", "lines:1"])
  })

  it("leaves a short run of context alone", () => {
    const lines = [add(1), ctx(2), ctx(3), add(4)]
    expect(foldContext(lines)).toHaveLength(3)
    expect(foldContext(lines).every((s) => s.kind === "lines")).toBe(true)
  })
})

describe("limitLines", () => {
  it("keeps the first lines across hunks and drops what follows", async () => {
    const { limitLines } = await import("@/lib/diff-fold")
    const hunk = (header: string, n: number) => ({
      header,
      section: "",
      lines: Array.from({ length: n }, (_, i) => ctx(i + 1)),
    })
    const cut = limitLines(
      { hunks: [hunk("@@ 1", 4), hunk("@@ 2", 4), hunk("@@ 3", 4)] },
      6
    )
    expect(cut.hunks.map((h) => h.lines.length)).toEqual([4, 2])
  })
})
