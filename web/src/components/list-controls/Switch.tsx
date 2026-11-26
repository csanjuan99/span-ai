"use client"

import { cn } from "@/lib/utils"

/** The design's on/off switch, with a 44px touch area around the 24px track. */
export function Switch({
  label,
  on,
  onChange,
}: {
  label: string
  on: boolean
  onChange: (on: boolean) => void
}) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={on}
      aria-label={label}
      onClick={() => onChange(!on)}
      className={cn(
        "relative h-6 w-10 shrink-0 rounded-full border border-input bg-muted transition-colors",
        "before:absolute before:-inset-x-1 before:-inset-y-2.5",
        "after:absolute after:top-0.5 after:left-0.5 after:size-4.5 after:rounded-full after:bg-muted-foreground after:transition-transform",
        on && "border-primary bg-accent after:translate-x-4 after:bg-primary"
      )}
    />
  )
}
