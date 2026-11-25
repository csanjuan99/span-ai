import { cleanup, render, screen } from "@testing-library/react"
import { afterEach, describe, expect, it } from "vitest"

import { FinishedRecord } from "@/components/finished-record/FinishedRecord"
import type { Risk } from "@/lib/report"

afterEach(cleanup)

const risk: Risk = {
  id: "env",
  level: "high",
  origin: "detected",
  title: "Touches an environment file",
  detail: "",
  files: [".env"],
}

describe("FinishedRecord", () => {
  it("counts the risks noted and names each with its level", () => {
    render(
      <FinishedRecord risks={[risk]} details={{ agents: [], time: null }} />
    )
    const region = screen.getByRole("region", { name: "Risks noted" })
    expect(region.textContent).toContain("Risks noted · 1")
    expect(region.textContent).toContain("Touches an environment file")
    expect(region.textContent).toContain("high")
  })

  it("gives the agents and the time of the run", () => {
    render(
      <FinishedRecord
        risks={[]}
        details={{
          agents: ["claude-code", "codex"],
          time: { started: "14:08", took: "1h 33m" },
        }}
      />
    )
    const region = screen.getByRole("region", { name: "Run details" })
    expect(region.textContent).toContain("claude-code, codex")
    expect(region.textContent).toContain("1h 33m, started 14:08")
    expect(screen.queryByRole("region", { name: "Risks noted" })).toBeNull()
  })

  it("draws nothing when there is neither a risk nor a detail", () => {
    const { container } = render(
      <FinishedRecord risks={[]} details={{ agents: [], time: null }} />
    )
    expect(container.textContent).toBe("")
  })
})
