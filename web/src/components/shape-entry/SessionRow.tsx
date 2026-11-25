"use client"

import type { ReactNode } from "react"
import { Link } from "react-router"

import { IdChip } from "@/components/list-controls/IdChip"
import { StatePill } from "@/components/shape-entry/StatePill"
import { buttonVariants } from "@/components/ui/button"
import {
  ageLabel,
  agentModel,
  rowAction,
  sessionRef,
  sessionTitle,
} from "@/lib/shape-sessions"
import type { ShapeSession } from "@/lib/types"
import { cn } from "@/lib/utils"
import { monoWrapped } from "@/components/shaping/styles"

/** The repo a session works in; the whole folder when it names none. */
function repoLabel(session: ShapeSession): string {
  return session.repo === "" ? "whole folder" : session.repo
}

/**
 * One session of a list: its ref, what it is about, its state, the agent and model, when it
 * last moved and the one action it needs. `tidy` is extra controls under the title.
 */
export function SessionRow({
  session,
  workspace,
  now,
  tidy,
}: {
  session: ShapeSession
  workspace: string
  now: Date
  tidy?: ReactNode
}) {
  const action = rowAction(session)
  const key = session.jiraKey ?? ""
  return (
    <li className="grid grid-cols-1 items-center gap-2 border-t border-border p-4 first:border-t-0 md:grid-cols-[96px_minmax(0,1fr)_150px_190px_110px_112px] md:gap-4 md:px-5 md:py-3">
      <span>
        <IdChip>{sessionRef(session)}</IdChip>
      </span>
      <div className="flex min-w-0 flex-col">
        <span className="text-sm leading-[22px] font-semibold md:truncate">
          {sessionTitle(session)}
        </span>
        <span className={monoWrapped}>
          {repoLabel(session)}
          {key !== "" ? <span> · {key}</span> : null}
        </span>
        {tidy}
      </div>
      <span>
        <StatePill state={session.status} />
      </span>
      <span className={monoWrapped}>{agentModel(session)}</span>
      <span className="text-xs leading-4 text-muted-foreground tabular-nums md:text-right">
        {ageLabel(session.updatedAt, now)}
      </span>
      <span className="flex">
        <Link
          to={`/shaping/${session.id}?workspace=${encodeURIComponent(workspace)}`}
          className={cn(
            buttonVariants({ variant: action.variant }),
            "h-11 flex-1 rounded-[10px] px-3 text-[13px] md:h-8 md:flex-none"
          )}
        >
          {action.label}
        </Link>
      </span>
    </li>
  )
}

/** The rows of a list, in the design's rounded panel. */
export function SessionList({ children }: { children: ReactNode }) {
  return (
    <ul className="overflow-hidden rounded-[14px] border border-border bg-card shadow-sm">
      {children}
    </ul>
  )
}
