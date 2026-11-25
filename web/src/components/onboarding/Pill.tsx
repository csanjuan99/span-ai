import type { ReactNode } from "react"

import { cn } from "@/lib/utils"

/** How a pill reads: sealed, worth a look, in the way, neutral, or the accent. */
export type PillTone = "pass" | "hold" | "fail" | "mut" | "acc"

const tones: Record<PillTone, string> = {
  pass: "bg-pass-soft text-pass",
  hold: "bg-hold-soft text-hold",
  fail: "bg-fail-soft text-fail",
  mut: "bg-muted text-muted-foreground",
  acc: "bg-accent text-primary",
}

/** The small rounded label next to a heading or a row: a state, a count, or "Optional". */
export function Pill({
  tone,
  children,
  className,
}: {
  tone: PillTone
  children: ReactNode
  className?: string
}) {
  return (
    <span
      className={cn(
        "inline-flex h-5 w-fit items-center gap-1.5 rounded-full px-2 text-xs font-medium whitespace-nowrap",
        tones[tone],
        className
      )}
    >
      {children}
    </span>
  )
}
