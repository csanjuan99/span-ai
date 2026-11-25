import type { ReactNode } from "react"

import { cn } from "@/lib/utils"

export interface NoteRow {
  term: string
  body: ReactNode
  /** Replaces the muted body style, for a value that reads as code. */
  bodyClassName?: string
}

/** A small card of the side column: a title and the rows that say what it covers. */
export function SideNote({
  title,
  rows,
  children,
}: {
  title: string
  rows: NoteRow[]
  children?: ReactNode
}) {
  return (
    <section
      aria-label={title}
      className="rounded-[14px] border border-border bg-card p-4 shadow-sm"
    >
      <h3 className="mb-2 text-[13px] font-semibold">{title}</h3>
      <dl className="flex flex-col gap-1.5 text-[13px] leading-5">
        {rows.map((row) => (
          <div key={row.term}>
            <dt className="inline font-medium">{row.term} </dt>
            <dd
              className={cn(
                "inline",
                row.bodyClassName ?? "text-muted-foreground"
              )}
            >
              {row.body}
            </dd>
          </div>
        ))}
      </dl>
      {children}
    </section>
  )
}
