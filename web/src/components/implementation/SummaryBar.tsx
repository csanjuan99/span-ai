import type { ReactNode } from "react"

import type { DiffFile } from "@/lib/diff"
import { statusCounts, testsAdded } from "@/lib/overview"
import { totals } from "@/lib/diff"
import { LineCounts } from "@/components/files-changed/LineCounts"

function Cell({
  label,
  children,
  caption,
}: {
  label: string
  children: ReactNode
  caption: ReactNode
}) {
  return (
    <div className="flex min-w-0 flex-col gap-1 border-t border-border p-4 first:border-t-0 lg:border-t-0 lg:border-l lg:px-5 lg:first:border-l-0">
      <div className="text-xs font-medium tracking-[0.02em] text-muted-foreground">
        {label}
      </div>
      <div className="text-lg leading-[26px] font-semibold tabular-nums">
        {children}
      </div>
      <div className="truncate text-xs text-muted-foreground">{caption}</div>
    </div>
  )
}

/** Files, lines, commits and tests the change adds, in four cells above the detail. */
export function SummaryBar({
  files,
  commits,
  branch,
  base,
}: {
  files: DiffFile[]
  commits: number
  branch: string
  base: string
}) {
  const counts = statusCounts(files)
  const sum = totals(files)
  const tests = testsAdded(files)
  return (
    <section
      aria-label="Summary"
      className="grid grid-cols-2 gap-y-0 rounded-[14px] border border-border bg-card shadow-sm lg:grid-cols-4 [&>div:nth-child(n+3)]:border-t [&>div:nth-child(n+3)]:lg:border-t-0"
    >
      <Cell
        label="Files"
        caption={`A ${counts.added} · M ${counts.modified} · R ${counts.renamed} · D ${counts.deleted}`}
      >
        {sum.files}
      </Cell>
      <Cell
        label="Lines"
        caption={base === "" ? "against its base" : `against ${base}`}
      >
        <LineCounts added={sum.added} removed={sum.removed} />
      </Cell>
      <Cell
        label="Commits"
        caption={
          branch === "" ? (
            "on the branch"
          ) : (
            <span className="font-mono">on {branch}</span>
          )
        }
      >
        {commits}
      </Cell>
      <Cell
        label="Tests added"
        caption={
          tests.files.length === 0 ? (
            "none found"
          ) : (
            <span className="font-mono">
              in {tests.files.map((path) => path.split("/").pop()).join(", ")}
            </span>
          )
        }
      >
        {tests.count}
      </Cell>
    </section>
  )
}
