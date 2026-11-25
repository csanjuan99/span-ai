import type { ReactNode } from "react"

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
    <div className="grid gap-1 border-t border-border pt-3 first:border-t-0 first:pt-0 sm:grid-cols-[12rem_minmax(0,1fr)]">
      <div className="flex flex-col">
        <span className="text-sm font-medium">{label}</span>
        <span className="text-xs text-muted-foreground">
          Default: {fallback}
        </span>
      </div>
      <div className="flex min-w-0 flex-col gap-1.5">
        {children}
        <div className="flex gap-2 empty:hidden">
          {changed ? (
            <span className="rounded-md bg-hold-soft px-1.5 py-0.5 text-xs text-hold">
              Changed
            </span>
          ) : null}
          {unsaved ? (
            <span className="rounded-md bg-accent px-1.5 py-0.5 text-xs text-primary">
              Unsaved
            </span>
          ) : null}
        </div>
      </div>
    </div>
  )
}
