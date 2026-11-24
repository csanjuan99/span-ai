import { cn } from "cn"
import { ShieldCheck } from "lucide-react"
import type { ReactNode } from "react"

import type { Kept } from "@/lib/kept"

/** What stays whatever the person does next, so acting on a held run never feels risky. */
export function SafeAndKept({ kept, className }: { kept: Kept; className?: string }) {
  const rows: ReactNode[] = [`${kept.sealed} ${kept.sealed === 1 ? "stage" : "stages"} sealed`]
  if (kept.files !== null) {
    rows.push(`${kept.files.files} ${kept.files.files === 1 ? "file" : "files"} changed, +${kept.files.plus} −${kept.files.minus}`)
  }
  if (kept.branch !== "") {
    rows.push(
      <span className="min-w-0 break-words">
        branch <span className="font-mono text-xs">{kept.branch}</span>
        {kept.head !== "" ? <span className="font-mono text-xs text-muted-foreground"> at {kept.head}</span> : null}
      </span>,
    )
  }
  return (
    <section aria-label="Safe and kept" className={cn("flex flex-col gap-2 rounded-xl border border-border bg-card p-4 shadow-sm", className)}>
      <h2 className="flex items-center gap-2 text-sm font-medium">
        <ShieldCheck className="size-4 text-pass" aria-hidden="true" />
        Safe and kept
      </h2>
      <ul className="flex flex-col gap-1 text-sm">
        {rows.map((row, index) => (
          <li key={index} className="flex gap-2">
            <span className="font-mono text-pass" aria-hidden="true">
              +
            </span>
            {row}
          </li>
        ))}
      </ul>
      <p className="text-pretty text-xs text-muted-foreground">
        Resuming and leaving a note keep all of this. Restarting a stage runs it again; the earlier attempt stays on disk.
      </p>
    </section>
  )
}
