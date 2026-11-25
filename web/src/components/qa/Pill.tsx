import type { ReactNode } from "react"

import { Pill, type PillTone } from "@/components/onboarding/Pill"
import type { QaVerdict } from "@/lib/qa"

const tones: Record<QaVerdict, PillTone> = {
  pass: "pass",
  fail: "fail",
  held: "hold",
  notrun: "mut",
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
  return <Pill tone={tones[verdict]}>{children ?? words[verdict]}</Pill>
}
