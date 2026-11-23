import { cleanup, render, screen } from "@testing-library/react"
import { afterEach, describe, expect, it } from "vitest"

import { Implementation } from "@/components/implementation/Implementation"

const patch = ["diff --git a/pay.go b/pay.go", "--- a/pay.go", "+++ b/pay.go", "@@ -1,1 +1,2 @@", " a", "+charge"].join("\n")

describe("Implementation", () => {
  afterEach(cleanup)

  it("shows the diff with its line and the totals", () => {
    render(<Implementation stat=" pay.go | 1 +" patch={patch} truncated={false} transcript={["wrote the charge"]} />)
    expect(screen.getByText("pay.go")).toBeTruthy()
    expect(screen.getByText("charge")).toBeTruthy()
    expect(screen.getAllByText("+1").length).toBeGreaterThan(0)
    expect(screen.getByText("wrote the charge")).toBeTruthy()
  })

  it("lists a file the cut patch did not reach, with its counts", () => {
    render(<Implementation stat={" pay.go | 1 +\n docs.md | 4 ++--"} patch={patch} truncated transcript={[]} />)
    expect(screen.getByText("docs.md")).toBeTruthy()
    expect(screen.getByText(/The patch was cut at 256 KiB/)).toBeTruthy()
  })

  it("says so when there is nothing yet", () => {
    render(<Implementation stat="" patch="" truncated={false} transcript={[]} />)
    expect(screen.getByText("No commit on this run yet.")).toBeTruthy()
    expect(screen.getByText("The agent has not written yet.")).toBeTruthy()
  })
})
