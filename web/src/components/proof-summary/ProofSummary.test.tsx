import { cleanup, render, screen } from "@testing-library/react"
import { afterEach, describe, expect, it } from "vitest"

import { ProofSummary } from "@/components/proof-summary/ProofSummary"
import type { Proof } from "@/lib/proof"

afterEach(cleanup)

const proof: Proof = {
  verdict: { level: "medium", headline: "A larger change than usual." },
  diff: { files: 7, added: 184, removed: 32, kinds: "A 2 · M 4 · R 1" },
  checks: {
    passed: 1,
    total: 2,
    failing: "go test ./... · exit 1",
    inspect: "not yet",
    tests: null,
  },
}

describe("ProofSummary", () => {
  it("shows risk, size and checks, each with its own words", () => {
    render(<ProofSummary proof={proof} />)
    const region = screen.getByRole("region", { name: "Proof of work" })
    expect(region.textContent).toContain("medium risk")
    expect(region.textContent).toContain("A larger change than usual.")
    expect(region.textContent).toContain("7 files")
    expect(region.textContent).toContain("+184")
    expect(region.textContent).toContain("−32")
    expect(region.textContent).toContain("A 2 · M 4 · R 1")
    expect(region.textContent).toContain("1/2 passed")
  })

  it("names the failing check instead of a bare count", () => {
    render(<ProofSummary proof={proof} />)
    expect(
      screen.getByRole("region", { name: "Proof of work" }).textContent
    ).toContain("go test ./... · exit 1")
  })

  it("says when no check ran and what inspect did, and uses the singular for one file", () => {
    render(
      <ProofSummary
        proof={{
          ...proof,
          diff: { files: 1, added: 1, removed: 0, kinds: "A 1" },
          checks: {
            passed: 0,
            total: 0,
            failing: "",
            inspect: "passed",
            tests: null,
          },
        }}
      />
    )
    const text = screen.getByRole("region", {
      name: "Proof of work",
    }).textContent
    expect(text).toContain("none ran")
    expect(text).toContain("inspect passed")
    expect(text).toContain("1 file ")
  })

  it("shows the tests the checks counted instead of only the commands", () => {
    render(
      <ProofSummary
        proof={{
          ...proof,
          checks: {
            passed: 2,
            total: 3,
            failing: "",
            inspect: "passed",
            tests: { passed: 41, failed: 2 },
          },
        }}
      />
    )
    const text = screen.getByRole("region", {
      name: "Proof of work",
    }).textContent
    expect(text).toContain("Tests")
    expect(text).toContain("41 passed · 2 failed")
    expect(text).toContain("2 of 3 checks passed · inspect passed")
  })
})

describe("ProofSummary at a position in the run", () => {
  it("says the run is held at the stage instead of rating its risk", () => {
    render(<ProofSummary proof={proof} run={{ kind: "held", stage: "prove", sealed: 3 }} />)
    const text = screen.getByRole("region", { name: "Proof of work" }).textContent
    expect(text).toContain("Held at prove")
    expect(text).toContain("1 of 2 checks not green")
    expect(text).not.toContain("medium risk")
  })

  it("says there is no code and no tests yet before build", () => {
    const empty = { ...proof, diff: { files: 0, added: 0, removed: 0, kinds: "" }, checks: { ...proof.checks, passed: 0, total: 0, failing: "" } }
    render(<ProofSummary proof={empty} run={{ kind: "paused", stage: "frame", sealed: 1 }} />)
    const text = screen.getByRole("region", { name: "Proof of work" }).textContent
    expect(text).toContain("Frame paused")
    expect(text).toContain("No code yet")
    expect(text).toContain("frame does not touch the repo")
    expect(text).toContain("Not run yet")
    expect(text).toContain("prove comes after frame")
  })
})
