import { cn } from "cn"
import type { ReactNode } from "react"

/** The controls of a run in one panel: a title, then compact rows that each bring their own divider. */
export function RunControls({
  children,
  className,
}: {
  children: ReactNode
  className?: string
}) {
  return (
    <section
      aria-label="Run controls"
      className={cn(
        "flex flex-col overflow-hidden rounded-xl border border-border bg-card shadow-sm",
        className
      )}
    >
      <h2 className="px-4 pt-4 pb-2 text-sm font-medium">Run controls</h2>
      {children}
    </section>
  )
}
