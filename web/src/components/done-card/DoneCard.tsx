import { CircleCheck, ExternalLink } from "lucide-react"

import { StatusBlock } from "@/components/status-block/StatusBlock"
import type { Done } from "@/lib/done"

/** Shown once the run is over: what finished, where the work went, and how to open it. */
export function DoneCard({ done }: { done: Done }) {
  return (
    <StatusBlock label="Done" tone="pass" icon={CircleCheck} title={`Done · ${done.title}`} where="">
      <p className="text-pretty text-base font-medium">{done.landed}</p>
      <dl className="grid gap-x-6 gap-y-1 text-sm sm:grid-cols-[auto_1fr]">
        {done.branch !== "" ? (
          <>
            <dt className="text-muted-foreground">Branch</dt>
            <dd className="min-w-0 break-words font-mono text-xs">{done.branch}</dd>
          </>
        ) : null}
        {done.issue !== "" ? (
          <>
            <dt className="text-muted-foreground">Story</dt>
            <dd className="font-mono text-xs">{done.issue}</dd>
          </>
        ) : null}
      </dl>
      {done.change !== null ? (
        <a
          href={done.change.href}
          className="inline-flex items-center gap-1.5 self-start text-sm text-primary underline-offset-4 hover:underline"
        >
          {done.change.label}
          <ExternalLink className="size-3.5" aria-hidden="true" />
        </a>
      ) : null}
    </StatusBlock>
  )
}
