import { filesOf, type DiffFile } from "@/lib/diff"
import type { FeatureReport } from "@/lib/featuredoc"
import { buildReport, type Report } from "@/lib/report"
import type { Cursor, Statement } from "@/lib/types"

/** The change a run made: its files and the report read from them. Built once, shown twice. */
export interface Change {
  files: DiffFile[]
  report: Report
}

/**
 * The one place the files and the report of a change are built, so the summary at the top of
 * the page and the Implementation tab read the same bytes and cannot disagree.
 */
export function changeOf(input: {
  stat: string
  patch: string
  truncated: boolean
  statements: Statement[]
  cursor: Cursor
  feature: FeatureReport | null
}): Change {
  const files = filesOf(input.stat, input.patch)
  const report = buildReport({
    files,
    truncated: input.truncated,
    statements: input.statements,
    cursor: input.cursor,
    feature: input.feature,
  })
  return { files, report }
}
