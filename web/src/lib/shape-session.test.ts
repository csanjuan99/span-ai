import { describe, expect, it } from "vitest"

import {
  answerSource,
  contextCounts,
  issuePayload,
  parseSourceRef,
  questionCounts,
  sealChecks,
  sessionStage,
  sessionSteps,
} from "@/lib/shape-session"
import type { ShapePreview, ShapeSession } from "@/lib/types"

function session(over: Partial<ShapeSession> = {}): ShapeSession {
  return {
    id: "s1",
    workspace: "/w",
    repo: "",
    idea: "idea",
    status: "interviewing",
    rules: [],
    examples: [],
    questions: [],
    split: [],
    proposals: [],
    story: "",
    digest: "",
    jira: null,
    ...over,
  }
}

const q = (
  id: string,
  status: "open" | "answered" | "assumed",
  answerId = ""
) => ({
  id,
  text: id,
  blocks: "R1",
  assume: "x",
  status,
  answer: status === "open" ? "" : "a",
  answerId,
})

describe("sessionStage", () => {
  it("is questions while one is open, interview while the agent has nothing to ask", () => {
    expect(sessionStage(session({ questions: [q("Q1", "open")] }), {})).toBe(
      "questions"
    )
    expect(sessionStage(session(), {})).toBe("interview")
  })

  it("reviews the map first, then the split, then context, then seal", () => {
    const ready = session({
      status: "ready",
      story: "# s",
      proposals: [{ id: "P1", section: "A", text: "t", status: "proposed" }],
    })
    expect(sessionStage(ready, {})).toBe("map")
    expect(sessionStage(ready, { reviewed: true })).toBe("context")
    expect(sessionStage(ready, { reviewed: true, contextDone: true })).toBe(
      "seal"
    )
    expect(
      sessionStage(session({ status: "ready", story: "# s" }), {
        reviewed: true,
      })
    ).toBe("seal")
    const settled = session({
      status: "ready",
      story: "# s",
      proposals: [{ id: "P1", section: "A", text: "t", status: "accepted" }],
    })
    expect(sessionStage(settled, { reviewed: true })).toBe("seal")
    expect(sessionStage(session({ status: "split" }), { reviewed: true })).toBe(
      "split"
    )
  })

  it("is sealed, and jira only when the person opened it", () => {
    const sealed = session({ status: "sealed", story: "# s" })
    expect(sessionStage(sealed, {})).toBe("sealed")
    expect(sessionStage(sealed, { jira: true })).toBe("jira")
  })
})

describe("sessionSteps", () => {
  it("marks earlier steps done, the current one and the rest waiting", () => {
    const steps = sessionSteps(
      session({ questions: [q("Q1", "open")] }),
      "questions"
    )
    expect(steps.map((step) => step.place)).toEqual([
      "done",
      "done",
      "current",
      "waiting",
      "waiting",
      "waiting",
      "waiting",
      "waiting",
    ])
    expect(steps[2]).toMatchObject({ name: "Questions", note: "you are here" })
    expect(steps[4]).toMatchObject({ name: "Split", note: "optional" })
  })

  it("closes every step but Jira once sealed, and Jira with its key", () => {
    const sealed = session({ status: "sealed", story: "s" })
    expect(sessionSteps(sealed, "sealed")[7]?.place).toBe("waiting")
    const withKey = session({
      status: "sealed",
      jira: { key: "B-1", url: "", status: "" },
    })
    expect(sessionSteps(withKey, "sealed")[7]).toMatchObject({
      place: "done",
      note: "B-1",
    })
    expect(sessionSteps(sealed, "jira")[7]?.place).toBe("current")
  })
})

describe("counts and checks", () => {
  const full = session({
    status: "ready",
    story: "# s",
    rules: [
      { id: "R1", text: "a", source: { kind: "prompt", ref: "" } },
      { id: "R2", text: "b", source: { kind: "code", ref: "x.go:1" } },
    ],
    examples: [
      {
        id: "E1",
        rule: "R1",
        text: "e",
        source: { kind: "answer", ref: "Q1" },
      },
    ],
    questions: [q("Q1", "answered", "A1"), q("Q2", "assumed", "A2")],
    proposals: [
      { id: "P1", section: "A", text: "t", status: "accepted" },
      { id: "P2", section: "B", text: "t", status: "proposed" },
      { id: "P3", section: "C", text: "t", status: "skipped" },
    ],
  })

  it("counts questions and notes", () => {
    expect(questionCounts(full)).toEqual({ open: 0, answered: 1, assumed: 1 })
    expect(contextCounts(full)).toEqual({
      accepted: 1,
      proposed: 1,
      skipped: 1,
    })
  })

  it("lists what sealing will lock in", () => {
    const checks = sealChecks(full, 1)
    expect(checks.find((c) => c.id === "examples")).toMatchObject({
      ok: false,
      detail: "1 of 2 rules",
    })
    expect(checks.find((c) => c.id === "sources")).toMatchObject({
      ok: true,
      detail: "3 of 3 items",
    })
    expect(checks.find((c) => c.id === "open")).toMatchObject({
      ok: true,
      detail: "0 open",
    })
    expect(checks.find((c) => c.id === "assumed")).toMatchObject({
      ok: false,
      title: "1 assumption will be sealed in",
    })
    expect(checks.find((c) => c.id === "notes")).toMatchObject({
      ok: false,
      title: "1 context note not accepted yet",
    })
  })

  it("has no assumption or note line when there is none", () => {
    const clean = session({
      status: "ready",
      questions: [q("Q1", "answered", "A1")],
    })
    expect(sealChecks(clean, 0).map((c) => c.id)).toEqual([
      "examples",
      "sources",
      "open",
    ])
  })
})

describe("sources", () => {
  it("names an answer by its answer id", () => {
    const s = session({ questions: [q("Q1", "answered", "A3")] })
    expect(answerSource(s, { kind: "answer", ref: "Q1" })).toBe("answer · A3")
    expect(answerSource(s, { kind: "code", ref: "a.go:2" })).toBe(
      "code · a.go:2"
    )
    expect(answerSource(s, { kind: "prompt", ref: "" })).toBe("prompt")
  })

  it("reads a code ref as a path and a line only when it can be opened", () => {
    expect(parseSourceRef({ kind: "code", ref: "a/b.go:88" })).toEqual({
      path: "a/b.go",
      line: 88,
    })
    expect(parseSourceRef({ kind: "code", ref: "a/b.go" })).toEqual({
      path: "a/b.go",
      line: 1,
    })
    expect(parseSourceRef({ kind: "answer", ref: "Q1" })).toBeNull()
    expect(parseSourceRef({ kind: "code", ref: "" })).toBeNull()
  })
})

describe("issuePayload", () => {
  const preview: ShapePreview = {
    project: "BILL",
    board: { id: 7, name: "Billing" },
    backlogStatuses: ["Backlog"],
    summary: "Retry it",
    description: "# Retry it\n\nbody",
  }

  it("is the fields Span sends: project, Story, summary, description and the span label", () => {
    const payload = JSON.parse(issuePayload(preview)) as Record<string, unknown>
    expect(payload).toMatchObject({
      fields: {
        project: { key: "BILL" },
        issuetype: { name: "Story" },
        summary: "Retry it",
        labels: ["span"],
      },
    })
    const fields = payload.fields as { description: string }
    expect(fields.description).toContain("# Retry it")
  })
})
