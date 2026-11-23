import { cleanup, render, screen } from "@testing-library/react"
import { afterEach, describe, expect, it } from "vitest"

import { FailureCard } from "@/components/failure/FailureCard"

const failure = {
  stage: "prove",
  iteration: 2,
  cap: 3,
  reason: "go test ./... exited 1",
  inconclusive: false,
  command: {
    command: "go test ./...",
    exitCode: 1,
    output: "FAIL src/cart.test.ts > totals\nexpected 2 to be 3",
    logPath: "iter/2/logs/prove-1.log",
  },
}

afterEach(cleanup)

describe("FailureCard", () => {
  it("shows the command, its output, the attempt, and where the full log is", () => {
    render(<FailureCard failure={failure} />)
    expect(screen.getByText("go test ./... · exit 1")).toBeTruthy()
    expect(screen.getByText("attempt 2 of 3")).toBeTruthy()
    expect(screen.getByLabelText("Command output").textContent).toContain("expected 2 to be 3")
    expect(screen.getByText("iter/2/logs/prove-1.log")).toBeTruthy()
    expect(screen.queryByText(/not attributable/)).toBeNull()
  })

  it("says a flipped result is not attributable to the change", () => {
    render(<FailureCard failure={{ ...failure, inconclusive: true }} />)
    expect(screen.getByText(/not attributable to the change/)).toBeTruthy()
  })

  it("hides the counter when no cap was recorded", () => {
    render(<FailureCard failure={{ ...failure, cap: 0 }} />)
    expect(screen.queryByText(/attempt/)).toBeNull()
  })
})
