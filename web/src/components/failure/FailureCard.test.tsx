import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react"
import { afterEach, describe, expect, it, vi } from "vitest"

const getRunLog = vi.fn()
vi.mock("@/lib/api", async (original) => ({
  ...(await original<typeof import("@/lib/api")>()),
  getRunLog: (id: string, path: string) => getRunLog(id, path),
}))

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
  it("shows the reason, the command with its exit code, its output, and where the full log is", () => {
    render(<FailureCard id="span-1" failure={failure} />)
    expect(
      screen.getByRole("heading", { name: "go test ./... exited 1" })
    ).toBeTruthy()
    expect(screen.getByText("go test ./...")).toBeTruthy()
    expect(screen.getByText("exit 1")).toBeTruthy()
    expect(screen.getByLabelText("Command output").textContent).toContain(
      "expected 2 to be 3"
    )
    expect(screen.getByRole("button", { name: "Open full log" })).toBeTruthy()
    expect(screen.queryByText(/not attributable/)).toBeNull()
  })

  it("says a flipped result is not attributable to the change", () => {
    render(
      <FailureCard id="span-1" failure={{ ...failure, inconclusive: true }} />
    )
    expect(screen.getByText(/not attributable to the change/)).toBeTruthy()
  })

  it("draws no command block when the failure names no command", () => {
    render(<FailureCard id="span-1" failure={{ ...failure, command: null }} />)
    expect(screen.queryByText("Failing output")).toBeNull()
  })
})

describe("FailureCard evidence", () => {
  const long = {
    ...failure,
    command: {
      ...failure.command!,
      lines: 142,
      output:
        "--- FAIL: TestRetryBackoff (0.02s)\n    retry_test.go:58: delay 2s, want 4s\nFAIL",
    },
  }

  it("says how much of the output is on screen", () => {
    render(<FailureCard id="span-1" failure={long} />)
    expect(screen.getByText("3 of 142 lines")).toBeTruthy()
  })

  it("says nothing about the count when the output is all there is", () => {
    render(<FailureCard id="span-1" failure={failure} />)
    expect(screen.queryByText(/of \d+ lines/)).toBeNull()
  })

  it("draws the failing test's name apart from the rest of its line", () => {
    render(<FailureCard id="span-1" failure={long} />)
    expect(screen.getByText("TestRetryBackoff").className).toContain(
      "text-primary"
    )
  })

  it("opens the full log from the server, in place", async () => {
    getRunLog.mockResolvedValue({
      path: "iter/2/logs/prove-1.log",
      text: "one\ntwo\n",
      lines: 2,
      truncated: false,
    })
    render(<FailureCard id="span-1" failure={long} />)
    fireEvent.click(screen.getByRole("button", { name: "Open full log" }))
    await waitFor(() =>
      expect(screen.getByLabelText("Full log").textContent).toContain("two")
    )
    expect(getRunLog).toHaveBeenCalledWith("span-1", "iter/2/logs/prove-1.log")
  })

  it("says so when the log cannot be read", async () => {
    getRunLog.mockRejectedValue(new Error("the log is not available"))
    render(<FailureCard id="span-1" failure={long} />)
    fireEvent.click(screen.getByRole("button", { name: "Open full log" }))
    expect(await screen.findByText("the log is not available")).toBeTruthy()
  })

  it("offers no full log when the command kept none", () => {
    const { logPath: _omit, ...bare } = long.command
    render(<FailureCard id="span-1" failure={{ ...long, command: bare }} />)
    expect(screen.queryByRole("button", { name: "Open full log" })).toBeNull()
  })
})
