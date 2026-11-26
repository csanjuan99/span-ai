"use client"

import { Fragment, type ReactNode } from "react"

import { Cell, DataTable, RowHead } from "@/components/data-table/DataTable"
import { VerdictPill } from "@/components/qa/Pill"
import { Button } from "@/components/ui/button"
import { touchArea } from "@/lib/button-class"
import { caseTime, type QaCase } from "@/lib/qa"
import { cn } from "@/lib/utils"

const columns = [
  { label: "Result" },
  { label: "Case" },
  { label: "File:line" },
  { label: "Time", right: true },
  { label: "New" },
]

const linkClass = cn(
  "text-sm text-primary underline-offset-2 hover:underline",
  touchArea
)

function caseVerdict(result: QaCase["result"]) {
  return result === "pass" ? "pass" : result === "fail" ? "fail" : "held"
}

/** What a case printed, in the design's code block. */
function Output({ lines }: { lines: string[] }) {
  return (
    <pre className="overflow-x-auto rounded-[10px] border border-border bg-surface-code py-2 font-mono text-xs leading-5">
      {lines.map((line, index) => (
        <div key={index} className="px-3.5 whitespace-pre">
          {line}
        </div>
      ))}
    </pre>
  )
}

/** A full-width row under a case. */
function Under({ children }: { children: ReactNode }) {
  return (
    <tr>
      <td colSpan={columns.length} className="px-3 pt-0 pb-3">
        <div className="flex flex-col items-start gap-2">{children}</div>
      </td>
    </tr>
  )
}

/**
 * Q2: one row per case. A failure shows its output under the row; opening a case adds why it
 * failed and offers to run it again.
 */
export function CaseTable({
  cases,
  open,
  onToggle,
  busy,
  onRerun,
}: {
  cases: QaCase[]
  open: string
  onToggle: (name: string) => void
  busy: boolean
  onRerun: (name: string) => void
}) {
  return (
    <DataTable columns={columns} minWidth="min-w-[440px]">
      {cases.map((item) => {
        const opened = open === item.name
        const failed = item.result === "fail"
        return (
          <Fragment key={item.name}>
            <tr>
              <Cell>
                <VerdictPill verdict={caseVerdict(item.result)}>
                  {item.result}
                </VerdictPill>
              </Cell>
              <RowHead>
                <button
                  type="button"
                  aria-expanded={opened}
                  onClick={() => onToggle(item.name)}
                  className={cn(
                    "text-left font-mono text-[13px] break-all",
                    failed && "font-semibold",
                    touchArea
                  )}
                >
                  {item.name}
                </button>
              </RowHead>
              <Cell className="font-mono text-[13px] text-muted-foreground">
                {item.line > 0 ? `${item.file}:${item.line}` : item.file}
              </Cell>
              <Cell right className="text-muted-foreground">
                {caseTime(item.elapsedMs)}
              </Cell>
              <Cell>
                {item.isNew ? (
                  <span className="rounded-md border border-dashed border-border px-1.5 text-[11px] text-muted-foreground">
                    new
                  </span>
                ) : null}
              </Cell>
            </tr>
            {failed || opened ? (
              <Under>
                {opened && item.message !== "" ? (
                  <p className="text-sm text-pretty">{item.message}</p>
                ) : null}
                {item.output.length > 0 ? (
                  <div className="w-full min-w-0">
                    <Output lines={item.output} />
                  </div>
                ) : null}
                {opened ? (
                  <Button
                    type="button"
                    size="sm"
                    variant="outline"
                    disabled={busy}
                    onClick={() => onRerun(item.name)}
                  >
                    Rerun this case
                  </Button>
                ) : (
                  <button
                    type="button"
                    className={linkClass}
                    onClick={() => onToggle(item.name)}
                  >
                    Open this case
                  </button>
                )}
              </Under>
            ) : null}
          </Fragment>
        )
      })}
    </DataTable>
  )
}
