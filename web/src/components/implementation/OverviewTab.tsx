import type { ReactNode } from "react"

import {
  FilesTable,
  type FileFlags,
} from "@/components/implementation/FilesTable"
import { PurposeBars } from "@/components/implementation/PurposeBars"
import { SummaryBar } from "@/components/implementation/SummaryBar"
import type { Change } from "@/lib/change"
import type { CommitStat } from "@/lib/implementation-api"
import type { Cursor } from "@/lib/types"
import { SideNote } from "@/components/implementation/SideNote"

/** The overview: the size of the change, where the lines went, the files, and what it was compared with. */
export function OverviewTab({
  change,
  flags,
  cursor,
  commits,
  commitCount,
  onOpen,
  onViewed,
  assessment,
}: {
  change: Change
  flags: Record<string, FileFlags>
  cursor: Cursor
  commits: CommitStat[]
  commitCount: number
  onOpen: (path: string) => void
  onViewed: (path: string, viewed: boolean) => void
  /** The verdict, risks and evidence the report reads from the change, kept beside the numbers. */
  assessment: ReactNode
}) {
  const { files, report } = change
  const oldest = commits.at(-1)
  const generated = files.filter(
    (file) => flags[file.path]?.generated === true
  ).length
  return (
    <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_300px] xl:items-start">
      <div className="flex min-w-0 flex-col gap-4">
        <SummaryBar
          files={files}
          commits={commitCount}
          branch={cursor.branch}
          base={cursor.base}
        />
        <PurposeBars groups={report.groups} />
        {files.length === 0 ? (
          <p className="rounded-[14px] border border-border bg-card p-5 text-sm text-muted-foreground">
            No commit on this run yet.
          </p>
        ) : (
          <FilesTable
            files={files}
            flags={flags}
            onOpen={onOpen}
            onViewed={onViewed}
          />
        )}
      </div>
      <aside className="flex min-w-0 flex-col gap-4">
        <SideNote
          title="Compared with"
          rows={[
            {
              term: "Base",
              bodyClassName: "font-mono text-xs",
              body: `${cursor.base === "" ? "—" : cursor.base}${oldest !== undefined && oldest.parent !== "" ? ` · ${oldest.parent.slice(0, 7)}` : ""}`,
            },
            {
              term: "Head",
              bodyClassName: "font-mono text-xs",
              body: `${cursor.gitSHA === "" ? "—" : cursor.gitSHA.slice(0, 7)}${cursor.branch === "" ? "" : ` on ${cursor.branch}`}`,
            },
          ]}
        >
          <p className="mt-2 text-xs text-pretty text-muted-foreground">
            {generated === 0
              ? "Generated files and lockfiles are collapsed. None changed in this run."
              : `Generated files and lockfiles are collapsed. ${generated} changed in this run.`}
          </p>
        </SideNote>
        {assessment}
      </aside>
    </div>
  )
}
