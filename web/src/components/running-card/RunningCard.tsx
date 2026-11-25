"use client"

import { Loader, PauseIcon, Route } from "lucide-react"

import { Consequences } from "@/components/status-block/Consequences"
import { Lede } from "@/components/status-block/Lede"
import { StatusBlock } from "@/components/status-block/StatusBlock"
import { Button } from "@/components/ui/button"
import type { Running } from "@/lib/running"

/** Shown while the run goes on by itself: nothing is asked, and the one thing to do is pause. */
export function RunningCard({
  running,
  onPause,
}: {
  running: Running
  onPause: () => void
}) {
  return (
    <StatusBlock
      label="Running"
      tone="info"
      icon={Loader}
      title="Running"
      where={running.agent}
      aside={
        <>
          <Consequences
            title="What happens next"
            icon={Route}
            rows={[
              { label: "Next", body: running.next },
              {
                label: "Ask",
                body: "Span asks you only if it needs a person.",
              },
              ...(running.where !== ""
                ? [
                    {
                      label: "Where",
                      body: (
                        <span className="font-mono text-xs break-all">
                          {running.where}
                        </span>
                      ),
                    },
                  ]
                : []),
            ]}
          />
          {running.pause !== "" ? (
            <div className="flex flex-col gap-2">
              <Button
                type="button"
                variant="outline"
                className="h-10 w-full"
                onClick={onPause}
              >
                <PauseIcon aria-hidden="true" />
                Pause after this stage
              </Button>
              <p className="text-xs text-pretty text-muted-foreground">
                {running.pause}
              </p>
            </div>
          ) : null}
        </>
      }
    >
      <Lede
        title="Nothing needed from you"
        body={`${running.title} Span holds the run here and asks you if it needs a person.`}
      />
      {running.now !== "" ? (
        <div className="flex flex-col gap-2">
          <span className="text-xs font-medium tracking-wide text-muted-foreground">
            Now
          </span>
          <p
            className="truncate rounded-[10px] border border-border bg-surface-code px-3.5 py-2.5 font-mono text-[13px] text-foreground"
            title={running.now}
          >
            <span className="text-primary">&gt; </span>
            {running.now}
          </p>
        </div>
      ) : null}
    </StatusBlock>
  )
}
