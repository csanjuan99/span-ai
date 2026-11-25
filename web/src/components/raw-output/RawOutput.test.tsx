import { cleanup, render, screen } from "@testing-library/react"
import { afterEach, describe, expect, it } from "vitest"

import { RawOutput } from "@/components/raw-output/RawOutput"
import type { Statement } from "@/lib/types"

afterEach(cleanup)

const statements: Statement[] = [
  {
    stage: "prove",
    iteration: 1,
    verdict: "advance",
    reason: "",
    commands: [{ command: "go test ./...", exitCode: 0 }],
  },
]

describe("RawOutput", () => {
  it("counts the transcript lines and the commands, closed until asked for", () => {
    render(
      <RawOutput
        transcript={["wrote the charge", "ran tests"]}
        statements={statements}
      />
    )
    expect(screen.getByText("2 lines")).toBeTruthy()
    expect(screen.getByText("1 command")).toBeTruthy()
    for (const entry of document.querySelectorAll("details")) {
      expect(entry.open).toBe(false)
    }
  })

  it("holds the lines of each entry with their numbers", () => {
    render(
      <RawOutput transcript={["wrote the charge"]} statements={statements} />
    )
    expect(screen.getByText("wrote the charge")).toBeTruthy()
    const log =
      screen.getByText("Command log").closest("details")?.textContent ?? ""
    expect(log).toContain("1prove  go test ./...  exit 0")
  })

  it("says so when nothing was written or run", () => {
    render(<RawOutput transcript={[]} statements={[]} />)
    expect(screen.getByText("The agent has not written yet.")).toBeTruthy()
    expect(screen.getByText("No command has run yet.")).toBeTruthy()
  })
})
