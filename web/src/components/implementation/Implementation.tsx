import { DiffFileCard } from "@/components/implementation/DiffFileCard"
import { Audit } from "@/components/implementation/report/Audit"
import { Evidence } from "@/components/implementation/report/Evidence"
import { Reasoning } from "@/components/implementation/report/Reasoning"
import { Risks } from "@/components/implementation/report/Risks"
import { Transcript } from "@/components/implementation/report/Transcript"
import { Verdict } from "@/components/implementation/report/Verdict"
import { parseDiff, parseStat, totals, type DiffFile } from "@/lib/diff"
import type { JournalState } from "@/hooks/use-events"
import type { FeatureReport } from "@/lib/featuredoc"
import { buildReport } from "@/lib/report"
import type { Cursor, Statement } from "@/lib/types"

/** Files the stat lists that the patch did not reach, because the patch was cut. */
function missingFromPatch(stat: string, shown: DiffFile[]): DiffFile[] {
  const known = new Set(shown.map((file) => file.path))
  return parseStat(stat)
    .filter((row) => !known.has(row.path))
    .map((row) => ({
      path: row.path,
      oldPath: row.path,
      status: "modified" as const,
      binary: false,
      added: row.plus,
      removed: row.minus,
      hunks: [],
    }))
}

/** How many files open by default. A long patch starts closed so the list stays scannable. */
const openFiles = 3

/** What the page knows about the run, for the report above the diff. */
export interface RunFacts {
  cursor: Cursor
  statements: Statement[]
  agents: Record<string, string>
  feature: FeatureReport | null
  journal: JournalState
}

/** The report a person reads before trusting the change, then the diff, then the raw transcript. */
export function Implementation({
  stat,
  patch,
  truncated,
  transcript,
  run,
}: {
  stat: string
  patch: string
  truncated: boolean
  transcript: string[]
  run: RunFacts
}) {
  const parsed = parseDiff(patch)
  const files = [...parsed, ...missingFromPatch(stat, parsed)]
  const sum = totals(files)
  const report = buildReport({ files, truncated, statements: run.statements, cursor: run.cursor, feature: run.feature })
  return (
    <div className="flex flex-col gap-8">
      <section aria-label="Report" className="flex flex-col gap-6">
        <Verdict report={report} />
        <Risks risks={report.risks} reported={run.feature?.risks ?? []} />
        <Reasoning feature={run.feature} groups={report.groups} />
        <Evidence evidence={report.evidence} gaps={report.gaps} />
      </section>
      <section aria-label="Changes">
        <h2 className="flex flex-wrap items-baseline gap-x-3 text-balance text-sm">
          <span>
            <span className="text-primary">$ </span>git diff
          </span>
          {files.length > 0 ? (
            <span className="text-xs text-muted-foreground tabular-nums">
              {sum.files} {sum.files === 1 ? "file" : "files"} <span className="text-pass">+{sum.added}</span>{" "}
              <span className="text-destructive">−{sum.removed}</span>
            </span>
          ) : null}
        </h2>
        {files.length === 0 ? (
          <p className="mt-2 text-pretty text-muted-foreground">No commit on this run yet.</p>
        ) : (
          <div className="mt-3 flex flex-col gap-2">
            {files.map((file, index) => (
              <DiffFileCard key={file.path} file={file} defaultOpen={index < openFiles} />
            ))}
          </div>
        )}
        {truncated ? (
          <p className="mt-2 text-sm text-hold">The patch was cut at 256 KiB. Files past the cut show their counts only.</p>
        ) : null}
      </section>
      <section aria-label="Record" className="flex flex-col gap-2">
        <Audit cursor={run.cursor} statements={run.statements} agents={run.agents} journal={run.journal} />
        <Transcript lines={transcript} />
      </section>
    </div>
  )
}
