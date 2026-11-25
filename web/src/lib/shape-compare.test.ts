import { describe, expect, it } from "vitest"

import {
  comparison,
  mergeCommand,
  suggestedTarget,
  undecided,
  type CompareSession,
} from "@/lib/shape-compare"

const session = (
  id: string,
  status: string,
  sealed = false,
  mergedInto = ""
): CompareSession => ({
  id,
  ref: id,
  title: id,
  status,
  sealed,
  items: 0,
  unique: 0,
  mergedInto,
})

describe("comparison", () => {
  it("reads groups, conflicts and duplicates with safe defaults", () => {
    const got = comparison({
      repo: "acme/billing",
      sessions: [{ id: "a", ref: "SES-A", sealed: true }],
      same: [
        {
          kind: "rule",
          text: "t",
          members: [{ ref: "SES-A", item: "R1", sealed: true }],
        },
      ],
      conflicts: [
        {
          id: "C1",
          title: "days",
          options: [{ text: "30", members: [{ ref: "SES-A" }] }],
        },
      ],
      duplicates: [
        { aRef: "SES-A", bRef: "SES-B", percent: 82, aItems: ["R1", 3] },
      ],
      unique: 5,
    })
    expect(got.sessions[0].sealed).toBe(true)
    expect(got.same[0].members[0].sealed).toBe(true)
    expect(got.conflicts[0].options[0].members[0].ref).toBe("SES-A")
    expect(got.duplicates[0]).toMatchObject({
      percent: 82,
      aItems: ["R1"],
      dismissed: false,
    })
    expect(got.estimate).toBe(true)
    expect(comparison("nope").conflicts).toEqual([])
  })
})

describe("merge helpers", () => {
  it("lists the conflicts that still need a decision", () => {
    const conflicts = [
      { id: "C1", title: "", options: [] },
      { id: "C2", title: "", options: [] },
    ]
    expect(undecided(conflicts, { C1: "a" })).toEqual(["C2"])
  })
  it("writes the same command the page would run", () => {
    expect(mergeCommand("SES-B", ["SES-A"], ["SES-C"], { C1: "SES-B" })).toBe(
      "span shape merge SES-B SES-A --against SES-C --resolve C1=SES-B --yes"
    )
    expect(mergeCommand("SES-B", ["SES-A"], [], {})).toBe(
      "span shape merge SES-B SES-A --yes"
    )
  })
  it("never targets a sealed or already merged session, and prefers the ready one", () => {
    const all = [
      session("a", "interviewing"),
      session("b", "ready"),
      session("c", "sealed", true),
      session("d", "ready", false, "b"),
    ]
    expect(suggestedTarget(all, ["a", "b", "c", "d"])).toBe("b")
    expect(suggestedTarget(all, ["a", "c"])).toBe("a")
    expect(suggestedTarget(all, ["c", "d"])).toBe("")
  })
})
