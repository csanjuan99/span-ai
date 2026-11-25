import { describe, expect, it } from "vitest"

import { ledgerRows } from "@/lib/ledger-rows"
import type { Statement } from "@/lib/types"

function statement(over: Partial<Statement> & { stage: string }): Statement {
  return { iteration: 1, verdict: "advance", reason: "", commands: [], ...over }
}

describe("ledgerRows", () => {
  it("keeps one row per stage: the latest sealed", () => {
    const rows = ledgerRows([
      statement({
        stage: "build",
        reason: "first try",
        time: "2026-11-24T14:00:00Z",
      }),
      statement({ stage: "prove", reason: "fast passed" }),
      statement({ stage: "build", reason: "second try" }),
    ])
    expect(rows.map((row) => [row.stage, row.line])).toEqual([
      ["build", "second try"],
      ["prove", "fast passed"],
    ])
  })

  it("says how many tests ran when the checks counted them", () => {
    const [row] = ledgerRows([
      statement({
        stage: "prove",
        reason: "focused passed",
        commands: [
          {
            command: "npm test",
            exitCode: 0,
            tests: { passed: 40, failed: 0, skipped: 0 },
          },
          {
            command: "npm run e2e",
            exitCode: 0,
            tests: { passed: 3, failed: 2, skipped: 0 },
          },
        ],
      }),
    ])
    expect(row.line).toBe("43 tests passed · 2 failed")
  })

  it("reads a reason as one clean line", () => {
    const [row] = ledgerRows([
      statement({
        stage: "inspect",
        reason:
          "\u001b[0;36mNo blocking findings.\u001b[0m Second sentence that is long.",
      }),
    ])
    expect(row.line).toBe("No blocking findings.")
  })

  it("cuts a long sentence and marks the cut", () => {
    const [row] = ledgerRows([
      statement({ stage: "frame", reason: "word ".repeat(60).trim() }),
    ])
    expect(row.line.length).toBeLessThanOrEqual(121)
    expect(row.line.endsWith("…")).toBe(true)
  })

  it("marks holds and stops", () => {
    const rows = ledgerRows([
      statement({ stage: "build", verdict: "hold", reason: "waiting" }),
      statement({ stage: "prove", verdict: "stop", reason: "no progress" }),
    ])
    expect(rows.map((row) => row.tone)).toEqual(["hold", "stop"])
  })

  it("folds the quiet stages into one row when they all just passed", () => {
    const rows = ledgerRows([
      statement({ stage: "intake", reason: "brief read" }),
      statement({ stage: "build", reason: "7 files" }),
      statement({ stage: "promote", reason: "ok" }),
      statement({ stage: "note", reason: "ok" }),
    ])
    expect(rows.map((row) => row.stage)).toEqual(["build", "3 others"])
    expect(rows[1].line).toBe("intake, promote and note sealed.")
  })

  it("keeps a quiet stage visible when it did not simply pass", () => {
    const rows = ledgerRows([
      statement({ stage: "intake", reason: "brief read" }),
      statement({
        stage: "promote",
        verdict: "hold",
        reason: "deploy needs approval",
      }),
    ])
    expect(rows.map((row) => row.stage)).toEqual(["intake", "promote"])
  })
})
