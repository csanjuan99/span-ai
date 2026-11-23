import { ChevronRightIcon } from "lucide-react"

import { shortHash, toneOf } from "@/lib/ledger"
import { clockOf } from "@/lib/timeline"
import { cn } from "cn"
import type { Cursor, Statement } from "@/lib/types"

const toneText = { pass: "text-pass", hold: "text-hold", stop: "text-destructive" } as const

function Row({ name, value }: { name: string; value: string }) {
  return (
    <div className="flex gap-3">
      <dt className="w-16 shrink-0 text-muted-foreground">{name}</dt>
      <dd className="min-w-0 break-all">{value === "" ? "—" : value}</dd>
    </div>
  )
}

/** Who ran what, on which commit, and when. Closed by default: it is for the day something goes wrong. */
export function Audit({
  cursor,
  statements,
  agents,
}: {
  cursor: Cursor
  statements: Statement[]
  agents: Record<string, string>
}) {
  return (
    <details className="group rounded-md border border-border">
      <summary className="flex cursor-pointer list-none items-center gap-2 px-3 py-2 text-sm hover:bg-muted/50 focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none">
        <ChevronRightIcon aria-hidden="true" className="size-3.5 text-muted-foreground transition-transform group-open:rotate-90" />
        <span>
          <span className="text-primary">$ </span>audit trail
        </span>
        <span className="text-xs text-muted-foreground tabular-nums">{statements.length} sealed records</span>
      </summary>
      <div className="flex flex-col gap-4 border-t border-border p-3 text-xs">
        <dl className="flex flex-col gap-1">
          <Row name="run" value={cursor.id} />
          <Row name="branch" value={cursor.branch} />
          <Row name="base" value={cursor.base} />
          <Row name="head" value={cursor.gitSHA === "" ? "" : shortHash(cursor.gitSHA)} />
          <Row name="weight" value={cursor.weight ?? ""} />
          <Row name="change" value={cursor.changeURL} />
        </dl>
        <ol className="flex flex-col gap-1 font-mono">
          {statements.map((item, index) => {
            const tone = toneOf(item.verdict)
            const agent = agents[item.stage]
            return (
              <li key={`${item.stage}-${index}`} className="flex flex-wrap gap-x-3">
                <span className="w-16 shrink-0 text-muted-foreground tabular-nums">{clockOf(item.time ?? "")}</span>
                <span className="w-14 shrink-0">{item.stage}</span>
                <span className={cn("w-16 shrink-0", toneText[tone])}>{item.verdict}</span>
                <span className="w-16 shrink-0 text-muted-foreground tabular-nums">{item.gitSHA ? shortHash(item.gitSHA) : ""}</span>
                {agent ? <span className="text-muted-foreground">{agent}</span> : null}
              </li>
            )
          })}
        </ol>
      </div>
    </details>
  )
}
