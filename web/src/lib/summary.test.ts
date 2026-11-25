import { describe, expect, it } from "vitest"

import { summaryOf, toMarkdown } from "@/lib/summary"
import type { Cursor, Statement } from "@/lib/types"

const cursor: Cursor = {
  id: "run-1",
  stage: "note",
  iteration: 1,
  hold: false,
  stop: false,
  done: true,
  reason: "",
  gitSHA: "c91f3ab0000000",
  changeURL: "https://github.com/o/r/pull/8",
  branch: "feat/SPAN-482",
  base: "main",
  workspace: "/w/api",
  issue: "SPAN-482",
  questions: [],
}

function sealed(stage: string, over: Partial<Statement> = {}): Statement {
  return {
    stage,
    iteration: 1,
    verdict: "advance",
    reason: "",
    commands: [],
    ...over,
  }
}

const stat =
  " src/a.go | 4 ++--\n src/new.go | 3 +++\n 2 files changed, 5 insertions(+), 2 deletions(-)\n"
const patch = [
  "diff --git a/src/a.go b/src/a.go",
  "--- a/src/a.go",
  "+++ b/src/a.go",
  "@@ -1,3 +1,3 @@",
  "-old",
  "-old2",
  "+new",
  "+new2",
  "diff --git a/src/new.go b/src/new.go",
  "new file mode 100644",
  "--- /dev/null",
  "+++ b/src/new.go",
  "@@ -0,0 +1,3 @@",
  "+a",
  "+b",
  "+c",
].join("\n")

const statements = [
  sealed("prove", {
    commands: [
      {
        command: "go test ./...",
        exitCode: 0,
        tests: { passed: 43, failed: 0, skipped: 0 },
      },
    ],
  }),
  sealed("inspect"),
  sealed("merge"),
]

function summary(over: Partial<Parameters<typeof summaryOf>[0]> = {}) {
  return summaryOf({
    cursor,
    statements,
    implementation: { stat, patch, truncated: false, commits: 2 },
    label: "SPAN-482-run-1-api",
    ...over,
  })
}

describe("summaryOf", () => {
  it("reads files with their letter and the totals", () => {
    const got = summary()
    expect(got.files.map((file) => [file.letter, file.path])).toEqual([
      ["M", "src/a.go"],
      ["A", "src/new.go"],
    ])
    expect(got.totals.files).toBe(2)
    expect(got.assessed).toBe(true)
  })

  it("counts the tests the checks printed", () => {
    expect(summary().tests).toEqual({ passed: 43, failed: 0 })
  })

  it("claims the merge only when the ledger sealed it", () => {
    expect(summary().outcome).toBe("Merged into main as c91f3ab")
    expect(
      summary({ statements: statements.slice(0, 2) }).outcome
    ).not.toContain("Merged")
  })

  it("links the change and names the merge commit", () => {
    const got = summary()
    expect(got.links).toEqual([
      { label: "PR #8", href: "https://github.com/o/r/pull/8" },
    ])
    expect(got.sha).toBe("c91f3ab")
  })

  it("says not assessed when there is no diff to read", () => {
    const got = summary({
      implementation: { stat: "", patch: "", truncated: false, commits: 0 },
    })
    expect(got.assessed).toBe(false)
    expect(got.files).toEqual([])
  })

  it("is not assessed while the diff has not loaded", () => {
    expect(summary({ implementation: null }).assessed).toBe(false)
  })
})

describe("toMarkdown", () => {
  it("writes the title, outcome, files, tests and links", () => {
    const text = toMarkdown(summary())
    expect(text).toContain("# SPAN-482-run-1-api")
    expect(text).toContain("Merged into main as c91f3ab")
    expect(text).toContain("- M `src/a.go` +2 −2")
    expect(text).toContain("- A `src/new.go` +3 −0")
    expect(text).toContain("43 passed")
    expect(text).toContain("[PR #8](https://github.com/o/r/pull/8)")
  })

  it("does not invent a diff that was not read", () => {
    const text = toMarkdown(summary({ implementation: null }))
    expect(text).toContain("Not assessed")
    expect(text).not.toContain("## Files")
  })
})
