import { InfoIcon, TriangleAlertIcon, type LucideIcon } from "lucide-react"
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
  icon,
  title,
  cause,
  command,
  kept,
  next,
  children,
}: {
  label: string
  tone: ProblemTone
  /** The mark in the band; the tone's own when left out. */
  icon?: LucideIcon
  title: ReactNode
  cause: string
  command?: string
  kept: string
  next: string
  children?: ReactNode
}) {
  const Icon = icon ?? (tone === "info" ? InfoIcon : TriangleAlertIcon)
  return (
    <section
      aria-label={label}
      className="overflow-hidden rounded-[14px] border border-border bg-card shadow-sm"
    >
      <div
        className={cn(
          "flex flex-wrap items-center gap-2 px-4 py-2.5 text-xs font-medium md:px-5",
          tones[tone]
        )}
      >
        <Icon className="size-4" aria-hidden="true" />
        <span>{label}</span>
      </div>
      <div className="flex flex-col gap-3 p-6">
        <h2 className="text-lg leading-[26px] font-semibold">{title}</h2>
        <p className="text-sm leading-[22px] text-pretty">
          <b>Cause.</b> {cause}
        </p>
        {command !== undefined ? <CommandLine command={command} /> : null}
        <KeptNext kept={kept} next={next} />
        {children !== undefined ? (
          <div className="flex flex-col gap-2 [&>*]:h-10 [&>*]:w-full [&>*]:rounded-[10px]">
            {children}
          </div>
        ) : null}
      </div>
    </section>
  )
}
