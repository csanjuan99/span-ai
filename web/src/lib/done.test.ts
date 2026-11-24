import { describe, expect, it } from "vitest"

import { doneOf } from "@/lib/done"

const input = {
  sealed: 10,
  merged: true,
  sha: "c91f3abdeadbeef",
  base: "main",
  branch: "span/SPAN-482",
  issue: "SPAN-482",
  changeURL: "https://github.com/o/r/pull/8",
}

describe("doneOf", () => {
  it("says every stage sealed and where the work went", () => {
    expect(doneOf(input)).toEqual({
      title: "all 10 stages sealed",
      landed: "Merged into main as c91f3ab",
      branch: "span/SPAN-482",
      issue: "SPAN-482",
      change: { label: "Open PR #8", href: "https://github.com/o/r/pull/8" },
    })
  })

  it("does not claim a merge the ledger did not record", () => {
    expect(doneOf({ ...input, merged: false, sealed: 9 }).landed).toBe("Finished at c91f3ab on span/SPAN-482")
  })

  it("leaves out what is not known instead of inventing it", () => {
    const done = doneOf({ ...input, sha: "", issue: "", changeURL: "", branch: "" })
    expect(done.landed).toBe("Merged into main")
    expect(done.issue).toBe("")
    expect(done.change).toBeNull()
  })

  it("uses the singular for one stage", () => {
    expect(doneOf({ ...input, sealed: 1 }).title).toBe("1 stage sealed")
  })
})
