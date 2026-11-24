import { describe, expect, it } from "vitest"

import { keptWork } from "@/lib/kept"
import type { Statement } from "@/lib/types"

function sealed(stage: string, verdict = "advance"): Statement {
  return { stage, iteration: 1, verdict, reason: "", commands: [] }
}

const stat = " web/a.ts | 12 ++++++++----\n internal/b.go | 3 +++\n 2 files changed, 11 insertions(+), 4 deletions(-)\n"

describe("keptWork", () => {
  it("says nothing before any stage is sealed", () => {
    expect(keptWork({ statements: [], branch: "", head: "", stat: "" })).toBeNull()
    expect(keptWork({ statements: [sealed("intake", "hold")], branch: "", head: "", stat: "" })).toBeNull()
  })

  it("counts a stage once however many times it was sealed", () => {
    const result = keptWork({
      statements: [sealed("intake"), sealed("frame"), sealed("build"), sealed("build")],
      branch: "span/x",
      head: "abcdef1234567",
      stat: "",
    })
    expect(result?.sealed).toBe(3)
  })

  it("names the branch and a short head, and leaves out what it does not know", () => {
    const result = keptWork({ statements: [sealed("intake")], branch: "span/x", head: "abcdef1234567", stat: "" })
    expect(result).toEqual({ sealed: 1, branch: "span/x", head: "abcdef1", files: null })
    expect(keptWork({ statements: [sealed("intake")], branch: "", head: "", stat: "" })?.branch).toBe("")
  })

  it("reads the changed files and the totals from the diff stat summary", () => {
    const result = keptWork({ statements: [sealed("build")], branch: "b", head: "h", stat })
    expect(result?.files).toEqual({ files: 2, plus: 11, minus: 4 })
  })

  it("trusts the summary over the bars, which git scales down on a large diff", () => {
    const big = " a.ts | 900 ++++++++++++++++++++++++++++++++++++++++\n 1 file changed, 880 insertions(+), 20 deletions(-)\n"
    expect(keptWork({ statements: [sealed("build")], branch: "b", head: "h", stat: big })?.files).toEqual({
      files: 1,
      plus: 880,
      minus: 20,
    })
  })

  it("falls back to the bars when the stat has no summary line", () => {
    const bars = " a.ts | 3 ++-\n"
    expect(keptWork({ statements: [sealed("build")], branch: "b", head: "h", stat: bars })?.files).toEqual({
      files: 1,
      plus: 2,
      minus: 1,
    })
  })
})
