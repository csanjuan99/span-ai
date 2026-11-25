import type { ReactNode } from "react"

import { cn } from "@/lib/utils"

const headClass =
  "px-3 py-1.5 text-left text-xs font-normal text-muted-foreground"
const cellClass = "px-3 py-1.5 tabular-nums"

/** One column of a {@link DataTable}: its title, and whether its figures sit on the right. */
export interface Column {
  label: string
  right?: boolean
}

/** A bordered table that scrolls sideways on a phone. The rows are the children. */
export function DataTable({
  columns,
  minWidth,
  children,
}: {
  columns: Column[]
  minWidth: string
  children: ReactNode
}) {
  return (
    <div className="overflow-x-auto border border-border">
      <table className={cn("w-full text-sm", minWidth)}>
        <thead className="border-b border-border">
          <tr>
            {columns.map((column) => (
              <th
                key={column.label}
                className={cn(headClass, column.right && "text-right")}
              >
                {column.label}
              </th>
            ))}
          </tr>
        </thead>
        <tbody className="divide-y divide-border">{children}</tbody>
      </table>
    </div>
  )
}

/** A body cell. `right` aligns a figure; `className` adds a tone. */
export function Cell({
  right = false,
  className,
  children,
}: {
  right?: boolean
  className?: string
  children: ReactNode
}) {
  return (
    <td className={cn(cellClass, right && "text-right", className)}>
      {children}
    </td>
  )
}

/** The first cell of a row, which names the row. */
export function RowHead({ children }: { children: ReactNode }) {
  return (
    <th scope="row" className={cn(cellClass, "text-left font-normal")}>
      {children}
    </th>
  )
}

/** The tone of a count: `tone` when there is something to see, muted when it is zero. */
export function nonZero(count: number, tone: string): string {
  return count > 0 ? tone : "text-muted-foreground"
}
