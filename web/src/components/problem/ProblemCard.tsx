import type { ReactNode } from "react"

import { CopyChip } from "@/components/ledger/CopyChip"
import { cn } from "@/lib/utils"

/** The colour of the label: something is wrong (fail), waiting on a person (hold) or just missing (info). */
export type ProblemTone = "fail" | "hold" | "info"

const tones: Record<ProblemTone, string> = {
  fail: "bg-fail-soft text-fail",
  hold: "bg-hold-soft text-hold",
  info: "bg-accent text-primary",
}

/**
 * One way a page can fail to show what it should, in the order a person needs it: what it is,
 * the cause, what is kept, and the one thing to do. `command` is something to copy into a terminal.
 */
export function ProblemCard({
  label,
  tone,
  title,
  cause,
  command,
  kept,
  next,
  children,
}: {
  label: string
  tone: ProblemTone
  title: string
  cause: string
  command?: string
  kept: string
  next: string
  children?: ReactNode
}) {
  return (
    <section
      aria-label={label}
      className="flex flex-col gap-4 rounded-xl border border-border bg-card p-5 shadow-sm"
    >
      <span
        className={cn(
          "w-fit rounded-md px-2 py-0.5 text-xs font-medium",
          tones[tone]
        )}
      >
        {label}
      </span>
      <h2 className="text-lg font-semibold text-balance">{title}</h2>
      <p className="text-sm text-pretty">
        <b>Cause.</b> {cause}
      </p>
      {command !== undefined ? (
        <div className="flex items-center gap-3 rounded-lg bg-surface-code px-3 py-2 font-mono text-sm">
          <span className="text-muted-foreground" aria-hidden="true">
            $
          </span>
          <span className="min-w-0 flex-1 truncate">{command}</span>
          <CopyChip label="command" value={command}>
            Copy
          </CopyChip>
        </div>
      ) : null}
      <dl className="grid gap-3 border-t border-border pt-3 text-sm sm:grid-cols-2">
        <div className="flex flex-col gap-1">
          <dt className="text-xs tracking-wide text-muted-foreground uppercase">
            Kept
          </dt>
          <dd className="text-pretty">
            <span aria-hidden="true" className="mr-1.5 font-mono text-pass">
              +
            </span>
            {kept}
          </dd>
        </div>
        <div className="flex flex-col gap-1">
          <dt className="text-xs tracking-wide text-muted-foreground uppercase">
            Next
          </dt>
          <dd className="text-pretty">{next}</dd>
        </div>
      </dl>
      {children !== undefined ? (
        <div className="flex flex-wrap gap-2">{children}</div>
      ) : null}
    </section>
  )
}
