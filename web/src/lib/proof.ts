import type { DiffFile } from "@/lib/diff"
import type { Level, Report } from "@/lib/report"

/** The proof of a run in three cells: how risky, how big, and what the checks said. */
export interface Proof {
  verdict: { level: Level; headline: string }
  diff: { files: number; added: number; removed: number; kinds: string }
  checks: {
    passed: number
    total: number
    failing: string
    inspect: "passed" | "blocked" | "not yet"
    /** The tests the checks counted, or null when none printed a count. */
    tests: { passed: number; failed: number } | null
  }
}

const kinds = [
  ["A", "added"],
  ["M", "modified"],
  ["R", "renamed"],
  ["D", "deleted"],
] as const

/**
 * Reads the proof from the report the page already builds, so the top of the page and the
 * Implementation tab never disagree. It adds no measurement of its own. The test counts are
 * the engine's, read from what each check printed; a check that printed none adds nothing.
 */
export function proofOf(report: Report, files: DiffFile[]): Proof {
  const { verdict, totals, evidence } = report
  const failed = evidence.commands.find((command) => !command.ok)
  const letters = kinds
    .map(([letter, status]) => ({ letter, count: files.filter((file) => file.status === status).length }))
    .filter((entry) => entry.count > 0)
    .map((entry) => `${entry.letter} ${entry.count}`)
  return {
    verdict: { level: verdict.level, headline: verdict.headline },
    diff: { files: totals.files, added: totals.added, removed: totals.removed, kinds: letters.join(" · ") },
    checks: {
      passed: evidence.commands.filter((command) => command.ok).length,
      total: evidence.commands.length,
      failing: failed === undefined ? "" : `${failed.command} · exit ${failed.exitCode}`,
      inspect: evidence.inspect === null ? "not yet" : evidence.inspect.verdict === "advance" ? "passed" : "blocked",
      tests: evidence.tests === null ? null : { passed: evidence.tests.passed, failed: evidence.tests.failed },
    },
  }
}
