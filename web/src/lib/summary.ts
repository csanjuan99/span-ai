import { changeOf } from "@/lib/change"
import { doneOf } from "@/lib/done"
import { changeLabel, shortHash } from "@/lib/ledger"
import { proofOf } from "@/lib/proof"
import type { Level } from "@/lib/report"
import type { Cursor, ImplementationView, Statement } from "@/lib/types"

/** The letter a changed file carries: added, modified, deleted or renamed. */
export type FileLetter = "A" | "M" | "D" | "R"

const letters = {
  added: "A",
  modified: "M",
  deleted: "D",
  renamed: "R",
} as const

/** One run, finished, in the shape a person can paste into a chat or a ticket. */
export interface Summary {
  title: string
  /** How risky and what to do about it. Empty when the diff was not read. */
  verdict: { level: Level; headline: string } | null
  /** Where the work landed, in words. Claims a merge only when the ledger sealed it. */
  outcome: string
  /** False when there is no diff to read, so nothing about risk or size is claimed. */
  assessed: boolean
  files: { path: string; letter: FileLetter; added: number; removed: number }[]
  totals: { files: number; added: number; removed: number }
  /** The tests the checks counted, or null when none printed a count. */
  tests: { passed: number; failed: number } | null
  risks: { title: string; level: Level }[]
  links: { label: string; href: string }[]
  /** The short commit the run ended on. */
  sha: string
}

/**
 * Reads a run into a summary. It adds no measurement of its own: files, risks and tests come
 * from the same report the run page builds, and a part with no evidence says nothing.
 */
export function summaryOf(input: {
  cursor: Cursor
  statements: Statement[]
  implementation: ImplementationView | null
  label: string
}): Summary {
  const { cursor, statements } = input
  const sealed = statements
    .filter((item) => item.verdict === "advance")
    .map((item) => item.stage)
  const done = doneOf({
    sealed: new Set(sealed).size,
    merged: sealed.includes("merge"),
    sha: cursor.gitSHA,
    base: cursor.base,
    branch: cursor.branch,
    issue: cursor.issue,
    changeURL: cursor.changeURL,
  })
  const base = {
    title: input.label === "" ? cursor.id : input.label,
    outcome: done.landed,
    links:
      done.change === null
        ? []
        : [{ label: changeLabel(cursor.changeURL), href: done.change.href }],
    sha: cursor.gitSHA === "" ? "" : shortHash(cursor.gitSHA),
  }
  const view = input.implementation
  const change =
    view === null
      ? null
      : changeOf({ ...view, statements, cursor, feature: null })
  if (change === null || change.files.length === 0) {
    return {
      ...base,
      verdict: null,
      assessed: false,
      files: [],
      totals: { files: 0, added: 0, removed: 0 },
      tests: null,
      risks: [],
    }
  }
  const proof = proofOf(change.report, change.files)
  return {
    ...base,
    verdict: proof.verdict,
    assessed: true,
    files: change.files.map((file) => ({
      path: file.path,
      letter: letters[file.status],
      added: file.added,
      removed: file.removed,
    })),
    totals: change.report.totals,
    tests: proof.checks.tests,
    risks: change.report.risks.map((risk) => ({
      title: risk.title,
      level: risk.level,
    })),
    links: base.links,
  }
}

/** The summary as markdown, ready to paste. */
export function toMarkdown(summary: Summary): string {
  const out = [`# ${summary.title}`, "", summary.outcome]
  if (!summary.assessed) {
    out.push("", "Not assessed: there is no diff to read for this run.")
  } else {
    if (summary.verdict !== null) {
      out.push(
        "",
        `**${summary.verdict.headline}** (${summary.verdict.level} risk)`
      )
    }
    out.push(
      "",
      `## Files · ${summary.totals.files} changed, +${summary.totals.added} −${summary.totals.removed}`,
      "",
      ...summary.files.map(
        (file) =>
          `- ${file.letter} \`${file.path}\` +${file.added} −${file.removed}`
      )
    )
  }
  if (summary.tests !== null) {
    const failed =
      summary.tests.failed > 0 ? ` · ${summary.tests.failed} failed` : ""
    out.push("", `## Tests`, "", `${summary.tests.passed} passed${failed}`)
  }
  if (summary.risks.length > 0) {
    out.push(
      "",
      "## Risks",
      "",
      ...summary.risks.map((risk) => `- ${risk.title} (${risk.level})`)
    )
  }
  if (summary.links.length > 0) {
    out.push(
      "",
      "## Links",
      "",
      ...summary.links.map((link) => `- [${link.label}](${link.href})`)
    )
  }
  return `${out.join("\n")}\n`
}
