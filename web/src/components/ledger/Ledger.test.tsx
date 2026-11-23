import { cleanup, render, screen } from "@testing-library/react"
import { afterEach, describe, expect, it } from "vitest"

import { Ledger } from "@/components/ledger/Ledger"
import type { Statement } from "@/lib/types"

const statements: Statement[] = [
  {
    stage: "prove",
    iteration: 1,
    verdict: "stop",
    reason: "fast failed",
    gitSHA: "f2f2080984371ad259b88ce5a89d6149822dacb6",
    commands: [{ command: "go test ./...", exitCode: 1, output: "FAIL pkg" }],
  },
  {
    stage: "open",
    iteration: 1,
    verdict: "advance",
    reason: "opened",
    changeURL: "https://github.com/o/r/pull/8",
    commands: [],
  },
]

describe("Ledger", () => {
  afterEach(cleanup)

  it("starts a new round when a stage repeats and shortens the commit", () => {
    const again: Statement = { stage: "prove", iteration: 1, verdict: "advance", reason: "passed", commands: [] }
    render(<Ledger statements={[...statements, again]} />)
    expect(screen.getByRole("region", { name: "Round 1" })).toBeTruthy()
    expect(screen.getByRole("region", { name: "Round 2" })).toBeTruthy()
    expect(screen.getByText("f2f2080")).toBeTruthy()
  })

  it("links the change by its number", () => {
    render(<Ledger statements={statements} />)
    const link = screen.getByRole("link", { name: "PR #8" })
    expect(link.getAttribute("href")).toBe("https://github.com/o/r/pull/8")
  })

  it("shows what a failing command printed", () => {
    render(<Ledger statements={statements} />)
    expect(screen.getByText("FAIL pkg")).toBeTruthy()
    expect(screen.getByText("exit 1")).toBeTruthy()
  })

  it("keeps the empty message", () => {
    render(<Ledger statements={[]} />)
    expect(screen.getByText(/No record yet/)).toBeTruthy()
  })
})
