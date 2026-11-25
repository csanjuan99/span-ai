import { cn } from "cn"
import type { ReactNode } from "react"

import type { QaVerdict } from "@/lib/qa"

const tones: Record<QaVerdict, string> = {
  pass: "bg-pass-soft text-pass",
  fail: "bg-fail-soft text-fail",
  held: "bg-hold-soft text-hold",
  notrun: "border border-border bg-muted text-muted-foreground",
}

const words: Record<QaVerdict, string> = {
  pass: "Pass",
  fail: "Fail",
  held: "Held",
  notrun: "Not run",
}

/** A verdict as a pill. A held or not-run gate never wears the pass colour. */
export function VerdictPill({
  verdict,
  children,
}: {
  verdict: QaVerdict
  children?: ReactNode
}) {
  return (
    <span
      className={cn(
        "inline-flex h-5 items-center rounded-full px-2 text-xs font-medium whitespace-nowrap",
        tones[verdict]
      )}
    >
      {children ?? words[verdict]}
    </span>
  )
}
