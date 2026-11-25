import type { LucideIcon } from "lucide-react"
import type { ComponentProps, ReactNode } from "react"
import { cn } from "@/lib/utils"

import { Pill as BasePill, type PillTone } from "@/components/onboarding/Pill"
import { Consequences } from "@/components/status-block/Consequences"
import { mutedLabelText } from "@/components/shaping/styles"

/** The color a status carries. A word or a mark always goes with it. */
export type Tone = PillTone

/** The text colors that go with each tone, for marks like + ! ? on the page. */
export const toneText: Record<Tone, string> = {
  acc: "text-primary",
  pass: "text-pass",
  hold: "text-hold",
  fail: "text-fail",
  mut: "text-muted-foreground",
}

/** The main column and the 300px side column every session board lays out. */
export const twoColumns =
  "grid items-start gap-4 lg:grid-cols-[minmax(0,1fr)_300px] lg:gap-6"

/** The main column and the 380px side column of the history, import, export and share pages. */
export const wideColumns = "grid gap-6 lg:grid-cols-[minmax(0,1fr)_380px]"

/** The card every block of a session sits on. */
export function Panel({ className, ...props }: ComponentProps<"section">) {
  return (
    <section
      className={cn(
        "rounded-[14px] border border-border bg-card shadow-sm",
        className
      )}
      {...props}
    />
  )
}

/** A status pill. */
export function Pill({
  tone,
  small = false,
  children,
}: {
  tone: Tone
  small?: boolean
  children: ReactNode
}) {
  return (
    <BasePill tone={tone} size={small ? "sm" : "md"}>
      {children}
    </BasePill>
  )
}

/** The small label above a value. */
export function Eyebrow({ className, ...props }: ComponentProps<"div">) {
  return <div className={cn(mutedLabelText, className)} {...props} />
}

/** A panel that says what pressing its buttons will do, then holds them. */
export function ActionPanel({
  title,
  icon,
  rows,
  children,
}: {
  title: string
  icon: LucideIcon
  rows: { label: string; body: ReactNode }[]
  children: ReactNode
}) {
  return (
    <Panel className="flex flex-col gap-3 p-4">
      <Consequences title={title} icon={icon} rows={rows} />
      {children}
    </Panel>
  )
}

/** The lines of a code or markdown block, on the code surface. */
export function CodeBlock({
  label,
  children,
}: {
  label: string
  children: ReactNode
}) {
  return (
    <div
      role="region"
      aria-label={label}
      tabIndex={0}
      className="max-h-[28rem] overflow-auto rounded-[10px] border border-border bg-surface-code py-2.5 font-mono text-[13px] leading-5 focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none"
    >
      {children}
    </div>
  )
}
