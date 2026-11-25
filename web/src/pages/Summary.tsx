"use client"

import { Link, useParams } from "react-router"

import { Shell } from "@/components/shell/Shell"
import { SharePanel } from "@/components/summary/SharePanel"
import { SummaryHead, SummaryView } from "@/components/summary/SummaryView"
import { useSpan } from "@/hooks/use-span"
import { touchArea } from "@/lib/button-class"
import { summaryOf } from "@/lib/summary"
import { cn } from "@/lib/utils"

/** The way back to the run the summary describes. */
function RunLink({ id }: { id: string }) {
  return (
    <Link
      className={cn(
        touchArea,
        "w-fit text-sm text-primary underline-offset-4 hover:underline"
      )}
      to={`/spans/${id}`}
    >
      $ run
    </Link>
  )
}

/** A finished run in one page a person can read, share or copy as markdown. */
export function Summary() {
  const { id = "" } = useParams()
  const state = useSpan(id)
  const span = state.span

  if (span === null) {
    return (
      <Shell path={`~/spans/${id}/summary`} status="summary" lines={[]}>
        <RunLink id={id} />
        <p role="status" className="text-muted-foreground">
          {state.problem === null
            ? "Loading the summary…"
            : "This run cannot be read, so there is no summary."}
        </p>
      </Shell>
    )
  }
  const summary = summaryOf({
    cursor: span.cursor,
    statements: span.statements,
    implementation: state.implementation,
    label: span.label ?? "",
    title:
      span.title !== undefined && span.title !== "" ? span.title : undefined,
  })
  return (
    <Shell path={`~/spans/${id}/summary`} status="summary" lines={[]}>
      <div className="mx-auto grid w-full max-w-[1440px] grid-cols-[minmax(0,1fr)] items-start gap-6 lg:grid-cols-[minmax(0,1fr)_440px]">
        <div className="flex min-w-0 flex-col gap-6">
          <RunLink id={id} />
          <SummaryHead summary={summary} done={span.cursor.done} />
          {span.cursor.done ? null : (
            <p className="text-sm text-pretty text-hold">
              This run is not finished, so the summary may change.
            </p>
          )}
          <SummaryView summary={summary} />
        </div>
        <SharePanel summary={summary} />
      </div>
    </Shell>
  )
}
