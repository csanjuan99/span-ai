"use client"

import type { ReactNode } from "react"

import { cn } from "@/lib/utils"

/** A rounded button that is either on or off: a filter, a stage or a pane of the QA tab. */
export function PillToggle({
  on,
  onClick,
  className,
  children,
}: {
  on: boolean
  onClick: () => void
  className?: string
  children: ReactNode
}) {
  return (
    <button
      type="button"
      aria-pressed={on}
      onClick={onClick}
      className={cn(
        "inline-flex h-8 items-center rounded-full border border-input bg-card px-3 text-[13px] aria-pressed:border-primary aria-pressed:bg-accent-soft aria-pressed:font-semibold aria-pressed:text-primary",
        className
      )}
    >
      {children}
    </button>
  )
}
