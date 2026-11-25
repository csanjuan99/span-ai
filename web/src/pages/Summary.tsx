"use client"

import { CopyIcon } from "lucide-react"
import { useState } from "react"
import { Link, useParams } from "react-router"

import { Shell } from "@/components/shell/Shell"
import { SummaryView } from "@/components/summary/SummaryView"
import { Button } from "@/components/ui/button"
import { useSpan } from "@/hooks/use-span"
import { summaryOf, toMarkdown } from "@/lib/summary"

/** A finished run in one page a person can read, share or copy as markdown. */
export function Summary() {
  const { id = "" } = useParams()
  const state = useSpan(id)
  const [note, setNote] = useState("")
  const span = state.span

  async function copy(text: string) {
    try {
      await navigator.clipboard.writeText(text)
      setNote("Copied as markdown.")
    } catch {
      setNote("The clipboard refused. Select the summary and copy it by hand.")
    }
  }

  if (span === null) {
    return (
      <Shell path={`~/spans/${id}/summary`} status="summary" lines={[]}>
        <Link
          className="text-sm text-primary underline-offset-4 hover:underline"
          to={`/spans/${id}`}
        >
          $ run
        </Link>
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
  })
  return (
    <Shell path={`~/spans/${id}/summary`} status="summary" lines={[]}>
      <header className="flex flex-col gap-3">
        <Link
          className="text-sm text-primary underline-offset-4 hover:underline"
          to={`/spans/${id}`}
        >
          $ run
        </Link>
        <h1 className="text-xl font-semibold text-balance tabular-nums">
          {summary.title}
        </h1>
        {span.cursor.done ? null : (
          <p className="text-sm text-pretty text-hold">
            This run is not finished, so the summary may change.
          </p>
        )}
        <div className="flex flex-wrap items-center gap-3">
          <Button
            type="button"
            variant="outline"
            onClick={() => void copy(toMarkdown(summary))}
          >
            <CopyIcon aria-hidden="true" />
            Copy as markdown
          </Button>
          {note !== "" ? (
            <p role="status" className="text-sm text-muted-foreground">
              {note}
            </p>
          ) : null}
        </div>
      </header>
      <SummaryView summary={summary} />
    </Shell>
  )
}
