import { cleanup, render, screen } from "@testing-library/react"
import { afterEach, describe, expect, it } from "vitest"

import { LedgerSummary } from "@/components/ledger/LedgerSummary"
import type { Statement } from "@/lib/types"

afterEach(cleanup)

function statement(
  stage: string,
  reason: string,
  verdict = "advance"
): Statement {
  return { stage, iteration: 1, verdict, reason, commands: [] }
}

describe("LedgerSummary", () => {
  it("shows one line per stage with its mark", () => {
    render(
      <LedgerSummary
        statements={[
          statement("build", "7 files."),
          statement("prove", "tests failed", "hold"),
        ]}
      />
    )
    const items = screen.getAllByRole("listitem")
    expect(items).toHaveLength(2)
    expect(items[0].textContent).toContain("build")
    expect(items[0].textContent).toContain("+")
    expect(items[1].textContent).toContain("!")
  })

  it("says so when nothing is sealed", () => {
    render(<LedgerSummary statements={[]} />)
    expect(screen.getByText(/No record yet/)).toBeTruthy()
  })
})
