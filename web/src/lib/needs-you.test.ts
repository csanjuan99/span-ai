import { describe, expect, it } from "vitest"

import { needsYou, type NeedsInput } from "@/lib/needs-you"

const calm: NeedsInput = {
  agentBlocked: false,
  questions: false,
  decision: false,
  conflict: false,
  approval: false,
  busy: false,
  failed: false,
  stop: false,
  idle: false,
  hold: false,
  done: false,
}

describe("needsYou", () => {
  it("says nothing while the run is going on its own", () => {
    expect(needsYou(calm)).toBeNull()
  })

  it("says nothing once the run is done, even if a failure is still on disk", () => {
    expect(needsYou({ ...calm, done: true, failed: true })).toBeNull()
  })

  it.each([
    [{ agentBlocked: true }, "agent-blocked", "hold", "Agent blocked"],
    [{ questions: true }, "questions", "hold", "Waiting for your answers"],
    [{ decision: true }, "decision", "hold", "Decision needed"],
    [{ conflict: true }, "conflict", "hold", "Merge conflict"],
    [{ approval: true }, "approval", "hold", "Approval needed"],
    [{ busy: true }, "busy", "hold", "Waiting for another run"],
    [{ failed: true }, "failed", "fail", "Check failed"],
    [{ stop: true }, "stopped", "fail", "Run stopped"],
    [{ idle: true }, "idle", "hold", "No agent is running"],
    [{ hold: true }, "held", "hold", "Run paused"],
  ] as const)("names %j as %s", (flags, kind, tone, label) => {
    expect(needsYou({ ...calm, ...flags })).toEqual({ kind, tone, label })
  })

  it("puts a blocked agent before everything else, since nothing else can move", () => {
    expect(needsYou({ ...calm, agentBlocked: true, failed: true, hold: true })?.kind).toBe("agent-blocked")
  })

  it("puts questions before a decision, as the page already does", () => {
    expect(needsYou({ ...calm, questions: true, decision: true })?.kind).toBe("questions")
  })

  it("keeps a busy checkout from reading as a failed check", () => {
    expect(needsYou({ ...calm, busy: true, failed: true, hold: true })?.kind).toBe("busy")
  })

  it("names a conflict before the failed merge statement left on disk", () => {
    expect(needsYou({ ...calm, conflict: true, failed: true })?.kind).toBe("conflict")
  })
})
