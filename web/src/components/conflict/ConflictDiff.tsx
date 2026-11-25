"use client"

import { useState } from "react"
import { cn } from "cn"

import { diffLines, lineNumbers, type DiffLineKind } from "@/lib/conflict"
import { Button } from "@/components/ui/button"

/** Lines shown before "Show more": enough to read a hunk, short enough to compare groups. */
const cap = 40

const rowClass: Record<DiffLineKind, string> = {
  add: "bg-pass-soft text-pass",
  del: "bg-fail-soft text-fail",
  header: "text-muted-foreground",
  context: "text-muted-foreground",
}

/**
 * The unified diff an option would write, in a code block with the line numbers in the margin.
 * The +/- prefix stays in the text, so the reading never depends on color. Lines in `differs`
 * carry a bar on the left, so a comparison shows where the options part ways.
 */
export function ConflictDiff({
  diff,
  differs,
}: {
  diff: string
  differs?: ReadonlySet<string>
}) {
  const [all, setAll] = useState(false)
  const lines = diffLines(diff)
  if (lines.length === 0) {
    return null
  }
  const numbers = lineNumbers(lines)
  const shown = all ? lines : lines.slice(0, cap)
  return (
    <div className="flex flex-col gap-2">
      <div
        role="region"
        aria-label="Diff"
        tabIndex={0}
        className="max-h-96 overflow-auto rounded-[10px] border border-border bg-muted py-2.5 font-mono text-[13px] leading-5"
      >
        {shown.map((line, at) => (
          <div
            key={at}
            data-differs={differs?.has(line.text) ? "true" : undefined}
            className={cn(
              "grid min-w-max grid-cols-[44px_minmax(0,1fr)] pr-3.5 whitespace-pre",
              rowClass[line.kind],
              differs?.has(line.text) &&
                "shadow-[inset_3px_0_0_0] shadow-primary"
            )}
          >
            <span className="pr-2 text-right text-muted-foreground tabular-nums">
              {numbers[at]}
            </span>
            <span data-kind={line.kind}>{line.text}</span>
          </div>
        ))}
      </div>
      {lines.length > shown.length ? (
        <Button
          type="button"
          variant="ghost"
          size="xs"
          className="self-start"
          onClick={() => setAll(true)}
        >
          Show more
        </Button>
      ) : null}
    </div>
  )
}
