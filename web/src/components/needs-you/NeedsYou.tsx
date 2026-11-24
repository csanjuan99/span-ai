import { cn } from "cn"
import { CircleAlert, Hand } from "lucide-react"
import type { ReactNode } from "react"

import type { Needs } from "@/lib/needs-you"

/**
 * The one block above the tabs that holds whatever the run waits on. The header names it in
 * words and color together; the panels the page already has (questions, decision, failed
 * check, conflict, approval) sit inside it, so nothing the person must act on hides in a tab.
 */
export function NeedsYou({ needs, where, children }: { needs: Needs; where: string; children: ReactNode }) {
  const fail = needs.tone === "fail"
  const Icon = fail ? CircleAlert : Hand
  return (
    <section aria-label="Needs you" className="flex flex-col overflow-hidden rounded-xl border border-border bg-card shadow-sm">
      <div
        className={cn(
          "flex flex-wrap items-center gap-x-3 gap-y-1 px-4 py-2.5 text-sm font-medium",
          fail ? "bg-fail-soft text-fail" : "bg-hold-soft text-hold",
        )}
      >
        <Icon className="size-4 shrink-0" aria-hidden="true" />
        <span>{needs.label}</span>
        {where !== "" ? <span className="ml-auto font-mono text-xs font-normal">{where}</span> : null}
      </div>
      <div className="flex flex-col gap-4 p-4">{children}</div>
    </section>
  )
}
