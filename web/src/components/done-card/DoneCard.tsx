import { CircleCheck, ExternalLink } from "lucide-react"

import { Consequences } from "@/components/status-block/Consequences"
import { StatusBlock } from "@/components/status-block/StatusBlock"
import { buttonVariants } from "@/components/ui/button"
import { quietButton, wideButton } from "@/lib/button-class"
import type { Done } from "@/lib/done"
import { cn } from "@/lib/utils"

/** Shown once the run is over: what finished on the left, where it went and how to open it on the right. */
export function DoneCard({ done }: { done: Done }) {
  const rows = [
    ...(done.branch !== ""
      ? [
          {
            label: "Branch",
            body: (
              <span className="font-mono text-xs break-all">{done.branch}</span>
            ),
          },
        ]
      : []),
    ...(done.issue !== ""
      ? [
          {
            label: "Story",
            body: <span className="font-mono text-xs">{done.issue}</span>,
          },
        ]
      : []),
  ]
  return (
    <StatusBlock
      label="Done"
      tone="pass"
      icon={CircleCheck}
      title="Done"
      where={done.title}
    >
      <div className="flex flex-col gap-6 lg:grid lg:grid-cols-[minmax(0,1fr)_300px]">
        <h3 className="text-lg font-semibold text-balance">{done.landed}</h3>
        <div className="flex min-w-0 flex-col gap-4">
          {rows.length > 0 ? (
            <Consequences
              title="Where it went"
              icon={CircleCheck}
              rows={rows}
            />
          ) : null}
          {done.change !== null ? (
            <a
              href={done.change.href}
              className={cn(
                buttonVariants({ variant: "outline" }),
                wideButton,
                quietButton
              )}
            >
              <ExternalLink aria-hidden="true" />
              {done.change.label}
            </a>
          ) : null}
        </div>
      </div>
    </StatusBlock>
  )
}
