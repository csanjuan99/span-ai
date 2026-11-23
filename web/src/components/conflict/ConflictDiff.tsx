import { useState } from "react"
import { cn } from "cn"

import { diffLines, type DiffLineKind } from "@/lib/conflict"
import { Button } from "@/components/ui/button"

/** Lines shown before "Show more": enough to read a hunk, short enough to compare cards. */
const cap = 40

const lineClass: Record<DiffLineKind, string> = {
  add: "text-pass",
  del: "text-destructive",
  header: "text-muted-foreground",
  context: "",
}

/**
 * The unified diff an option would write, collapsed by default so the cards stay
 * comparable. The +/- prefix stays in the text, so the reading never depends on color.
 */
export function ConflictDiff({ diff }: { diff: string }) {
  const [open, setOpen] = useState(false)
  const [all, setAll] = useState(false)
  const lines = diffLines(diff)
  if (lines.length === 0) {
    return null
  }
  const shown = all ? lines : lines.slice(0, cap)
  return (
    <div className="flex flex-col gap-2">
      <Button type="button" variant="ghost" size="xs" className="self-start" aria-expanded={open} onClick={() => setOpen(!open)}>
        {open ? "Hide diff" : "Show diff"}
      </Button>
      {open ? (
        <pre
          aria-label="Diff"
          className="max-h-96 overflow-auto border border-border bg-muted/40 p-2 font-mono text-xs leading-5"
        >
          {shown.map((line, at) => (
            <span key={at} data-kind={line.kind} className={cn("block whitespace-pre", lineClass[line.kind])}>
              {line.text}
            </span>
          ))}
        </pre>
      ) : null}
      {open && lines.length > shown.length ? (
        <Button type="button" variant="ghost" size="xs" className="self-start" onClick={() => setAll(true)}>
          Show more
        </Button>
      ) : null}
    </div>
  )
}
