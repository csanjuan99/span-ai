import { describe, expect, it } from "vitest"

import { agentBlockLine } from "@/lib/agent-block"

describe("agentBlockLine", () => {
  it("names the spent quota and the way out", () => {
    const line = agentBlockLine({
      kind: "quota",
      agent: "claude",
      detail: "usage limit reached",
    })
    expect(line?.title).toBe("claude is out of quota")
    expect(line?.detail).toBe("usage limit reached")
    expect(line?.hint).toContain("Resume after the limit resets")
  })

  it("asks for a login when the key is missing", () => {
    const line = agentBlockLine({
      kind: "credentials",
      agent: "codex",
      detail: "No API key found for openai-codex.",
    })
    expect(line?.title).toBe("codex has no usable credentials")
    expect(line?.hint).toContain("Log in")
  })

  it("says nothing when no agent is blocked", () => {
    expect(agentBlockLine(null)).toBeNull()
    expect(agentBlockLine({ kind: "quota", agent: "" })).toBeNull()
  })

  it("says what is kept and what to do next, with the login command for a CLI that has one", () => {
    const line = agentBlockLine(
      { kind: "credentials", agent: "codex" },
      { stage: "build", sealed: ["intake", "frame"] }
    )
    expect(line?.kept).toBe("intake and frame are sealed. Nothing is lost.")
    expect(line?.next).toBe(
      "Log in in a terminal, then resume. Or pick another agent for build."
    )
    expect(line?.command).toBe("codex login")
  })

  it("has no command for a CLI whose login is not a plain command", () => {
    expect(
      agentBlockLine(
        { kind: "credentials", agent: "claude" },
        { stage: "build", sealed: [] }
      )?.command
    ).toBeUndefined()
  })

  it("offers to wait or switch when the quota is spent", () => {
    const line = agentBlockLine(
      { kind: "quota", agent: "claude" },
      { stage: "build", sealed: [] }
    )
    expect(line?.kept).toBe("Nothing is sealed yet, and nothing is lost.")
    expect(line?.next).toBe(
      "Choose another agent for build and resume, or wait for the reset."
    )
  })
})
