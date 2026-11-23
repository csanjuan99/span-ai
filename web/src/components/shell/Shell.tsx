import type { ReactNode } from "react"

import { Timeline } from "@/components/timeline/Timeline"
import type { TimelineLine } from "@/lib/timeline"

/** One terminal session: a title bar, the pane, the timeline, and a status line. */
export function Shell({
  path,
  status,
  lines = [],
  children,
}: {
  path: string
  status: string
  lines?: TimelineLine[]
  children: ReactNode
}) {
  return (
    <div className="flex h-dvh flex-col overflow-hidden">
      <div className="flex shrink-0 items-center justify-between gap-4 border-b border-border px-4 py-2 text-sm">
        <span className="shrink-0 text-primary">span</span>
        <span className="min-w-0 truncate text-muted-foreground tabular-nums">{path}</span>
      </div>
      <main className="flex min-h-0 w-full flex-1 flex-col gap-6 overflow-y-auto px-4 py-6">{children}</main>
      <footer className="shrink-0">
        <Timeline lines={lines} />
        <div className="flex items-center justify-between border-t border-border px-4 py-2 text-xs text-muted-foreground">
          <span className="tabular-nums">{status}</span>
          <span>span serve</span>
        </div>
      </footer>
    </div>
  )
}
