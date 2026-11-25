import { describe, expect, it } from "vitest"

import { agoOf, deskCounts, groupRuns, pillOf, searchRuns } from "@/lib/desk"
import { head } from "@/test/heads"

const runs = [
  head({
    id: "a",
    verdict: "hold",
    stage: "merge",
    issue: "SPAN-1",
    title: "Retry webhooks",
    branch: "feat/retry",
    reason: "conflict in 3 files",
  }),
  head({
    id: "b",
    verdict: "stop",
    stage: "prove",
    issue: "SPAN-2",
    title: "Export CSV",
  }),
  head({ id: "c", verdict: "running", issue: "SPAN-3", title: "Cache rates" }),
  head({
    id: "d",
    verdict: "done",
    stage: "note",
    issue: "SPAN-4",
    title: "Validate codes",
  }),
]

describe("groupRuns", () => {
  it("orders the groups by what needs a person first", () => {
    expect(
      groupRuns(runs, "all").map((group) => [group.key, group.runs.length])
    ).toEqual([
      ["waiting", 1],
      ["failed", 1],
      ["running", 1],
      ["done", 1],
    ])
  })

  it("leaves out an empty group", () => {
    expect(groupRuns(runs.slice(2), "all").map((group) => group.key)).toEqual([
      "running",
      "done",
    ])
  })

  it("filters: needs me is waiting and failed", () => {
    expect(groupRuns(runs, "needs").map((group) => group.key)).toEqual([
      "waiting",
      "failed",
    ])
  })
})

describe("deskCounts", () => {
  it("counts the runs a person has to act on", () => {
    expect(deskCounts(runs)).toEqual({ all: 4, needs: 2, running: 1, done: 1 })
  })
})

describe("searchRuns", () => {
  it("matches the key, the title, the branch and the id, ignoring case", () => {
    expect(searchRuns(runs, "span-2").map((run) => run.id)).toEqual(["b"])
    expect(searchRuns(runs, "WEBHOOKS").map((run) => run.id)).toEqual(["a"])
    expect(searchRuns(runs, "feat/retry").map((run) => run.id)).toEqual(["a"])
  })

  it("an empty search keeps every run", () => {
    expect(searchRuns(runs, "  ")).toHaveLength(4)
  })
})

describe("pillOf", () => {
  it("names the stage and why it waits", () => {
    expect(pillOf(runs[0])).toBe("merge · conflict in 3 files")
  })
  it("says running for a run in motion and done for a finished one", () => {
    expect(pillOf(runs[2])).toBe("build · running")
    expect(pillOf(runs[3])).toBe("note · done")
  })
  it("falls back to the verdict when no reason was kept", () => {
    expect(pillOf(runs[1])).toBe("prove · failed")
  })
})

describe("agoOf", () => {
  const now = new Date("2026-11-24T15:00:00Z")
  it("reads a time as how long ago", () => {
    expect(agoOf("2026-11-24T14:59:40Z", now)).toBe("just now")
    expect(agoOf("2026-11-24T14:44:00Z", now)).toBe("16 min ago")
    expect(agoOf("2026-11-24T12:00:00Z", now)).toBe("3 h ago")
    expect(agoOf("2026-11-21T15:00:00Z", now)).toBe("3 days ago")
  })
  it("says nothing for a missing, bad or future time", () => {
    expect(agoOf("", now)).toBe("")
    expect(agoOf("nope", now)).toBe("")
    expect(agoOf("2026-11-25T15:00:00Z", now)).toBe("")
  })
})
