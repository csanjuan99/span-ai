"use client"

import { cn } from "cn"

/** The design's tab strip: a row of equal tabs on a muted track, one selected. */
export function TabStrip<T extends string>({
  label,
  value,
  options,
  onChange,
  className,
}: {
  label: string
  value: T
  options: { value: T; label: string }[]
  onChange: (next: T) => void
  className?: string
}) {
  return (
    <div
      role="tablist"
      aria-label={label}
      className={cn("flex gap-1 rounded-[14px] bg-muted p-1", className)}
    >
      {options.map((option) => (
        <button
          key={option.value}
          type="button"
          role="tab"
          aria-selected={option.value === value}
          onClick={() => onChange(option.value)}
          className={cn(
            "h-11 flex-1 rounded-[10px] px-3 text-[13px] font-medium text-muted-foreground focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none sm:h-9",
            option.value === value &&
              "bg-card font-semibold text-primary shadow-sm"
          )}
        >
          {option.label}
        </button>
      ))}
    </div>
  )
}
