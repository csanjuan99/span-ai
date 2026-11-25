"use client"

import { AlertTriangleIcon } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Panel } from "@/components/shaping/session-ui"
import type { ShapeFailure } from "@/lib/types"

const lead: Record<ShapeFailure["kind"], string> = {
  usage_limit: "The agent hit a usage limit.",
  repo_changed: "The agent changed your repository, so the turn was dropped.",
  error: "The agent did not finish the turn.",
}

/** Why the last turn stopped, what was kept, and the one way forward: run the turn again. */
export function SessionFailure({
  failure,
  busy,
  onRetry,
}: {
  failure: ShapeFailure
  busy: boolean
  onRetry: () => void
}) {
  return (
    <Panel
      role="alert"
      className="flex flex-col gap-3 border-fail bg-fail-soft p-4 text-fail"
    >
      <div className="flex items-center gap-2 text-sm font-semibold">
        <AlertTriangleIcon className="size-4" aria-hidden="true" />
        {lead[failure.kind]}
      </div>
      <p className="text-sm text-pretty whitespace-pre-line text-foreground">
        {failure.message}
      </p>
      <p className="text-sm text-pretty text-foreground">
        Your answers and the map so far are kept. Nothing was sealed.
      </p>
      <Button
        type="button"
        className="h-11 self-start md:h-10"
        disabled={busy}
        onClick={onRetry}
      >
        Try the turn again
      </Button>
    </Panel>
  )
}
