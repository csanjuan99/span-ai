import type { ReactNode } from "react"

import { KeptNext } from "@/components/problem/KeptNext"
import { CommandLine } from "@/components/problem/CommandLine"
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
      {command !== undefined ? <CommandLine command={command} /> : null}
      <KeptNext kept={kept} next={next} />
      {children !== undefined ? (
        <div className="flex flex-wrap gap-2">{children}</div>
      ) : null}
    </section>
  )
}
