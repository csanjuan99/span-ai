import { describe, expect, it } from "vitest"

import { agentBlockLine } from "@/lib/agent-block"

describe("agentBlockLine", () => {
  it("names the spent quota and the way out", () => {
    const line = agentBlockLine({ kind: "quota", agent: "claude", detail: "usage limit reached" })
    expect(line?.title).toBe("claude is out of quota")
    expect(line?.detail).toBe("usage limit reached")
    expect(line?.hint).toContain("Resume after the limit resets")
  })

  it("asks for a login when the key is missing", () => {
    const line = agentBlockLine({ kind: "credentials", agent: "codex", detail: "No API key found for openai-codex." })
    expect(line?.title).toBe("codex has no usable credentials")
    expect(line?.hint).toContain("Log in")
  })

  it("says nothing when no agent is blocked", () => {
    expect(agentBlockLine(null)).toBeNull()
    expect(agentBlockLine({ kind: "quota", agent: "" })).toBeNull()
  })
})
