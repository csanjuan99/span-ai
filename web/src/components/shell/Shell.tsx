import type { ReactNode } from "react"

import { AppBar } from "@/components/app-bar/AppBar"
import { Timeline } from "@/components/timeline/Timeline"
import type { TimelineLine } from "@/lib/timeline"

/**
 * One page: the shared top bar, the content, and, for a run that is writing a log, its
 * timeline and status line at the foot.
 */
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
      <AppBar path={path} />
      <main className="min-h-0 w-full flex-1 overflow-y-auto">
        <div className="mx-auto flex w-full max-w-[1440px] flex-col gap-6 px-4 pt-4 pb-12 sm:px-10 sm:pt-8 sm:pb-16">
          {children}
        </div>
      </main>
      {lines.length > 0 ? (
        <footer className="shrink-0 bg-surface-code">
          <Timeline lines={lines} />
          <div className="flex items-center justify-between border-t border-border px-4 py-2 font-mono text-xs text-muted-foreground">
            <span className="tabular-nums">{status}</span>
            <span>span serve</span>
          </div>
        </footer>
      ) : null}
    </div>
  )
}
