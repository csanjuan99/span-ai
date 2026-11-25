"use client"

import { Link } from "react-router"

import { StartedRuns } from "@/components/started-runs/StartedRuns"
import type { Started } from "@/lib/api"

/** N3: one result per repository after Start. */
export function StartingResults({
  result,
  onOpen,
}: {
  result: Started
  onOpen: (id: string) => void
}) {
  if (result.runs.length <= 1 && !result.partial) {
    return null
  }
  return (
    <section aria-label="Starting results" className="flex flex-col gap-3">
      <h2 className="text-lg font-semibold">Starting runs</h2>
      <StartedRuns
        runs={result.runs}
        partial={result.partial}
        notes={result.notes}
      />
      <ul className="flex flex-col gap-2">
        {result.runs.map((run) => (
          <li
            key={run.id}
            className="flex flex-wrap items-center gap-2 text-sm"
          >
            <span className="font-mono">
              {run.repo === "" ? "repo" : run.repo}
            </span>
            <Link
              className="text-primary underline-offset-4 hover:underline"
              to={`/spans/${run.id}`}
              onClick={(event) => {
                event.preventDefault()
                onOpen(run.id)
              }}
            >
              Open {run.id}
            </Link>
          </li>
        ))}
      </ul>
    </section>
  )
}
