"use client"

import { Link } from "react-router"

import { SessionList, SessionRow } from "@/components/shape-entry/SessionRow"
import type { ShapeSession } from "@/lib/types"
import { headingText } from "@/components/shaping/styles"
import { cn } from "@/lib/utils"
import { touchArea } from "@/lib/button-class"

/** How many sessions the start page shows before "See all". */
export const recentCount = 4

/** The latest sessions under the start panel, with a link to the full list. */
export function RecentSessions({
  sessions,
  total,
  workspace,
  now,
  allHref,
}: {
  sessions: ShapeSession[]
  /** How many sessions the folder has, archived ones left out. */
  total: number
  workspace: string
  now: Date
  allHref: string
}) {
  return (
    <section aria-label="Recent sessions">
      <div className="flex flex-wrap items-center gap-2 px-1 py-2">
        <h2 className={headingText}>Recent sessions</h2>
        <Link
          to={allHref}
          className={cn(
            touchArea,
            "ml-auto text-[13px] leading-5 text-primary underline-offset-4 hover:underline"
          )}
        >
          See all {total}
        </Link>
      </div>
      <SessionList>
        {sessions.slice(0, recentCount).map((session) => (
          <SessionRow
            key={session.id}
            session={session}
            workspace={workspace}
            now={now}
          />
        ))}
      </SessionList>
    </section>
  )
}
