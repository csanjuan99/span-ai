import type { ReactNode } from "react"

import { Pill } from "@/components/onboarding/Pill"

/** The small pill that says a value moved: amber for changed from the default, violet for unsaved. */
export function ChangedBadge({
  children,
  tone = "hold",
}: {
  children: ReactNode
  tone?: "hold" | "accent"
}) {
  return <Pill tone={tone === "hold" ? "hold" : "acc"}>{children}</Pill>
}
