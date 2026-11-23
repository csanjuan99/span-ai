import { cleanup, render, screen } from "@testing-library/react"
import { afterEach, describe, expect, it } from "vitest"

import { Implementation, type RunFacts } from "@/components/implementation/Implementation"
import { parseFeature } from "@/lib/featuredoc"
import type { Cursor } from "@/lib/types"

const cursor: Cursor = {
  id: "r1", stage: "inspect", iteration: 1, hold: false, stop: false, done: false, reason: "",
  gitSHA: "f2f2080984371ad259b88ce5a89d6149822dacb6", changeURL: "", branch: "feat/x", base: "dev",
  workspace: "/w", issue: "", questions: [],
}

const run: RunFacts = { cursor, statements: [], agents: {}, feature: null }

const patch = ["diff --git a/pay.go b/pay.go", "--- a/pay.go", "+++ b/pay.go", "@@ -1,1 +1,2 @@", " a", "+charge"].join("\n")

describe("Implementation", () => {
  afterEach(cleanup)

  it("shows the diff with its line and the totals", () => {
    render(<Implementation stat=" pay.go | 1 +" patch={patch} truncated={false} transcript={["wrote the charge"]} run={run} />)
    expect(screen.getAllByText("pay.go").length).toBeGreaterThan(0)
    expect(screen.getByText("charge")).toBeTruthy()
    expect(screen.getAllByText("+1").length).toBeGreaterThan(0)
  })

  it("lists a file the cut patch did not reach, with its counts", () => {
    render(<Implementation stat={" pay.go | 1 +\n docs.md | 4 ++--"} patch={patch} truncated transcript={[]} run={run} />)
    expect(screen.getByText("docs.md")).toBeTruthy()
    expect(screen.getByText(/The patch was cut at 256 KiB/)).toBeTruthy()
  })

  it("says so when there is nothing yet", () => {
    render(<Implementation stat="" patch="" truncated={false} transcript={[]} run={run} />)
    expect(screen.getByText("No commit on this run yet.")).toBeTruthy()
    expect(screen.getByText("Nothing to assess yet")).toBeTruthy()
  })

  it("leads with the verdict and names a risk with the file it points at", () => {
    const risky = ["diff --git a/.env b/.env", "@@ -0,0 +1,1 @@", "+X=1"].join("\n")
    render(<Implementation stat=" .env | 1 +" patch={risky} truncated={false} transcript={[]} run={run} />)
    expect(screen.getByText("high risk")).toBeTruthy()
    expect(screen.getByText("Touches an environment file")).toBeTruthy()
    expect(screen.getAllByText(".env").length).toBeGreaterThan(0)
  })

  it("shows the agent's decisions as assumed, and its own risks as reported", () => {
    const feature = parseFeature("# T\n\n## Log\nDecision: keep cents because floats drift\nRisk: double charge on a repeated webhook")
    render(<Implementation stat=" pay.go | 1 +" patch={patch} truncated={false} transcript={[]} run={{ ...run, feature }} />)
    expect(screen.getByText("keep cents")).toBeTruthy()
    expect(screen.getByText("floats drift")).toBeTruthy()
    expect(screen.getByText(/assumed by the agent/)).toBeTruthy()
    expect(screen.getByText("double charge on a repeated webhook")).toBeTruthy()
  })

  it("keeps the raw transcript reachable but closed", () => {
    render(<Implementation stat="" patch="" truncated={false} transcript={["wrote the charge"]} run={run} />)
    expect(screen.getByText("1 lines")).toBeTruthy()
    expect(screen.getByText("wrote the charge")).toBeTruthy()
  })

  it("tells the reader what the agent did not record instead of staying silent", () => {
    render(<Implementation stat=" pay.go | 1 +" patch={patch} truncated={false} transcript={[]} run={run} />)
    expect(screen.getByText(/The agent recorded no decision/)).toBeTruthy()
    expect(screen.getByText(/no feature document log/)).toBeTruthy()
  })
})
