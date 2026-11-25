import {
  CheckIcon,
  CircleAlertIcon,
  ClockIcon,
  PauseIcon,
  type LucideIcon,
} from "lucide-react"
import type { ReactNode } from "react"

import { Pill, type PillTone } from "@/components/onboarding/Pill"
import type { QaVerdict } from "@/lib/qa"

const tones: Record<QaVerdict, PillTone> = {
  pass: "pass",
  fail: "fail",
  held: "hold",
  notrun: "mut",
}

const marks: Record<QaVerdict, LucideIcon> = {
  pass: CheckIcon,
  fail: CircleAlertIcon,
  held: PauseIcon,
  notrun: ClockIcon,
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
  const Mark = marks[verdict]
  return (
    <Pill tone={tones[verdict]}>
      <Mark className="size-3.5" aria-hidden="true" />
      {children ?? words[verdict]}
    </Pill>
  )
}
