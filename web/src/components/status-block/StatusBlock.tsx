import { cn } from "cn"
import type { LucideIcon } from "lucide-react"
import type { ReactNode } from "react"

/** The header colors of a block above the tabs: a person is needed, something failed, all is well, or the run works. */
export type BlockTone = "hold" | "fail" | "pass" | "info"

const header: Record<BlockTone, string> = {
  hold: "bg-hold-soft text-hold",
  fail: "bg-fail-soft text-fail",
  pass: "bg-pass-soft text-pass",
  info: "bg-accent text-primary",
}

/**
 * The one card every block above the tabs is drawn in: a tinted header band with an icon and a
 * short label, then the body. With an `aside` the body splits in two on a wide screen: what
 * happened on the left, what Span will do and the actions on the right.
 */
export function StatusBlock({
  label,
  tone,
  icon: Icon,
  title,
  where,
  strong = false,
  aside,
  children,
}: {
  label: string
  tone: BlockTone
  icon: LucideIcon
  title: string
  where: string
  /** A block that waits on a person gets the stronger border. */
  strong?: boolean
  aside?: ReactNode
  children: ReactNode
}) {
  return (
    <section
      aria-label={label}
      className={cn(
        "flex flex-col overflow-hidden rounded-[14px] border bg-card shadow-sm",
        strong ? "border-input" : "border-border"
      )}
    >
      <div
        className={cn(
          "flex flex-wrap items-center gap-2 px-4 py-2.5 sm:px-5",
          header[tone]
        )}
      >
        <Icon className="size-4 shrink-0" aria-hidden="true" />
        <span className="text-xs font-medium tracking-wide">{title}</span>
        {where !== "" ? (
          <span className="ml-auto font-mono text-xs">{where}</span>
        ) : null}
      </div>
      <div
        className={cn(
          "flex flex-col gap-4 p-4",
          aside === undefined
            ? "lg:p-6"
            : "lg:grid lg:grid-cols-[minmax(0,1fr)_340px] lg:gap-0 lg:p-0"
        )}
      >
        <div
          className={cn(
            "flex min-w-0 flex-col gap-4",
            aside !== undefined && "max-lg:contents lg:p-6"
          )}
        >
          {children}
        </div>
        {aside !== undefined ? (
          <div className="flex min-w-0 flex-col gap-4 lg:border-l lg:border-border lg:p-6">
            {aside}
          </div>
        ) : null}
      </div>
    </section>
  )
}
