import { ledgerRows, type LedgerRow } from "@/lib/ledger-rows"
import type { Tone } from "@/lib/ledger"
import type { Statement } from "@/lib/types"
import { cn } from "@/lib/utils"

const mark: Record<Tone, { glyph: string; className: string }> = {
  pass: { glyph: "+", className: "bg-pass-soft text-pass" },
  hold: { glyph: "!", className: "bg-hold-soft text-hold" },
  stop: { glyph: "×", className: "bg-fail-soft text-fail" },
}

function Row({ row }: { row: LedgerRow }) {
  const style = mark[row.tone]
  return (
    <li className="grid grid-cols-[24px_minmax(0,1fr)] items-start gap-x-3 gap-y-1 border-t border-border py-3 first:border-t-0 sm:grid-cols-[24px_96px_minmax(0,1fr)_auto]">
      <span
        aria-hidden="true"
        className={cn(
          "grid size-5.5 place-items-center rounded-full font-mono text-xs font-semibold",
          style.className
        )}
      >
        {style.glyph}
      </span>
      <span className="text-sm leading-[22px] font-medium">{row.stage}</span>
      <span className="col-start-2 text-sm leading-[22px] text-pretty sm:col-start-auto">
        {row.line}
      </span>
      <span className="col-start-2 text-xs leading-[22px] text-muted-foreground tabular-nums sm:col-start-auto">
        {row.clock}
      </span>
    </li>
  )
}

/** The ledger at a glance: one line per stage, with the long evidence left to the records. */
export function LedgerSummary({ statements }: { statements: Statement[] }) {
  const rows = ledgerRows(statements)
  if (rows.length === 0) {
    return (
      <p className="text-pretty text-muted-foreground">
        No record yet. It appears when this stage finishes.
      </p>
    )
  }
  return (
    <ol aria-label="Ledger" className="flex flex-col">
      {rows.map((row) => (
        <Row key={row.stage} row={row} />
      ))}
    </ol>
  )
}
