import { describe, expect, it } from "vitest"

import type { DiffFile } from "@/lib/diff"
import { proofOf } from "@/lib/proof"
import type { Report } from "@/lib/report"

function file(path: string, status: DiffFile["status"], added = 1, removed = 0): DiffFile {
  return { path, oldPath: path, status, binary: false, added, removed, hunks: [] }
}

function report(over: Partial<Report> = {}): Report {
  return {
    verdict: { level: "low", headline: "Small change with passing checks.", depth: "A quick read is enough." },
    risks: [],
    groups: [],
    totals: { files: 3, added: 184, removed: 32 },
    evidence: {
      commands: [
        { stage: "prove", command: "go build ./...", exitCode: 0, ok: true },
        { stage: "prove", command: "go test ./...", exitCode: 0, ok: true },
      ],
      inspect: { verdict: "advance", reason: "" },
      sentBack: 0,
      repairs: 0,
      flaky: false,
    },
    gaps: [],
    ...over,
  }
}

describe("proofOf", () => {
  it("reads the verdict, the size and the checks from the report", () => {
    const files = [file("a.go", "added"), file("b.go", "modified"), file("c.go", "modified")]
    const proof = proofOf(report(), files)
    expect(proof.verdict).toEqual({ level: "low", headline: "Small change with passing checks." })
    expect(proof.diff).toEqual({ files: 3, added: 184, removed: 32, kinds: "A 1 · M 2" })
    expect(proof.checks).toEqual({ passed: 2, total: 2, failing: "", inspect: "passed" })
  })

  it("names the first check that failed, so the cell says what to look at", () => {
    const failing = report({
      evidence: {
        commands: [
          { stage: "prove", command: "go build ./...", exitCode: 0, ok: true },
          { stage: "prove", command: "go test ./...", exitCode: 1, ok: false },
        ],
        inspect: null,
        sentBack: 0,
        repairs: 0,
        flaky: false,
      },
    })
    const proof = proofOf(failing, [])
    expect(proof.checks).toEqual({ passed: 1, total: 2, failing: "go test ./... · exit 1", inspect: "not yet" })
  })

  it("says inspect blocked the change when it did not advance", () => {
    const blocked = report({
      evidence: { commands: [], inspect: { verdict: "hold", reason: "needs a test" }, sentBack: 1, repairs: 0, flaky: false },
    })
    expect(proofOf(blocked, []).checks.inspect).toBe("blocked")
  })

  it("lists every kind of change present, in a fixed order, and skips the absent ones", () => {
    const files = [file("d.go", "deleted"), file("r.go", "renamed"), file("a.go", "added"), file("m.go", "modified")]
    expect(proofOf(report(), files).diff.kinds).toBe("A 1 · M 1 · R 1 · D 1")
    expect(proofOf(report(), []).diff.kinds).toBe("")
  })
})
