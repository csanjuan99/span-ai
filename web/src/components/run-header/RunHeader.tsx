import { cn } from "@/lib/utils"
import { ClockIcon, FolderIcon, GitBranchIcon } from "lucide-react"
import { Link } from "react-router"

import { IdChip } from "@/components/list-controls/IdChip"
import { stateBadge, type RunState } from "@/lib/run-header"

const tones = {
  acc: "bg-accent text-primary",
  hold: "bg-hold-soft text-hold",
  fail: "bg-fail-soft text-fail",
  pass: "bg-pass-soft text-pass",
} as const

/** Where the run is and what it is about: the story key, how it stands, its title, repo, branch and time. */
export function RunHeader({
  chip,
  state,
  conflict = false,
  title,
  repo,
  branch,
  time,
  summaryTo,
}: {
  chip: string
  state: RunState
  /** The run holds in the merge-conflict phase. */
  conflict?: boolean
  title: string
  repo: string
  branch: string
  time: string
  summaryTo: string
}) {
  const badge = stateBadge[state]
  return (
    <header className="flex flex-col gap-2">
      <div className="flex items-center gap-3">
        <IdChip>{chip}</IdChip>
        <span
          className={cn(
            "inline-flex h-6 items-center rounded-full px-2.5 text-xs font-medium tracking-wide whitespace-nowrap",
            tones[badge.tone]
          )}
        >
          {conflict ? "On hold · conflict" : badge.label}
        </span>
      </div>
      <h1 className="text-2xl leading-8 font-semibold tracking-[-0.01em] text-balance">
        {title}
      </h1>
      <div className="flex flex-wrap items-center gap-x-4 gap-y-1.5 text-[13px] leading-5 text-muted-foreground">
        {repo !== "" ? (
          <span className="inline-flex items-center gap-1.5">
            <FolderIcon className="size-3.5" aria-hidden="true" />
            <span className="font-mono text-xs break-all">{repo}</span>
          </span>
        ) : null}
        {branch !== "" ? (
          <span className="inline-flex items-center gap-1.5">
            <GitBranchIcon className="size-3.5" aria-hidden="true" />
            <span className="font-mono text-xs break-all">{branch}</span>
          </span>
        ) : null}
        {time !== "" ? (
          <span className="inline-flex items-center gap-1.5 text-xs tabular-nums">
            <ClockIcon className="size-3.5" aria-hidden="true" />
            {time}
          </span>
        ) : null}
        <Link
          to={summaryTo}
          className="text-xs text-primary underline-offset-4 hover:underline"
        >
          Summary to share
        </Link>
      </div>
    </header>
  )
}
