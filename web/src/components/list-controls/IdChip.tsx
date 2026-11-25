import type { ReactNode } from "react"

/** The small monospace chip that names a row: a Jira key or a session id. */
export function IdChip({ children }: { children: ReactNode }) {
  return (
    <span className="inline-block rounded-md border border-border bg-muted px-1.5 font-mono text-xs leading-[18px] whitespace-nowrap text-foreground">
      {children}
    </span>
  )
}
