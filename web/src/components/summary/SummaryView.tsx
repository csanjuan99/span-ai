import type { ReactNode } from "react"

import type { Summary } from "@/lib/summary"
import { cn } from "@/lib/utils"

const letterTone = {
  A: "bg-pass-soft text-pass",
  M: "bg-accent text-primary",
  D: "bg-fail-soft text-fail",
  R: "bg-hold-soft text-hold",
} as const

const levelTone = {
  high: "text-fail",
  medium: "text-hold",
  low: "text-pass",
} as const

function Block({ name, children }: { name: string; children: ReactNode }) {
  return (
    <section
      aria-label={name}
      className="flex flex-col gap-2 rounded-xl border border-border bg-card p-4 shadow-sm"
    >
      <h2 className="text-sm font-medium text-muted-foreground">{name}</h2>
      {children}
    </section>
  )
}

/** The summary of a finished run: what landed, how risky, what changed, what the checks said. */
export function SummaryView({ summary }: { summary: Summary }) {
  return (
    <div className="flex flex-col gap-4">
      <Block name="Outcome">
        <p className="font-medium text-pretty">{summary.outcome}</p>
        {summary.verdict !== null ? (
          <p className={cn("text-sm", levelTone[summary.verdict.level])}>
            {summary.verdict.headline} · {summary.verdict.level} risk
          </p>
        ) : (
          <p className="text-sm text-muted-foreground">
            Not assessed: there is no diff to read for this run.
          </p>
        )}
      </Block>
      {summary.assessed ? (
        <Block name={`Files changed · ${summary.totals.files}`}>
          <p className="text-xs text-muted-foreground tabular-nums">
            +{summary.totals.added} −{summary.totals.removed}
          </p>
          <ul className="flex flex-col divide-y divide-border">
            {summary.files.map((file) => (
              <li
                key={file.path}
                className="flex items-center gap-3 py-1.5 text-sm"
              >
                <span
                  aria-hidden="true"
                  className={cn(
                    "grid size-5.5 place-items-center rounded-md font-mono text-xs font-semibold",
                    letterTone[file.letter]
                  )}
                >
                  {file.letter}
                </span>
                <span className="min-w-0 flex-1 truncate font-mono text-xs">
                  {file.path}
                </span>
                <span className="text-xs text-muted-foreground tabular-nums">
                  +{file.added} −{file.removed}
                </span>
              </li>
            ))}
          </ul>
        </Block>
      ) : null}
      {summary.tests !== null ? (
        <Block name="Tests">
          <p className="text-sm tabular-nums">
            {summary.tests.passed} passed
            {summary.tests.failed > 0
              ? ` · ${summary.tests.failed} failed`
              : ""}
          </p>
        </Block>
      ) : null}
      {summary.risks.length > 0 ? (
        <Block name={`Risks noted · ${summary.risks.length}`}>
          <ul className="flex flex-col gap-1.5 text-sm">
            {summary.risks.map((risk) => (
              <li key={risk.title} className="flex gap-2">
                <span
                  aria-hidden="true"
                  className={cn("font-mono", levelTone[risk.level])}
                >
                  !
                </span>
                {risk.title}
              </li>
            ))}
          </ul>
        </Block>
      ) : null}
      {summary.links.length > 0 ? (
        <Block name="Links">
          <ul className="flex flex-col gap-1 text-sm">
            {summary.links.map((link) => (
              <li key={link.href}>
                <a
                  className="text-primary underline-offset-4 hover:underline"
                  href={link.href}
                >
                  {link.label}
                </a>
              </li>
            ))}
          </ul>
        </Block>
      ) : null}
    </div>
  )
}
