import { CircleAlertIcon } from "lucide-react"

/** A run that finished without sealing its build: what happened, what is kept, and what to do next. */
export function UnsealedCard({ sealed, commits }: { sealed: number; commits: number }) {
  return (
    <section
      aria-label="Build never sealed"
      className="flex flex-col gap-3 rounded-[14px] border border-border bg-card p-4 shadow-sm sm:p-5"
    >
      <div className="flex items-center gap-2 text-hold">
        <CircleAlertIcon className="size-4" aria-hidden="true" />
        <h2 className="text-sm font-semibold">Build never sealed</h2>
      </div>
      <p className="text-sm text-pretty">
        <span className="font-medium">Cause.</span> The run finished before build was sealed, so there are no proof
        results to show.
      </p>
      <dl className="grid grid-cols-[64px_1fr] gap-x-3 gap-y-2 text-sm">
        <dt className="text-muted-foreground">Kept</dt>
        <dd>
          {sealed} {sealed === 1 ? "stage" : "stages"} sealed
          {commits > 0 ? ` · ${commits} ${commits === 1 ? "commit" : "commits"} on the branch` : ""}.
        </dd>
        <dt className="text-muted-foreground">Next</dt>
        <dd>Start a new run to build again; the branch and the sealed stages above stay as they are.</dd>
      </dl>
    </section>
  )
}
