import { describe, expect, it } from "vitest"

import { parseDiff } from "@/lib/diff"
import { purposeShares, statusCounts, testsAdded } from "@/lib/overview"

const patch = [
  "diff --git a/retry_test.go b/retry_test.go",
  "new file mode 100644",
  "--- /dev/null",
  "+++ b/retry_test.go",
  "@@ -0,0 +1,4 @@",
  "+func TestBackoff(t *testing.T) {",
  "+}",
  "+func TestLimit(t *testing.T) {",
  "+}",
  "diff --git a/retry.go b/retry.go",
  "--- a/retry.go",
  "+++ b/retry.go",
  "@@ -1,1 +1,2 @@",
  " a",
  "+func TestNotATestFile() {",
].join("\n")

describe("overview helpers", () => {
  it("counts the tests a change adds, only in test files", () => {
    const files = parseDiff(patch)
    expect(testsAdded(files)).toEqual({ count: 2, files: ["retry_test.go"] })
  })

  it("counts files by status", () => {
    expect(statusCounts(parseDiff(patch))).toEqual({
      added: 1,
      modified: 1,
      renamed: 0,
      deleted: 0,
    })
  })

  it("shares the changed lines by family and lists every family", () => {
    const shares = purposeShares([
      { kind: "source", files: ["a.go"], added: 3, removed: 1 },
      { kind: "test", files: ["a_test.go"], added: 4, removed: 0 },
    ])
    expect(shares.map((s) => s.label)).toEqual([
      "Source",
      "Tests",
      "Docs",
      "Config",
      "Dependencies",
      "Build and deploy",
      "Generated",
    ])
    expect(shares[0]?.percent).toBe(50)
    expect(shares[1]?.percent).toBe(50)
    expect(shares[2]).toMatchObject({ files: 0, percent: 0 })
  })
})
