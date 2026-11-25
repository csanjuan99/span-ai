import type { LucideIcon } from "lucide-react"
import type { ReactNode } from "react"

import { cn } from "@/lib/utils"

/** The box that says what an action will do before it is pressed: where, what is kept, what comes next. */
export function Consequences({
  title,
  icon: Icon,
  iconClassName,
  rows,
}: {
  title: string
  icon: LucideIcon
  /** Colors the mark when the design does. */
  iconClassName?: string
  rows: { label: string; body: ReactNode }[]
}) {
  return (
    <div className="rounded-[10px] bg-muted px-3.5 py-3">
      <div className="mb-1 flex items-center gap-2 text-sm font-semibold">
        <Icon className={cn("size-4", iconClassName)} aria-hidden="true" />
        {title}
      </div>
      {rows.map((row, index) => (
        <div
          key={`${row.label}-${index}`}
          className="grid grid-cols-[64px_minmax(0,1fr)] gap-2.5 border-t border-border py-2"
        >
          <div className="pt-0.5 text-xs font-medium tracking-wide text-muted-foreground">
            {row.label}
          </div>
          <div className="min-w-0 text-[13px] leading-5">{row.body}</div>
        </div>
      ))}
    </div>
  )
}
