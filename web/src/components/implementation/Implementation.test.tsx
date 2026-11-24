import { cleanup, render, screen } from "@testing-library/react"
import { afterEach, describe, expect, it } from "vitest"

import { Implementation, type RunFacts } from "@/components/implementation/Implementation"
import { changeOf } from "@/lib/change"
import { parseFeature } from "@/lib/featuredoc"
import type { Cursor } from "@/lib/types"

const cursor: Cursor = {
  id: "r1", stage: "inspect", iteration: 1, hold: false, stop: false, done: false, reason: "",
  gitSHA: "f2f2080984371ad259b88ce5a89d6149822dacb6", changeURL: "", branch: "feat/x", base: "dev",
  workspace: "/w", issue: "", questions: [],
}

const run: RunFacts = { cursor, statements: [], agents: {}, feature: null, journal: { events: [], intact: true, brokenAt: 0 } }

const patch = ["diff --git a/pay.go b/pay.go", "--- a/pay.go", "+++ b/pay.go", "@@ -1,1 +1,2 @@", " a", "+charge"].join("\n")

/** Renders the tab the way the page does: the change is built once and handed in. */
function view(over: { stat?: string; patch?: string; truncated?: boolean; transcript?: string[]; run?: RunFacts } = {}) {
  const facts = over.run ?? run
  const truncated = over.truncated ?? false
  const change = changeOf({
    stat: over.stat ?? "",
    patch: over.patch ?? "",
    truncated,
    statements: facts.statements,
    cursor: facts.cursor,
    feature: facts.feature,
  })
  return <Implementation change={change} truncated={truncated} transcript={over.transcript ?? []} run={facts} />
}

describe("Implementation", () => {
  afterEach(cleanup)

  it("shows the diff with its line and the totals", () => {
    render(view({ stat: " pay.go | 1 +", patch, transcript: ["wrote the charge"] }))
    expect(screen.getAllByText("pay.go").length).toBeGreaterThan(0)
    expect(screen.getByText("charge")).toBeTruthy()
    expect(screen.getAllByText("+1").length).toBeGreaterThan(0)
  })

  it("lists a file the cut patch did not reach, with its counts", () => {
    render(view({ stat: " pay.go | 1 +\n docs.md | 4 ++--", patch, truncated: true }))
    expect(screen.getByText("docs.md")).toBeTruthy()
    expect(screen.getByText(/The patch was cut at 256 KiB/)).toBeTruthy()
  })

  it("says so when there is nothing yet", () => {
    render(view())
    expect(screen.getByText("No commit on this run yet.")).toBeTruthy()
    expect(screen.getByText("Nothing to assess yet")).toBeTruthy()
  })

  it("leads with the verdict and names a risk with the file it points at", () => {
    const risky = ["diff --git a/.env b/.env", "@@ -0,0 +1,1 @@", "+X=1"].join("\n")
    render(view({ stat: " .env | 1 +", patch: risky }))
    expect(screen.getByText("high risk")).toBeTruthy()
    expect(screen.getByText("Touches an environment file")).toBeTruthy()
    expect(screen.getAllByText(".env").length).toBeGreaterThan(0)
  })

  it("shows the agent's decisions as assumed, and its own risks as reported", () => {
    const feature = parseFeature("# T\n\n## Log\nDecision: keep cents because floats drift\nRisk: double charge on a repeated webhook")
    render(view({ stat: " pay.go | 1 +", patch, run: { ...run, feature } }))
    expect(screen.getByText("keep cents")).toBeTruthy()
    expect(screen.getByText("floats drift")).toBeTruthy()
    expect(screen.getByText(/assumed by the agent/)).toBeTruthy()
    expect(screen.getByText("double charge on a repeated webhook")).toBeTruthy()
  })

  it("keeps the raw transcript reachable but closed", () => {
    render(view({ transcript: ["wrote the charge"] }))
    expect(screen.getByText("1 lines")).toBeTruthy()
    expect(screen.getByText("wrote the charge")).toBeTruthy()
  })

  it("tells the reader what the agent did not record instead of staying silent", () => {
    render(view({ stat: " pay.go | 1 +", patch }))
    expect(screen.getByText(/The agent recorded no decision/)).toBeTruthy()
    expect(screen.getByText(/no feature document log/)).toBeTruthy()
  })
})

describe("Implementation audit trail", () => {
  afterEach(cleanup)

  it("lists what a person did and warns when the journal was altered", () => {
    const journal = {
      events: [{ seq: 1, time: "2026-11-23T09:03:11Z", actor: "person", type: "rewind", detail: { stage: "build" } }],
      intact: false,
      brokenAt: 2,
    }
    render(view({ run: { ...run, journal } }))
    expect(screen.getByText("rewind")).toBeTruthy()
    expect(screen.getByText("stage build")).toBeTruthy()
    expect(screen.getByText("journal altered at event 2")).toBeTruthy()
  })
})
