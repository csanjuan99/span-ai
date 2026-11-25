import {
  ShieldAlertIcon,
  ShieldCheckIcon,
  TriangleAlertIcon,
} from "lucide-react"

import { levelTone } from "@/components/implementation/report/tone"
import { cn } from "@/lib/utils"
import type { Report } from "@/lib/report"
import { LineCounts } from "@/components/files-changed/LineCounts"

const icon = {
  high: ShieldAlertIcon,
  medium: TriangleAlertIcon,
  low: ShieldCheckIcon,
} as const

/** The first thing a reader sees: how risky, why, and how hard to read. */
export function Verdict({ report }: { report: Report }) {
  const { verdict, totals, evidence } = report
  const tone = levelTone[verdict.level]
  const Icon = icon[verdict.level]
  const passed = evidence.commands.filter((command) => command.ok).length
  return (
    <div
      className={cn(
        "flex flex-col gap-3 rounded-md border p-4",
        tone.border,
        tone.fill
      )}
    >
      <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
        <Icon aria-hidden="true" className={cn("size-5 shrink-0", tone.text)} />
        <span className={cn("text-base font-medium", tone.text)}>
          {tone.label} risk
        </span>
        <span className="text-pretty">{verdict.headline}</span>
      </div>
      <p className="text-sm text-pretty">{verdict.depth}</p>
      <dl className="flex flex-wrap gap-x-6 gap-y-1 text-xs text-muted-foreground tabular-nums">
        <div className="flex gap-1.5">
          <dt>size</dt>
          <dd className="text-foreground">
            {totals.files} {totals.files === 1 ? "file" : "files"}{" "}
            <LineCounts added={totals.added} removed={totals.removed} />
          </dd>
        </div>
        <div className="flex gap-1.5">
          <dt>checks</dt>
          <dd className="text-foreground">
            {evidence.commands.length === 0
              ? "none ran"
              : `${passed}/${evidence.commands.length} passed`}
          </dd>
        </div>
        <div className="flex gap-1.5">
          <dt>inspect</dt>
          <dd className="text-foreground">
            <span
              className={cn(
                evidence.inspect === null
                  ? ""
                  : evidence.inspect.verdict === "advance"
                    ? "text-pass"
                    : "text-destructive"
              )}
            >
              {evidence.inspect === null
                ? "not yet"
                : evidence.inspect.verdict === "advance"
                  ? "passed"
                  : "blocked"}
            </span>
            {evidence.sentBack > 0 ? (
              <span className="text-hold">
                {" "}
                · sent back {evidence.sentBack}×
              </span>
            ) : null}
          </dd>
        </div>
      </dl>
    </div>
  )
}
