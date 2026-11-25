import { cn } from "cn"
import type { ReactNode } from "react"

import { DiffTotals } from "@/components/diff-totals/DiffTotals"
import { levelTone } from "@/components/implementation/report/tone"
import type { Proof } from "@/lib/proof"

function Cell({
  label,
  children,
  caption,
}: {
  label: string
  children: ReactNode
  caption: string
}) {
  return (
    <div className="flex min-w-0 flex-col gap-0.5 border-t border-border p-4 first:border-t-0 sm:border-t-0 sm:border-l sm:px-5 sm:first:border-l-0">
      <dt className="text-xs font-medium tracking-wide text-muted-foreground">
        {label}
      </dt>
      <dd className="text-lg leading-[26px] font-semibold tabular-nums">
        {children}
      </dd>
      <dd className="text-xs text-pretty text-muted-foreground">{caption}</dd>
    </div>
  )
}

/** The proof of the run at a glance: how risky, how big, and what the checks said. */
export function ProofSummary({ proof }: { proof: Proof }) {
  const tone = levelTone[proof.verdict.level]
  const { diff, checks } = proof
  const allPassed = checks.total > 0 && checks.passed === checks.total
  // Beside the tests, say how the commands did and what inspect made of it; a failing command is named.
  const commandsLine =
    checks.total === 0
      ? ""
      : `${checks.passed} of ${checks.total} checks passed`
  const caption =
    checks.failing !== ""
      ? checks.failing
      : checks.tests === null
        ? `inspect ${checks.inspect}`
        : [commandsLine, `inspect ${checks.inspect}`]
            .filter((part) => part !== "")
            .join(" · ")
  return (
    <section
      aria-label="Proof of work"
      className="overflow-hidden rounded-[14px] border border-border bg-card shadow-sm"
    >
      <dl className="grid sm:grid-cols-3">
        {diff.files === 0 ? (
          <Cell
            label="Verdict"
            caption="There is no diff to read for this run, so it was not assessed."
          >
            <span className="text-muted-foreground">Not assessed</span>
          </Cell>
        ) : (
          <Cell label="Verdict" caption={proof.verdict.headline}>
            <span className={tone.text}>{tone.label} risk</span>
          </Cell>
        )}
        <Cell label="Diff" caption={diff.kinds}>
          <DiffTotals
            files={diff.files}
            added={diff.added}
            removed={diff.removed}
          />
        </Cell>
        <Cell
          label={checks.tests === null ? "Checks" : "Tests"}
          caption={caption}
        >
          {checks.tests === null ? (
            <span className={cn(checks.total > 0 && !allPassed && "text-fail")}>
              {checks.total === 0
                ? "none ran"
                : `${checks.passed}/${checks.total} passed`}
            </span>
          ) : (
            <span className={cn(checks.tests.failed > 0 && "text-fail")}>
              {checks.tests.passed} passed · {checks.tests.failed} failed
            </span>
          )}
        </Cell>
      </dl>
    </section>
  )
}
