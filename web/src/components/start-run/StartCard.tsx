import type { ReactNode } from "react"

import { Panel } from "@/components/shaping/session-ui"
import { hintText } from "@/components/shaping/styles"

/** One numbered step of the start form: where, what, how. */
export function StartCard({
  step,
  title,
  hint,
  badge,
  children,
}: {
  step: number
  title: string
  hint: string
  /** A mark beside the title, such as "Came from Shape a story". */
  badge?: ReactNode
  children: ReactNode
}) {
  return (
    <Panel
      aria-label={title}
      className="flex flex-col gap-3 p-4 md:p-5"
    >
      <header className="flex flex-col gap-1">
        <h2 className="flex flex-wrap items-center gap-2 text-lg leading-[26px] font-semibold">
          <span
            aria-hidden="true"
            className="text-xs font-medium text-muted-foreground"
          >
            {step}
          </span>
          {title}
          {badge}
        </h2>
        <p className={hintText}>{hint}</p>
      </header>
      {children}
    </Panel>
  )
}
