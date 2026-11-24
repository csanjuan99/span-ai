import { cn } from "cn"
import type { LucideIcon } from "lucide-react"
import type { ReactNode } from "react"

/** The header colors of a block above the tabs: a person is needed, something failed, or all is well. */
export type BlockTone = "hold" | "fail" | "pass" | "info"

const header: Record<BlockTone, string> = {
  hold: "bg-hold-soft text-hold",
  fail: "bg-fail-soft text-fail",
  pass: "bg-pass-soft text-pass",
  info: "bg-accent text-accent-foreground",
}

/** The one card every block above the tabs is drawn in: a tinted header with an icon, then the body. */
export function StatusBlock({
  label,
  tone,
  icon: Icon,
  title,
  where,
  children,
}: {
  label: string
  tone: BlockTone
  icon: LucideIcon
  title: string
  where: string
  children: ReactNode
}) {
  return (
    <section aria-label={label} className="flex flex-col overflow-hidden rounded-xl border border-border bg-card shadow-sm">
      <div className={cn("flex flex-wrap items-center gap-x-3 gap-y-1 px-4 py-2.5 text-sm font-medium", header[tone])}>
        <Icon className="size-4 shrink-0" aria-hidden="true" />
        <span>{title}</span>
        {where !== "" ? <span className="ml-auto font-mono text-xs font-normal">{where}</span> : null}
      </div>
      <div className="flex flex-col gap-4 p-4">{children}</div>
    </section>
  )
}
