import { Link } from "react-router"

import type { StartedRun } from "@/lib/api"

interface StartedRunsProps {
  runs: StartedRun[]
  partial: boolean
  /** What the server said about repos that did not start. */
  notes: string
}

/**
 * The runs one submission started. A submission over several repos starts one run per
 * repo, and following only the first would hide the others while their agents work.
 */
export function StartedRuns({ runs, partial, notes }: StartedRunsProps) {
  return (
    <section
      className="mt-3 flex flex-col gap-2"
      aria-label="Runs started"
      role="status"
    >
      {partial ? (
        <p className="text-sm text-pretty text-hold">
          Some repos did not start. The others are running.
        </p>
      ) : null}
      {notes.trim() !== "" ? (
        <p className="text-sm text-pretty whitespace-pre-line text-hold">
          {notes.trim()}
        </p>
      ) : null}
      <ul className="flex flex-col gap-1">
        {runs.map((run) => (
          <li key={run.id}>
            <Link
              className="text-primary tabular-nums underline-offset-4 hover:underline"
              to={`/spans/${run.id}`}
            >
              {run.repo !== "" ? `${run.repo} · ${run.id}` : run.id}
            </Link>
          </li>
        ))}
      </ul>
    </section>
  )
}
