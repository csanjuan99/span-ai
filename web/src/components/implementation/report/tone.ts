import type { Level, Origin } from "@/lib/report"

/** Text, fill and border classes for each level. The same tones as the ledger verdicts. */
export const levelTone: Record<Level, { text: string; fill: string; border: string; label: string }> = {
  high: { text: "text-destructive", fill: "bg-destructive/10", border: "border-destructive/40", label: "high" },
  medium: { text: "text-hold", fill: "bg-hold/10", border: "border-hold/40", label: "medium" },
  low: { text: "text-pass", fill: "bg-pass/10", border: "border-pass/40", label: "low" },
}

/** How a reader should read where a line came from. */
export const originLabel: Record<Origin, string> = {
  detected: "detected by Span",
  reported: "reported by the agent",
  verified: "verified by a check",
}
