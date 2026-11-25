import type { ReactNode } from "react"

import { cn } from "@/lib/utils"

/** How a live check came out: fine, worth a look, or in the way. */
export type CheckTone = "pass" | "hold" | "fail"

const marks: Record<CheckTone, { glyph: string; className: string }> = {
  pass: { glyph: "+", className: "bg-pass-soft text-pass" },
  hold: { glyph: "!", className: "bg-hold-soft text-hold" },
  fail: { glyph: "×", className: "bg-fail-soft text-fail" },
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
    <li className="flex items-start gap-3 text-sm">
      <span
        aria-hidden="true"
        className={cn(
          "mt-0.5 grid size-5 shrink-0 place-items-center rounded-full font-mono text-xs font-semibold",
          mark.className
        )}
      >
        {mark.glyph}
      </span>
      <div className="flex min-w-0 flex-col gap-1.5">
        <span className="text-pretty">{children}</span>
        {fix !== undefined ? (
          <div className="flex flex-col gap-1.5 text-muted-foreground">
            {fix}
          </div>
        ) : null}
      </div>
    </li>
  )
}
