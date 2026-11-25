import { describe, expect, it } from "vitest"

import {
  ageLabel,
  failureCause,
  resetHint,
  agentModel,
  keptLines,
  needsYou,
  rowAction,
  sessionRef,
  sessionTitle,
  stateCounts,
} from "@/lib/shape-sessions"
import type { ShapeSession } from "@/lib/types"

function session(patch: Partial<ShapeSession> = {}): ShapeSession {
  return {
    id: "20261125-094428.123456789-ab12cd",
    workspace: ".",
    repo: "",
    idea: "Customers see why a delivery failed\nand retry it",
    status: "interviewing",
    rules: [],
    examples: [],
    questions: [],
    split: [],
    proposals: [],
    story: "",
    digest: "",
    jira: null,
    ...patch,
  }
}

describe("shape sessions", () => {
  it("names a session by the server ref, else by the tail of its id", () => {
    expect(sessionRef(session())).toBe("SES-AB12CD")
    expect(sessionRef(session({ ref: "SES-0142" }))).toBe("SES-0142")
  })

  it("titles a session by the server title, else by the first line of the idea", () => {
    expect(sessionTitle(session())).toBe("Customers see why a delivery failed")
    expect(sessionTitle(session({ title: "Show the reason" }))).toBe(
      "Show the reason"
    )
  })

  it("counts sessions by state", () => {
    const rows = [
      session(),
      session({ status: "sealed" }),
      session({ status: "sealed" }),
    ]
    expect(stateCounts(rows)).toEqual({
      all: 3,
      interviewing: 1,
      ready: 0,
      split: 0,
      sealed: 2,
    })
  })

  it("says a sealed session never needs you, and a stopped one does", () => {
    expect(needsYou(session({ status: "sealed" }))).toBe(false)
    expect(needsYou(session({ status: "ready" }))).toBe(true)
    expect(needsYou(session({ openQuestions: 0 }))).toBe(false)
    expect(needsYou(session({ openQuestions: 2 }))).toBe(true)
    expect(
      needsYou(
        session({ failure: { at: "", agent: "", kind: "error", message: "x" } })
      )
    ).toBe(true)
  })

  it("words the age as the design does", () => {
    const now = new Date("2026-11-25T12:00:00Z")
    const ago = (ms: number) => new Date(now.getTime() - ms).toISOString()
    const m = 60_000
    const cases: [number, string][] = [
      [10_000, "just now"],
      [2 * m, "2 min ago"],
      [60 * m, "1 h ago"],
      [30 * 60 * m, "yesterday"],
      [48 * 60 * m, "2 days ago"],
      [7 * 24 * 60 * m, "1 week ago"],
      [14 * 24 * 60 * m, "2 weeks ago"],
    ]
    for (const [ms, text] of cases) {
      expect(ageLabel(ago(ms), now)).toBe(text)
    }
    expect(ageLabel("", now)).toBe("")
    expect(ageLabel(undefined, now)).toBe("")
  })

  it("shows Auto for an agent or model the session does not name", () => {
    expect(agentModel(session())).toBe("Auto · Auto")
    expect(agentModel(session({ agent: "codex", model: "o3" }))).toBe(
      "codex · o3"
    )
  })

  it("offers the action each state needs", () => {
    expect(rowAction(session()).label).toBe("Continue")
    expect(rowAction(session({ status: "ready" }))).toEqual({
      label: "Review",
      variant: "default",
    })
    expect(rowAction(session({ status: "split" })).label).toBe("Review split")
    expect(rowAction(session({ status: "sealed" })).label).toBe("Open")
    expect(
      rowAction(
        session({ failure: { at: "", agent: "", kind: "error", message: "x" } })
      ).label
    ).toBe("Resume")
  })

  it("lists what a stopped interview kept", () => {
    const rule = {
      id: "R1",
      text: "t",
      source: { kind: "prompt" as const, ref: "idea" },
    }
    const lines = keptLines(
      session({
        rules: [rule, rule, rule],
        examples: [{ id: "E1", rule: "R1", text: "t", source: rule.source }],
        questions: [
          {
            id: "Q1",
            text: "",
            blocks: "",
            assume: "",
            status: "answered",
            answer: "a",
          },
          {
            id: "Q2",
            text: "",
            blocks: "",
            assume: "",
            status: "assumed",
            answer: "",
          },
          {
            id: "Q3",
            text: "",
            blocks: "",
            assume: "",
            status: "open",
            answer: "",
          },
        ],
      })
    )
    expect(lines).toEqual([
      "3 rules and 1 example, each with its source",
      "2 questions: 1 answered, 1 assumed",
      "Your idea and the files it read",
    ])
  })

  it("finds the reset time in a limit message", () => {
    expect(resetHint("usage limit reached, resets at 18:00")).toBe("18:00")
    expect(resetHint("limit resets 6pm")).toBe("6pm")
    expect(resetHint("agent crashed")).toBe("")
  })

  it("explains each kind of stop in one sentence", () => {
    const at = "2026-11-25T09:36:00"
    expect(
      failureCause(
        {
          at,
          agent: "claude-code",
          kind: "usage_limit",
          message: "resets at 18:00",
        },
        ""
      )
    ).toBe("claude-code reported a usage limit at 09:36. It resets at 18:00.")
    expect(
      failureCause({ at, agent: "", kind: "error", message: "boom" }, "codex")
    ).toBe("codex stopped at 09:36.")
    expect(
      failureCause(
        { at, agent: "codex", kind: "repo_changed", message: "x" },
        ""
      )
    ).toContain("changed the repository")
  })
})
