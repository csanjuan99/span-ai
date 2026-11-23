import { describe, expect, it } from "vitest"

import { agentChoice } from "@/lib/api"

describe("agentChoice", () => {
  it("reads the rows and which one would run", () => {
    const choice = agentChoice({
      agents: [
        { id: "cursor", label: "Cursor", binary: "cursor-agent", installed: false, gentle: false, ready: false, reason: "cursor-agent is not on PATH" },
        { id: "pi", label: "Pi", binary: "pi", installed: true, gentle: true, ready: true },
      ],
      selected: "pi",
      chosen: "pi",
      exists: true,
    })
    expect(choice.agents).toHaveLength(2)
    expect(choice.agents[0].reason).toContain("not on PATH")
    expect(choice.agents[1].ready).toBe(true)
    expect(choice.selected).toBe("pi")
    expect(choice.chosen).toBe("pi")
    expect(choice.exists).toBe(true)
  })

  it("falls back to any when the project has no policy", () => {
    const choice = agentChoice({ agents: [], exists: false, reason: "span.yaml is not there yet." })
    expect(choice.selected).toBe("any")
    expect(choice.exists).toBe(false)
    expect(choice.reason).toContain("span.yaml")
  })

  it("drops a row without an id", () => {
    const choice = agentChoice({ agents: [{ label: "Ghost" }] })
    expect(choice.agents).toHaveLength(0)
  })
})
