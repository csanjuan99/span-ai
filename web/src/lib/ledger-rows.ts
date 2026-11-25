import { toneOf, type Tone } from "@/lib/ledger"
import { clockOf } from "@/lib/timeline"
import { tidy } from "@/lib/tidy"
import type { Statement } from "@/lib/types"

/** One line of the ledger at a glance: the stage, how it went, and when. */
export interface LedgerRow {
  stage: string
  tone: Tone
  line: string
  clock: string
}

/** Stages that carry no news when they simply pass. They share one row. */
const quiet = new Set(["intake", "promote", "note"])
const longest = 120

function testsLine(statement: Statement): string {
  let passed = 0
  let failed = 0
  let counted = false
  for (const command of statement.commands) {
    if (command.tests !== undefined) {
      counted = true
      passed += command.tests.passed
      failed += command.tests.failed
    }
  }
  if (!counted) {
    return ""
  }
  const base = `${passed} ${passed === 1 ? "test" : "tests"} passed`
  return failed > 0 ? `${base} · ${failed} failed` : base
}

function sentenceOf(reason: string): string {
  const clean = tidy(reason).replace(/\s+/g, " ").trim()
  const end = /[.!?](?=\s|$)/.exec(clean)
  const first = end === null ? clean : clean.slice(0, end.index + 1)
  return first.length > longest
    ? `${first.slice(0, longest).trimEnd()}…`
    : first
}

function listOf(names: string[]): string {
  if (names.length < 2) {
    return names.join("")
  }
  return `${names.slice(0, -1).join(", ")} and ${names[names.length - 1]}`
}

/**
 * The ledger as the page opens it: one row per stage, the latest sealed, each in a line. The
 * long evidence stays in the records. Quiet stages that only passed share a single row.
 */
export function ledgerRows(statements: Statement[]): LedgerRow[] {
  const latest = new Map<string, Statement>()
  for (const item of statements) {
    latest.delete(item.stage)
    latest.set(item.stage, item)
  }
  const ordered = [...new Set(statements.map((item) => item.stage))]
  const rows: LedgerRow[] = []
  const folded: string[] = []
  for (const stage of ordered) {
    const item = latest.get(stage)
    if (item === undefined) {
      continue
    }
    const tone = toneOf(item.verdict)
    if (quiet.has(stage) && tone === "pass") {
      folded.push(stage)
      continue
    }
    rows.push({
      stage,
      tone,
      line: testsLine(item) || sentenceOf(item.reason),
      clock: clockOf(item.time ?? ""),
    })
  }
  if (folded.length === 1) {
    const item = latest.get(folded[0])
    if (item !== undefined) {
      rows.unshift({
        stage: folded[0],
        tone: "pass",
        line: sentenceOf(item.reason),
        clock: clockOf(item.time ?? ""),
      })
      rows.sort((a, b) => ordered.indexOf(a.stage) - ordered.indexOf(b.stage))
    }
  } else if (folded.length > 1) {
    rows.push({
      stage: `${folded.length} others`,
      tone: "pass",
      line: `${listOf(folded)} sealed.`,
      clock: "",
    })
  }
  return rows
}
