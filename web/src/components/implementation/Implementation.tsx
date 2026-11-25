import { DiffFileCard } from "@/components/implementation/DiffFileCard"
import { RanWithList } from "@/components/implementation/report/RanWith"
import { Audit } from "@/components/implementation/report/Audit"
import { Evidence } from "@/components/implementation/report/Evidence"
import { Reasoning } from "@/components/implementation/report/Reasoning"
import { Risks } from "@/components/implementation/report/Risks"
import { Verdict } from "@/components/implementation/report/Verdict"
import { DiffTotals } from "@/components/diff-totals/DiffTotals"
import type { Change } from "@/lib/change"
import { totals } from "@/lib/diff"
import type { JournalState } from "@/hooks/use-events"
import type { FeatureReport } from "@/lib/featuredoc"
import type { Cursor, Statement } from "@/lib/types"

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
  change,
  truncated,
  run,
}: {
  change: Change
  truncated: boolean
  run: RunFacts
}) {
  const { files, report } = change
  const sum = totals(files)
  return (
    <div className="flex flex-col gap-8">
      <section aria-label="Report" className="flex flex-col gap-6">
        <Verdict report={report} />
        <RanWithList events={run.journal.events} />
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
              <DiffTotals files={sum.files} added={sum.added} removed={sum.removed} />
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
      </section>
    </div>
  )
}
