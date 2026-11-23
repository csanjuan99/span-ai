import { describe, expect, it } from "vitest"

import {
  actionLabel,
  canApply,
  choicePairs,
  conflictHold,
  diffLines,
  isConflictHold,
  kindLabel,
  remaining,
  riskLabel,
} from "@/lib/conflict"
import type { Conflict, ConflictGroup, ConflictOption } from "@/lib/types"

function option(id: string): ConflictOption {
  return { id, title: id, summary: "", risk: "low", action: "ours", command: "", diff: "" }
}

function group(id: string, options: string[] = ["a", "b"]): ConflictGroup {
  return { id, kind: "content", files: [], summary: "", options: options.map(option), chosen: "" }
}

function conflict(over: Partial<Conflict> = {}): Conflict {
  return { base: "main", baseSHA: "", state: "choosing", note: "", groups: [group("g1"), group("g2")], ...over }
}

describe("remaining", () => {
  it("counts the groups without a choice", () => {
    expect(remaining(conflict(), {})).toBe(2)
    expect(remaining(conflict(), { g1: "a" })).toBe(1)
    expect(remaining(conflict(), { g1: "a", g2: "b" })).toBe(0)
  })

  it("does not count a choice naming an option the group does not have", () => {
    expect(remaining(conflict(), { g1: "zzz", g2: "b" })).toBe(1)
  })
})

describe("canApply", () => {
  it("needs every group chosen", () => {
    expect(canApply(conflict(), { g1: "a" })).toBe(false)
    expect(canApply(conflict(), { g1: "a", g2: "a" })).toBe(true)
  })

  it("is false outside the choosing state", () => {
    expect(canApply(conflict({ state: "applying" }), { g1: "a", g2: "a" })).toBe(false)
  })
})

describe("choicePairs", () => {
  it("lists one pair per group in group order", () => {
    expect(choicePairs(conflict(), { g2: "b", g1: "a" })).toEqual([
      { group: "g1", option: "a" },
      { group: "g2", option: "b" },
    ])
  })
})

describe("labels", () => {
  it("names each risk in words", () => {
    expect(riskLabel("low")).toBe("low risk")
    expect(riskLabel("medium")).toBe("medium risk")
    expect(riskLabel("high")).toBe("high risk")
  })

  it("names each action in plain words", () => {
    expect(actionLabel("ours")).toBe("Keep this branch's version")
    expect(actionLabel("theirs")).toBe("Take the base's version")
    expect(actionLabel("combine")).toBe("Combine both")
    expect(actionLabel("regenerate")).toBe("Regenerate")
    expect(actionLabel("delete")).toBe("Delete")
    expect(actionLabel("keep")).toBe("Keep")
  })

  it("names each kind", () => {
    expect(kindLabel("content")).toBe("both changed")
    expect(kindLabel("modify-delete")).toBe("changed and deleted")
    expect(kindLabel("other")).toBe("other")
  })
})

describe("diffLines", () => {
  it("classifies headers, additions, deletions and context", () => {
    const diff = "diff --git a/x b/x\n--- a/x\n+++ b/x\n@@ -1,2 +1,2 @@\n same\n-old\n+new\n"
    expect(diffLines(diff)).toEqual([
      { kind: "header", text: "diff --git a/x b/x" },
      { kind: "header", text: "--- a/x" },
      { kind: "header", text: "+++ b/x" },
      { kind: "header", text: "@@ -1,2 +1,2 @@" },
      { kind: "context", text: " same" },
      { kind: "del", text: "-old" },
      { kind: "add", text: "+new" },
    ])
  })

  it("returns nothing for an empty diff", () => {
    expect(diffLines("")).toEqual([])
  })
})

describe("conflict holds", () => {
  it("recognizes the conflict reason prefix", () => {
    expect(isConflictHold("conflict: choose how to resolve")).toBe(true)
    expect(isConflictHold("conflict: aborted")).toBe(true)
    expect(isConflictHold("go test ./... exited 1")).toBe(false)
  })

  it("tells choosing from aborted", () => {
    expect(conflictHold("conflict: choose how to resolve")).toBe("choose")
    expect(conflictHold("conflict: aborted")).toBe("aborted")
    expect(conflictHold("prove failed")).toBeNull()
  })
})
