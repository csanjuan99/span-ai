import type { ReactNode } from "react"

/** The controls of a run in one panel: agent, note, restart, pause. Each child brings its own padding. */
export function RunControls({ children }: { children: ReactNode }) {
  return (
    <section
      aria-label="Run controls"
      className="flex flex-col divide-y divide-border overflow-hidden rounded-xl border border-border bg-card shadow-sm [&>*]:p-4"
    >
      <h2 className="text-sm font-medium">Run controls</h2>
      {children}
    </section>
  )
}
