import { describe, expect, it } from "vitest"

import { highlight } from "@/lib/output-lines"

describe("highlight", () => {
  it("marks a failing Go test, its name and its timing apart", () => {
    expect(highlight("--- FAIL: TestRetryBackoff (0.02s)")).toEqual([
      { text: "--- FAIL:", tone: "fail" },
      { text: " ", tone: "plain" },
      { text: "TestRetryBackoff", tone: "name" },
      { text: " ", tone: "plain" },
      { text: "(0.02s)", tone: "muted" },
    ])
  })

  it("dims the assertion detail under a failing test", () => {
    expect(highlight("    retry_test.go:58: delay 2s, want 4s")).toEqual([
      { text: "    retry_test.go:58: delay 2s, want 4s", tone: "muted" },
    ])
  })

  it("marks the closing FAIL of a package and keeps the rest plain", () => {
    expect(highlight("FAIL\tacme/webhook\t0.412s")).toEqual([
      { text: "FAIL", tone: "fail" },
      { text: "\tacme/webhook\t0.412s", tone: "plain" },
    ])
  })

  it("names failing tests from other runners", () => {
    expect(highlight("FAIL src/retry.test.ts > backs off")).toEqual([
      { text: "FAIL", tone: "fail" },
      { text: " src/retry.test.ts > backs off", tone: "plain" },
    ])
    expect(highlight("  ✗ TestJitter")[0]?.tone).toBe("plain")
  })

  it("leaves a plain line alone and an empty one empty", () => {
    expect(highlight("ok  acme/webhook 0.3s")).toEqual([{ text: "ok  acme/webhook 0.3s", tone: "plain" }])
    expect(highlight("")).toEqual([])
  })
})
