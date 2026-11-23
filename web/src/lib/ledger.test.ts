import { describe, expect, it } from "vitest"

import { changeLabel, failed, rounds, shortHash, toneOf } from "@/lib/ledger"
import type { Statement } from "@/lib/types"

function item(stage: string, iteration: number): Statement {
  return { stage, iteration, verdict: "advance", reason: "", commands: [] }
}

describe("ledger helpers", () => {
  it("reads a verdict as pass, hold, or stop", () => {
    expect(toneOf("stop")).toBe("stop")
    expect(toneOf("hold")).toBe("hold")
    expect(toneOf("advance")).toBe("pass")
  })

  it("shortens a hash and drops a sha256 prefix", () => {
    expect(shortHash("f2f2080984371ad259b88ce5a89d6149822dacb6")).toBe("f2f2080")
    expect(shortHash("sha256:1992c81fba269f")).toBe("1992c81")
  })

  it("names a pull request, a merge request, or any other link", () => {
    expect(changeLabel("https://github.com/o/r/pull/8")).toBe("PR #8")
    expect(changeLabel("https://gitlab.com/o/r/-/merge_requests/3")).toBe("MR !3")
    expect(changeLabel("https://example.com/change/1")).toBe("example.com/change/1")
  })

  it("calls a command failed on a non-zero exit or a recorded error", () => {
    expect(failed({ command: "x", exitCode: 0 })).toBe(false)
    expect(failed({ command: "x", exitCode: 1 })).toBe(true)
    expect(failed({ command: "x", exitCode: 0, error: "timed out" })).toBe(true)
  })

  it("starts a new round when a stage repeats, even at the same iteration", () => {
    const out = rounds([item("build", 1), item("prove", 1), item("inspect", 1), item("build", 1), item("prove", 1)])
    expect(out.map((group) => [group.number, group.items.map((entry) => entry.stage)])).toEqual([
      [1, ["build", "prove", "inspect"]],
      [2, ["build", "prove"]],
    ])
  })
})
