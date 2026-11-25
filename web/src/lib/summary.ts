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
  /** The story key, or empty when the run has none. */
  issue: string
  /** The branch the work landed on, or empty. */
  base: string
  /** How long the run took in milliseconds, from its start to its last sealed stage. 0 when unknown. */
  durationMs: number
  /** When the last stage sealed, RFC 3339. Empty when unknown. */
  finishedAt: string
}

/** The parts of a summary a person can leave out of what they share. */
export const sections = ["files", "tests", "risks", "links"] as const

/** One of {@link sections}. */
export type Section = (typeof sections)[number]

/** How long a run took, in words: "45 s", "12 min", "1 h 33 min". Empty when it is not known. */
export function durationText(ms: number): string {
  if (ms <= 0) {
    return ""
  }
  if (ms < 60000) {
    return `${Math.max(1, Math.round(ms / 1000))} s`
  }
  const minutes = Math.round(ms / 60000)
  if (minutes < 60) {
    return `${minutes} min`
  }
  const rest = minutes % 60
  return rest === 0
    ? `${Math.floor(minutes / 60)} h`
    : `${Math.floor(minutes / 60)} h ${rest} min`
}

function span(
  cursor: Cursor,
  statements: Statement[]
): { durationMs: number; finishedAt: string } {
  const start = Date.parse(cursor.startedAt ?? "")
  const ends = statements
    .filter((item) => item.verdict === "advance")
    .map((item) => Date.parse(item.time ?? ""))
    .filter((time) => !Number.isNaN(time))
  if (Number.isNaN(start) || ends.length === 0) {
    return { durationMs: 0, finishedAt: "" }
  }
  const end = Math.max(...ends)
  return {
    durationMs: Math.max(0, end - start),
    finishedAt: new Date(end).toISOString(),
  }
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
  /** What the run is about. It heads the page when there is one. */
  title?: string
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
    title: input.title ?? (input.label === "" ? cursor.id : input.label),
    outcome: done.landed,
    links:
      done.change === null
        ? []
        : [{ label: changeLabel(cursor.changeURL), href: done.change.href }],
    sha: cursor.gitSHA === "" ? "" : shortHash(cursor.gitSHA),
    issue: cursor.issue,
    base: cursor.base,
    ...span(cursor, statements),
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

/** The summary as markdown, ready to paste. A part left out of `include` is left out of the text. */
export function toMarkdown(
  summary: Summary,
  include: ReadonlySet<Section> = new Set(sections)
): string {
  const head =
    summary.issue === "" || summary.title.includes(summary.issue)
      ? summary.title
      : `${summary.issue} · ${summary.title}`
  const out = [`# ${head}`, "", summary.outcome]
  const took = durationText(summary.durationMs)
  if (took !== "") {
    out.push("", `Took ${took}.`)
  }
  if (!summary.assessed) {
    out.push("", "Not assessed: there is no diff to read for this run.")
  } else {
    if (summary.verdict !== null) {
      out.push(
        "",
        `**${summary.verdict.headline}** (${summary.verdict.level} risk)`
      )
    }
    if (include.has("files")) {
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
  }
  if (summary.tests !== null && include.has("tests")) {
    const failed =
      summary.tests.failed > 0 ? ` · ${summary.tests.failed} failed` : ""
    out.push("", `## Tests`, "", `${summary.tests.passed} passed${failed}`)
  }
  if (summary.risks.length > 0 && include.has("risks")) {
    out.push(
      "",
      "## Risks",
      "",
      ...summary.risks.map((risk) => `- ${risk.title} (${risk.level})`)
    )
  }
  if (summary.links.length > 0 && include.has("links")) {
    out.push(
      "",
      "## Links",
      "",
      ...summary.links.map((link) => `- [${link.label}](${link.href})`)
    )
  }
  return `${out.join("\n")}\n`
}
