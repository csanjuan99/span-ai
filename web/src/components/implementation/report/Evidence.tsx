import { CircleCheckIcon, CircleXIcon } from "lucide-react"

import { cn } from "cn"
import type { Evidence as EvidenceData } from "@/lib/report"

/** What was run against the change and what the inspector said. Claims without a command are not evidence. */
export function Evidence({ evidence, gaps }: { evidence: EvidenceData; gaps: string[] }) {
  return (
    <section aria-label="Evidence" className="flex flex-col gap-3">
      <h3 className="text-sm">
        <span className="text-primary">$ </span>evidence
      </h3>
      {evidence.commands.length === 0 ? (
        <p className="text-sm text-muted-foreground">No check has run on this change.</p>
      ) : (
        <ul className="flex flex-col gap-1.5">
          {evidence.commands.map((command, index) => (
            <li key={`${command.stage}-${command.command}-${index}`} className="flex flex-wrap items-center gap-x-2.5 rounded-md border border-border bg-muted/40 px-2.5 py-1.5 font-mono text-xs">
              {command.ok ? (
                <CircleCheckIcon aria-label="passed" className="size-3.5 shrink-0 text-pass" />
              ) : (
                <CircleXIcon aria-label="failed" className="size-3.5 shrink-0 text-destructive" />
              )}
              <span className="text-muted-foreground">{command.stage}</span>
              <span className="min-w-0 flex-1 break-all">{command.command}</span>
              <span className={cn("tabular-nums", command.ok ? "text-pass" : "text-destructive")}>exit {command.exitCode}</span>
            </li>
          ))}
        </ul>
      )}
      {evidence.inspect !== null ? (
        <div className="rounded-md border border-border p-3">
          <p className="text-xs text-muted-foreground">inspect · a separate agent that did not write the change</p>
          <p className={cn("mt-1 text-sm", evidence.inspect.verdict === "advance" ? "text-pass" : "text-destructive")}>
            {evidence.inspect.verdict === "advance" ? "passed" : "blocked"}
          </p>
          {evidence.inspect.reason !== "" ? <p className="mt-1 text-pretty text-sm">{evidence.inspect.reason}</p> : null}
        </div>
      ) : null}
      {gaps.length > 0 ? (
        <div className="rounded-md border border-hold/40 bg-hold/10 p-3">
          <p className="text-xs font-medium text-hold uppercase">not checked</p>
          <ul className="mt-1.5 flex flex-col gap-1 text-sm">
            {gaps.map((gap) => (
              <li key={gap} className="text-pretty">
                {gap}
              </li>
            ))}
          </ul>
        </div>
      ) : null}
    </section>
  )
}
