import { Loader } from "lucide-react"

import { Button } from "@/components/ui/button"
import type { Running } from "@/lib/running"

/** Shown while the run goes on by itself: nothing is asked, and the one thing to do is pause. */
export function RunningCard({ running, onPause }: { running: Running; onPause: () => void }) {
  return (
    <section aria-label="Running" className="flex flex-col overflow-hidden rounded-xl border border-border bg-card shadow-sm">
      <div className="flex flex-wrap items-center gap-x-3 gap-y-1 bg-accent px-4 py-2.5 text-sm font-medium text-accent-foreground">
        <Loader className="size-4 shrink-0" aria-hidden="true" />
        <span>Running · Nothing needed from you</span>
        {running.where !== "" ? <span className="ml-auto font-mono text-xs font-normal">{running.where}</span> : null}
      </div>
      <div className="flex flex-col gap-4 p-4">
        <p className="text-pretty">{running.title} Span holds the run here and asks you if it needs a person.</p>
        {running.now !== "" ? (
          <p className="truncate rounded-md bg-surface-code px-3 py-2 font-mono text-xs text-foreground" title={running.now}>
            <span className="text-primary">&gt; </span>
            {running.now}
          </p>
        ) : null}
        <p className="text-pretty text-sm text-muted-foreground">{running.next}</p>
        {running.pause !== "" ? (
          <div className="flex flex-col gap-1.5">
            <Button type="button" variant="secondary" className="self-start" onClick={onPause}>
              Pause after this stage
            </Button>
            <p className="text-pretty text-sm text-muted-foreground">{running.pause}</p>
          </div>
        ) : null}
      </div>
    </section>
  )
}
