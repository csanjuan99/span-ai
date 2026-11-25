import { describe, expect, it } from "vitest"

import {
  caseTime,
  exitLabel,
  foldLog,
  gateLog,
  groupByFile,
  qaCasesView,
  qaView,
  seconds,
  transcriptView,
  type LogLine,
  type QaCase,
} from "@/lib/qa"

const line = (n: number, kind: "ctx" | "err" = "ctx"): LogLine => ({ n, text: `line ${n}`, kind })

function aCase(over: Partial<QaCase>): QaCase {
  return {
    name: "TestA",
    gate: "fast",
    result: "pass",
    elapsedMs: 10,
    path: "a_test.go",
    file: "a_test.go",
    line: 1,
    message: "",
    output: [],
    isNew: false,
    history: [],
    ...over,
  }
}

describe("reading the QA payloads", () => {
  it("reads the overview with its gates and honest fields", () => {
    const view = qaView({
      run: "SPAN-482",
      state: "no_cases",
      verdict: "held",
      headline: "The fast command ran, but it found 0 cases",
      exit: 11,
      evidence: { kind: "span", note: "Span ran the command." },
      counts: { passed: 41, failed: 2, skipped: 0, seconds: 6.4 },
      gates: [{ name: "fast", verdict: "fail", ran: true, exitCode: 1, durationMs: 6400, counts: { passed: 41, failed: 2 } }],
      next: "Ask the agent to add tests.",
    })
    expect(view.state).toBe("no_cases")
    expect(view.exit).toBe(11)
    expect(view.evidenceNote).toBe("Span ran the command.")
    expect(view.gates[0]?.counts).toEqual({ passed: 41, failed: 2, skipped: 0 })
    expect(view.next).toBe("Ask the agent to add tests.")
  })

  it("falls back to not run for a verdict it does not know", () => {
    expect(qaView({ verdict: "weird", state: "weird", gates: [] }).verdict).toBe("notrun")
    expect(() => qaView(null)).toThrow()
  })

  it("reads cases with their history and the new flag", () => {
    const view = qaCasesView({
      known: true,
      total: 2,
      new: 1,
      cases: [{ name: "TestA", result: "fail", new: true, line: 58, history: [{ kind: "first run", result: "fail" }] }],
    })
    expect(view.added).toBe(1)
    expect(view.cases[0]?.isNew).toBe(true)
    expect(view.cases[0]?.history[0]?.kind).toBe("first run")
  })

  it("reads a log and a transcript", () => {
    const log = gateLog({ gate: "fast", totalLines: 142, firstLine: 1, matches: [4, "x"], lines: [{ n: 4, text: "boom", kind: "err" }] })
    expect(log.matches).toEqual([4])
    expect(log.lines[0]?.kind).toBe("err")
    const transcript = transcriptView({ stage: "build", total: 3, turns: [{ n: 1, text: "hi", firstLine: 1, lines: 1 }] })
    expect(transcript.turns[0]?.text).toBe("hi")
  })
})

describe("how the numbers read", () => {
  it("writes seconds as the design does", () => {
    expect(seconds(4100)).toBe("4.1 s")
    expect(seconds(11000)).toBe("11 s")
    expect(seconds(0)).toBe("–")
    expect(caseTime(20)).toBe("0.02s")
  })

  it("names the exit codes", () => {
    expect(exitLabel(10)).toBe("a gate failed")
    expect(exitLabel(11)).toBe("no proof")
    expect(exitLabel(0)).toBe("ok")
  })
})

describe("grouping cases by file", () => {
  it("keeps the server order and counts each file", () => {
    const groups = groupByFile([
      aCase({ name: "TestFail", result: "fail", path: "r_test.go" }),
      aCase({ name: "TestOk", path: "r_test.go" }),
      aCase({ name: "TestOther", path: "c_test.go" }),
    ])
    expect(groups.map((group) => group.path)).toEqual(["r_test.go", "c_test.go"])
    expect(groups[0]).toMatchObject({ passed: 1, failed: 1 })
  })
})

describe("folding a log", () => {
  const lines = Array.from({ length: 30 }, (_, index) => line(index + 1, index + 1 === 20 ? "err" : "ctx"))

  it("folds the quiet middle and keeps the head, the tail and the error with its context", () => {
    const items = foldLog(lines, [], new Set())
    const folds = items.filter((item) => item.kind === "fold")
    expect(folds).toHaveLength(2)
    expect(folds[0]).toMatchObject({ from: 6, to: 17, count: 12 })
    const shown = items.filter((item) => item.kind === "line").map((item) => (item.kind === "line" ? item.line.n : 0))
    expect(shown).toContain(20)
    expect(shown).toContain(18)
    expect(shown).toContain(30)
  })

  it("keeps a search match unfolded and opens the folds a person asked for", () => {
    const withMatch = foldLog(lines, [10], new Set())
    expect(withMatch.some((item) => item.kind === "line" && item.line.n === 10)).toBe(true)
    const opened = foldLog(lines, [], new Set(["6-17"]))
    expect(opened.filter((item) => item.kind === "fold")).toHaveLength(1)
  })

  it("does not fold a stretch shorter than three lines", () => {
    const short = Array.from({ length: 14 }, (_, index) => line(index + 1, index + 1 === 8 ? "err" : "ctx"))
    expect(foldLog(short, [], new Set(), 2, 5, 3).filter((item) => item.kind === "fold")).toHaveLength(0)
  })
})
