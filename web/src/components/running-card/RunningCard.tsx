"use client"

import { Loader } from "lucide-react"

import { StatusBlock } from "@/components/status-block/StatusBlock"
import { Button } from "@/components/ui/button"
import type { Running } from "@/lib/running"

/** Shown while the run goes on by itself: nothing is asked, and the one thing to do is pause. */
export function RunningCard({ running, onPause }: { running: Running; onPause: () => void }) {
  return (
    <StatusBlock label="Running" tone="info" icon={Loader} title="Running · Nothing needed from you" where={running.where}>
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
    </StatusBlock>
  )
}
