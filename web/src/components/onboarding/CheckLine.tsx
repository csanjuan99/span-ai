import type { ReactNode } from "react"

import { cn } from "@/lib/utils"

/** How a live check came out: fine, worth a look, or in the way. */
export type CheckTone = "pass" | "hold" | "fail"

const marks: Record<CheckTone, { glyph: string; className: string }> = {
  pass: { glyph: "+", className: "text-pass" },
  hold: { glyph: "!", className: "text-hold" },
  fail: { glyph: "×", className: "text-fail" },
}

/** One result of a check, with the exact fix beneath it when there is one. */
export function CheckLine({
  tone,
  children,
  fix,
}: {
  tone: CheckTone
  children: ReactNode
  fix?: ReactNode
}) {
  const mark = marks[tone]
  return (
    <li className="grid grid-cols-[16px_minmax(0,1fr)] gap-2 py-1 text-[13px]">
      <span
        aria-hidden="true"
        className={cn("font-mono font-semibold", mark.className)}
      >
        {mark.glyph}
      </span>
      <div className="min-w-0">
        <span className="text-pretty">{children}</span>
        {fix}
      </div>
    </li>
  )
}
