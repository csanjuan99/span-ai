import type { ReactNode } from "react"

import { cn } from "@/lib/utils"

/** The card every onboarding section sits in. */
export function Card({
  title,
  note,
  children,
  className,
}: {
  title: string
  note?: string
  children: ReactNode
  className?: string
}) {
  return (
    <section
      aria-label={title}
      className={cn(
        "flex flex-col gap-3 rounded-xl border border-border bg-card p-5 shadow-sm",
        className
      )}
    >
      <div className="flex flex-wrap items-center gap-2">
        <h2 className="text-lg font-semibold">{title}</h2>
        {note !== undefined ? (
          <span className="rounded-md bg-accent px-2 py-0.5 text-xs text-primary">
            {note}
          </span>
        ) : null}
      </div>
      {children}
    </section>
  )
}

/** The step counter, title and lead of one onboarding step. */
export function StepHead({
  step,
  title,
  lead,
}: {
  step: string
  title: string
  lead: string
}) {
  return (
    <header className="flex flex-col gap-2">
      <span className="text-xs text-muted-foreground">{step}</span>
      <h1 className="text-3xl font-semibold text-balance">{title}</h1>
      <p className="text-pretty text-muted-foreground">{lead}</p>
    </header>
  )
}

/** A section of the connect step: a title, an optional badge, a lead line, and the controls. */
export function Panel({
  title,
  badge,
  lead,
  children,
}: {
  title: string
  badge?: ReactNode
  lead?: string
  children: ReactNode
}) {
  return (
    <section
      aria-label={title}
      className="flex flex-col gap-3 rounded-[14px] border border-border bg-card p-4 shadow-sm sm:px-5"
    >
      <div className="flex items-center gap-2">
        <h2 className="text-lg font-semibold">{title}</h2>
        {badge}
      </div>
      {lead !== undefined ? (
        <p className="text-[13px] text-muted-foreground">{lead}</p>
      ) : null}
      {children}
    </section>
  )
}
