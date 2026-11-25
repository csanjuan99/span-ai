import { FigureCell } from "@/components/list-controls/FigureCell"
import { Panel } from "@/components/shaping/session-ui"

/** One figure of a summary strip: what it counts, how much, and a caption. */
export interface SummaryCell {
  label: string
  value: string
  caption: string
}

/** A strip of four figures. Two columns on a phone, so each stays readable at 390px. */
export function SummaryCells({
  label,
  cells,
}: {
  label: string
  cells: SummaryCell[]
}) {
  return (
    <Panel
      aria-label={label}
      className="grid grid-cols-2 gap-y-3 sm:grid-cols-4 sm:gap-y-0"
    >
      {cells.map((cell) => (
        <FigureCell
          key={cell.label}
          label={cell.label}
          value={cell.value}
          caption={cell.caption}
          valueClassName="text-base"
          className="border-border p-4 not-first:border-l max-sm:nth-[odd]:border-l-0 sm:px-5 sm:py-4 sm:first:border-l-0"
        />
      ))}
    </Panel>
  )
}
