import type { ReactNode } from "react"

import { ChangedBadge } from "@/components/settings-edit/ChangedBadge"

/** One setting: its name, what it means when unset, the field, and whether it moved. */
export function FieldRow({
  label,
  fallback,
  changed,
  unsaved,
  children,
}: {
  label: string
  fallback: string
  /** The value differs from what span.yaml means when the key is absent. */
  changed: boolean
  /** The value was typed and is not saved yet. */
  unsaved: boolean
  children: ReactNode
}) {
  return (
    <div className="grid gap-2 border-t border-border py-4 first:border-t-0 sm:grid-cols-[200px_minmax(0,1fr)] sm:gap-4">
      <div className="flex flex-col gap-0.5">
        <div className="text-sm font-semibold">{label}</div>
        <div className="flex flex-wrap items-center gap-2">
          <span className="text-xs text-muted-foreground">
            Default: {fallback}
          </span>
          {changed ? <ChangedBadge>Changed</ChangedBadge> : null}
        </div>
      </div>
      <div className="flex min-w-0 flex-col gap-1.5">
        {children}
        {unsaved ? <ChangedBadge tone="accent">Unsaved</ChangedBadge> : null}
      </div>
    </div>
  )
}
