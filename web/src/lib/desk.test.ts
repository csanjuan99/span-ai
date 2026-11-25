import { describe, expect, it } from "vitest"

import {
  actionOf,
  agoOf,
  deskCounts,
  groupRuns,
  pillOf,
  searchRuns,
  whenOf,
} from "@/lib/desk"
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
      groupRuns(runs, "all", clock).map((group) => [
        group.key,
        group.runs.length,
      ])
    ).toEqual([
      ["waiting", 1],
      ["failed", 1],
      ["running", 1],
      ["done", 1],
    ])
  })

  it("leaves out an empty group", () => {
    expect(
      groupRuns(runs.slice(2), "all", clock).map((group) => group.key)
    ).toEqual(["running", "done"])
  })

  it("filters: needs me is waiting and failed", () => {
    expect(groupRuns(runs, "needs", clock).map((group) => group.key)).toEqual([
      "waiting",
      "failed",
    ])
  })
})

describe("deskCounts", () => {
  it("counts the runs a person has to act on", () => {
    expect(deskCounts(runs, clock)).toEqual({
      all: 4,
      needs: 2,
      running: 1,
      done: 1,
    })
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

const clock = new Date("2026-11-24T15:00:00Z")

describe("pillOf by what a run waits for", () => {
  it("counts the files, the questions and the decisions", () => {
    const wait = (need: string, count: number) =>
      pillOf(head({ id: "x", verdict: "hold", stage: "merge", need, count }))
    expect(wait("conflict", 3)).toBe("merge · conflict in 3 files")
    expect(wait("conflict", 1)).toBe("merge · conflict in 1 file")
    expect(wait("questions", 3)).toBe("merge · 3 questions")
    expect(wait("questions", 1)).toBe("merge · 1 question")
    expect(wait("decision", 1)).toBe("merge · 1 decision")
  })
})

describe("actionOf", () => {
  it("offers the action that fits what the run needs", () => {
    const of = (over: Partial<Parameters<typeof head>[0]>) =>
      actionOf(head({ id: "x", ...over }))
    expect(of({ verdict: "hold", need: "conflict", count: 2 })).toEqual({
      label: "Resolve",
      primary: true,
      to: "/spans/x",
    })
    expect(of({ verdict: "hold", need: "questions" }).label).toBe("Answer")
    expect(of({ verdict: "hold", need: "decision" }).label).toBe("Decide")
    expect(of({ verdict: "hold" }).label).toBe("Review")
    expect(of({ verdict: "stop" }).label).toBe("Resume")
    expect(of({ verdict: "running" })).toEqual({
      label: "Open",
      primary: false,
      to: "/spans/x",
    })
  })

  it("sends a finished run to its summary", () => {
    expect(actionOf(head({ id: "x", verdict: "done" }))).toEqual({
      label: "Summary",
      primary: false,
      to: "/spans/x/summary",
    })
  })
})

describe("whenOf", () => {
  it("says when a running run started, and when the others last moved", () => {
    const moved = "2026-11-24T14:48:00Z"
    expect(
      whenOf(head({ id: "x", verdict: "running", startedAt: moved }), clock)
    ).toBe("started 12 min ago")
    expect(
      whenOf(head({ id: "x", verdict: "hold", updatedAt: moved }), clock)
    ).toBe("12 min ago")
    expect(whenOf(head({ id: "x", verdict: "running" }), clock)).toBe("")
  })
})

describe("old finished runs", () => {
  const old = head({
    id: "old",
    verdict: "done",
    updatedAt: "2026-10-01T10:00:00Z",
  })
  const fresh = head({
    id: "fresh",
    verdict: "done",
    updatedAt: "2026-11-20T10:00:00Z",
  })

  it("stay out of All and Needs me but show under Done", () => {
    expect(groupRuns([old, fresh], "all", clock)[0]?.runs).toEqual([fresh])
    expect(groupRuns([old, fresh], "done", clock)[0]?.runs).toHaveLength(2)
  })

  it("count the same way", () => {
    const counts = deskCounts([old, fresh], clock)
    expect(counts.all).toBe(1)
    expect(counts.done).toBe(2)
  })

  it("a finished run with an unknown time is never hidden", () => {
    expect(
      groupRuns([head({ id: "x", verdict: "done" })], "all", clock)
    ).toHaveLength(1)
  })
})
