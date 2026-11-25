import type { ReactNode } from "react"

import { cn } from "@/lib/utils"

/** The small pill that says a value moved: amber for changed from the default, violet for unsaved. */
export function ChangedBadge({
  children,
  tone = "hold",
}: {
  children: ReactNode
  tone?: "hold" | "accent"
}) {
  return (
    <span
      className={cn(
        "inline-flex h-5 w-fit items-center rounded-full px-2 text-xs font-medium whitespace-nowrap",
        tone === "hold" ? "bg-hold-soft text-hold" : "bg-accent text-primary"
      )}
    >
      {children}
    </span>
  )
}
