import { cn } from "cn"
import type { ReactNode } from "react"

import { DiffTotals } from "@/components/diff-totals/DiffTotals"
import { levelTone } from "@/components/implementation/report/tone"
import type { Proof } from "@/lib/proof"

function Cell({ label, children, caption }: { label: string; children: ReactNode; caption: string }) {
  return (
    <div className="flex min-w-0 flex-col gap-0.5">
      <dt className="text-xs font-medium text-muted-foreground">{label}</dt>
      <dd className="text-base font-semibold tabular-nums">{children}</dd>
      <dd className="text-pretty text-xs text-muted-foreground">{caption}</dd>
    </div>
  )
}

/** The proof of the run at a glance: how risky, how big, and what the checks said. */
export function ProofSummary({ proof }: { proof: Proof }) {
  const tone = levelTone[proof.verdict.level]
  const { diff, checks } = proof
  const allPassed = checks.total > 0 && checks.passed === checks.total
  return (
    <section aria-label="Proof of work" className="rounded-xl border border-border bg-card p-4 shadow-sm">
      <dl className="grid gap-4 sm:grid-cols-3">
        <Cell label="Verdict" caption={proof.verdict.headline}>
          <span className={tone.text}>{tone.label} risk</span>
        </Cell>
        <Cell label="Diff" caption={diff.kinds}>
          <DiffTotals files={diff.files} added={diff.added} removed={diff.removed} />
        </Cell>
        <Cell
          label="Checks"
          caption={checks.failing !== "" ? checks.failing : `inspect ${checks.inspect}`}
        >
          <span className={cn(checks.total > 0 && !allPassed && "text-fail")}>
            {checks.total === 0 ? "none ran" : `${checks.passed}/${checks.total} passed`}
          </span>
        </Cell>
      </dl>
    </section>
  )
}
