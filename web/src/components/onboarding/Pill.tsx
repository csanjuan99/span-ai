import type { ReactNode } from "react"

import { cn } from "@/lib/utils"

/** How a pill reads: sealed, worth a look, in the way, neutral, or the accent. */
export type PillTone = "pass" | "hold" | "fail" | "mut" | "acc"

const tones: Record<PillTone, string> = {
  pass: "bg-pass-soft text-pass",
  hold: "bg-hold-soft text-hold",
  fail: "bg-fail-soft text-fail",
  mut: "border border-border bg-muted text-muted-foreground",
  acc: "bg-accent text-primary",
}

/** The small rounded label next to a heading or a row: a state, a count, or "Optional". */
export function Pill({
  tone,
  size = "sm",
  children,
  className,
}: {
  tone: PillTone
  /** `md` is the taller pill of a page header. */
  size?: "sm" | "md"
  children: ReactNode
  className?: string
}) {
  return (
    <span
      className={cn(
        "inline-flex w-fit items-center gap-1.5 rounded-full text-xs font-medium tracking-[0.02em] whitespace-nowrap",
        size === "sm" ? "h-5 px-2" : "h-6 px-2.5",
        tones[tone],
        className
      )}
    >
      {children}
    </span>
  )
}
