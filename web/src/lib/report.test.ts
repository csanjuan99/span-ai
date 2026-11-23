import { describe, expect, it } from "vitest"

import { parseDiff } from "@/lib/diff"
import { parseFeature } from "@/lib/featuredoc"
import { buildReport, kindOf } from "@/lib/report"
import type { Cursor, Statement } from "@/lib/types"

const cursor: Cursor = {
  id: "r1", stage: "inspect", iteration: 1, hold: false, stop: false, done: false, reason: "",
  gitSHA: "abc", changeURL: "", branch: "feat/x", base: "dev", workspace: "/w", issue: "", questions: [],
}

function patchFor(path: string, lines: string[], status = ""): string {
  return [`diff --git a/${path} b/${path}`, status, `@@ -0,0 +1,${lines.length} @@`, ...lines.map((line) => `+${line}`)].filter((part) => part !== "").join("\n")
}

function report(patch: string, statements: Statement[] = [], truncated = false) {
  return buildReport({ files: parseDiff(patch), truncated, statements, cursor, feature: null })
}

const ids = (patch: string) => report(patch).risks.map((item) => item.id)

describe("kindOf", () => {
  it("buckets paths by purpose", () => {
    expect(kindOf("pay/charge.go")).toBe("source")
    expect(kindOf("pay/charge_test.go")).toBe("test")
    expect(kindOf("web/src/a.test.tsx")).toBe("test")
    expect(kindOf("go.mod")).toBe("deps")
    expect(kindOf(".github/workflows/ci.yml")).toBe("ci")
    expect(kindOf("tests/__pycache__/x.cpython-314.pyc")).toBe("generated")
    expect(kindOf("README.md")).toBe("docs")
    expect(kindOf("span.yaml")).toBe("config")
  })
})

describe("buildReport risks", () => {
  it("flags generated files that were committed", () => {
    expect(ids(patchFor("tests/__pycache__/t.pyc", ["x"]))).toContain("generated")
  })

  it("flags source changed with no test, and not when a test changed too", () => {
    expect(ids(patchFor("pay.go", ["x"]))).toContain("untested")
    expect(ids(`${patchFor("pay.go", ["x"])}\n${patchFor("pay_test.go", ["x"])}`)).not.toContain("untested")
  })

  it("flags auth, migrations, ci and env as high", () => {
    for (const path of ["internal/auth/login.go", "db/migrations/001.sql", ".github/workflows/ci.yml", ".env.local"]) {
      const found = report(patchFor(path, ["x"])).risks.find((item) => item.level === "high")
      expect(found, path).toBeDefined()
    }
  })

  it("flags a deleted test file", () => {
    const patch = patchFor("pay_test.go", ["x"], "deleted file mode 100644")
    expect(ids(patch)).toContain("tests-deleted")
  })

  it("finds secrets and TODOs only on added lines", () => {
    const patch = patchFor("a.go", ['key := "AKIAABCDEFGHIJKLMNOP"', "// TODO later"])
    expect(ids(patch)).toEqual(expect.arrayContaining(["secret", "todo"]))
    expect(ids(patchFor("a.go", ["clean"]))).not.toContain("secret")
  })

  it("scores size by the published bands", () => {
    const big = patchFor("a.go", Array.from({ length: 450 }, (_, index) => `line ${index}`))
    expect(report(big).risks.find((item) => item.id === "size")?.level).toBe("high")
    const mid = patchFor("a.go", Array.from({ length: 250 }, (_, index) => `line ${index}`))
    expect(report(mid).risks.find((item) => item.id === "size")?.level).toBe("medium")
  })

  it("orders high before medium before low and sets the verdict from the worst", () => {
    const got = report(`${patchFor("a.go", ["// TODO"])}\n${patchFor("go.mod", ["x"])}\n${patchFor(".env", ["x"])}`)
    expect(got.risks.map((item) => item.level)).toEqual([...got.risks.map((item) => item.level)].sort((a, b) => ["high", "medium", "low"].indexOf(a) - ["high", "medium", "low"].indexOf(b)))
    expect(got.verdict.level).toBe("high")
  })

  it("says plainly that no signals is not proof", () => {
    const got = report(`${patchFor("a.go", ["x"])}\n${patchFor("a_test.go", ["x"])}`)
    expect(got.verdict.level).toBe("low")
    expect(got.verdict.depth).toMatch(/not proof/)
  })
})

describe("buildReport evidence and gaps", () => {
  const statements: Statement[] = [
    { stage: "prove", iteration: 1, verdict: "advance", reason: "", commands: [{ command: "go test ./...", exitCode: 0 }] },
    { stage: "inspect", iteration: 1, verdict: "stop", reason: "missing spec", commands: [] },
    { stage: "inspect", iteration: 1, verdict: "advance", reason: "", commands: [] },
  ]

  it("collects commands, the last inspect verdict and how often inspect sent it back", () => {
    const got = report(patchFor("a.go", ["x"]), statements)
    expect(got.evidence.commands).toEqual([{ stage: "prove", command: "go test ./...", exitCode: 0, ok: true }])
    expect(got.evidence.inspect?.verdict).toBe("advance")
    expect(got.evidence.sentBack).toBe(1)
  })

  it("keeps only the latest prove and qa so old rounds do not read as current", () => {
    const rounds: Statement[] = [
      { stage: "qa", iteration: 1, verdict: "stop", reason: "", commands: [{ command: "old", exitCode: 1 }] },
      { stage: "qa", iteration: 1, verdict: "advance", reason: "", commands: [{ command: "new", exitCode: 0 }] },
    ]
    expect(report(patchFor("a.go", ["x"]), rounds).evidence.commands.map((item) => item.command)).toEqual(["new"])
  })

  it("treats repeated inspect returns as a risk", () => {
    const back: Statement = { stage: "inspect", iteration: 1, verdict: "stop", reason: "x", commands: [] }
    const ids = report(patchFor("a.go", ["x"]), [back, back]).risks.map((item) => item.id)
    expect(ids).toContain("sent-back")
    expect(report(patchFor("a.go", ["x"]), [back]).risks.map((item) => item.id)).not.toContain("sent-back")
  })

  it("lists what it could not check", () => {
    const got = report(patchFor("a.go", ["x"]), [], true)
    expect(got.gaps.join(" ")).toMatch(/patch was cut/)
    expect(got.gaps.join(" ")).toMatch(/No prove or QA/)
    expect(got.gaps.join(" ")).toMatch(/Inspect has not read/)
    expect(got.gaps.join(" ")).toMatch(/no feature document log/)
  })

  it("notes a log with no Decision or Risk lines", () => {
    const feature = parseFeature("# T\n\n## Log\nL1 brief")
    const got = buildReport({ files: parseDiff(patchFor("a.go", ["x"])), truncated: false, statements, cursor, feature })
    expect(got.gaps.join(" ")).toMatch(/no Decision or Risk lines/)
  })

  it("has nothing to assess before the first commit", () => {
    const got = report("")
    expect(got.verdict.headline).toBe("Nothing to assess yet")
  })
})
