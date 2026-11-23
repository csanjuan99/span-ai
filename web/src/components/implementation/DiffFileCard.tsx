import { ChevronRightIcon } from "lucide-react"

import { cn } from "cn"
import { splitPath, type DiffFile, type DiffLine, type FileStatus } from "@/lib/diff"

const statusMark: Record<FileStatus, { letter: string; label: string; tone: string }> = {
  added: { letter: "A", label: "added", tone: "text-pass" },
  deleted: { letter: "D", label: "deleted", tone: "text-destructive" },
  renamed: { letter: "R", label: "renamed", tone: "text-hold" },
  modified: { letter: "M", label: "modified", tone: "text-primary" },
}

const rowTone: Record<DiffLine["kind"], string> = {
  add: "bg-pass/10",
  del: "bg-destructive/10",
  ctx: "",
  note: "text-muted-foreground italic",
}

const signTone: Record<DiffLine["kind"], string> = {
  add: "text-pass",
  del: "text-destructive",
  ctx: "text-transparent",
  note: "text-transparent",
}

const sign: Record<DiffLine["kind"], string> = { add: "+", del: "-", ctx: " ", note: " " }

/** Five blocks that show the share of additions against removals, like the GitHub file bar. */
function ChangeBar({ added, removed }: { added: number; removed: number }) {
  const total = added + removed
  const green = total === 0 ? 0 : Math.round((added / total) * 5)
  const red = total === 0 ? 0 : 5 - green
  return (
    <span aria-hidden="true" className="inline-flex gap-0.5">
      {Array.from({ length: 5 }, (_, index) => (
        <span
          key={index}
          className={cn("size-2 rounded-[2px]", index < green ? "bg-pass" : index < green + red ? "bg-destructive" : "bg-border")}
        />
      ))}
    </span>
  )
}

/** The `-a,b +c,d` part of a hunk header, without the `@@` fences. */
function rangeOf(header: string): string {
  return /^@@ (.*?) @@/.exec(header)?.[1] ?? header
}

function Gutter({ value }: { value: number | null }) {
  return (
    <span className="w-10 shrink-0 pr-2 text-right text-muted-foreground/70 tabular-nums select-none">{value ?? ""}</span>
  )
}

/** One file of the patch: its path and counts, and the hunks when the patch carried them. */
export function DiffFileCard({ file, defaultOpen }: { file: DiffFile; defaultOpen: boolean }) {
  const mark = statusMark[file.status]
  const { dir, name } = splitPath(file.path)
  const hasBody = file.hunks.length > 0
  return (
    <details open={defaultOpen && hasBody} className="group overflow-hidden rounded-md border border-border bg-card">
      <summary
        className={cn(
          "flex flex-wrap items-center gap-x-3 gap-y-1 bg-muted/40 px-3 py-2 text-sm focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none",
          hasBody ? "cursor-pointer list-none hover:bg-muted/70" : "list-none",
        )}
      >
        <ChevronRightIcon
          aria-hidden="true"
          className={cn("size-3.5 shrink-0 text-muted-foreground transition-transform group-open:rotate-90", !hasBody && "opacity-0")}
        />
        <span className={cn("w-3 text-xs font-medium", mark.tone)} title={mark.label}>
          {mark.letter}
        </span>
        <span className="min-w-0 flex-1 break-all">
          <span className="text-muted-foreground">{dir}</span>
          <span>{name}</span>
          {file.status === "renamed" ? <span className="text-xs text-muted-foreground"> ← {file.oldPath}</span> : null}
        </span>
        {file.binary ? <span className="text-xs text-muted-foreground">binary</span> : null}
        <span className="inline-flex items-center gap-2 text-xs tabular-nums">
          <span className="text-pass">+{file.added}</span>
          <span className="text-destructive">−{file.removed}</span>
          <ChangeBar added={file.added} removed={file.removed} />
        </span>
      </summary>
      {file.hunks.map((hunk) => (
        <div key={hunk.header} className="overflow-x-auto border-t border-border font-mono text-xs leading-5">
          <div className="flex min-w-max gap-2 bg-primary/10 px-3 py-0.5 text-muted-foreground">
            <span className="text-primary">{rangeOf(hunk.header)}</span>
            {hunk.section !== "" ? <span>{hunk.section}</span> : null}
          </div>
          {hunk.lines.map((line, index) => (
            <div key={index} className={cn("flex min-w-max", rowTone[line.kind])}>
              <Gutter value={line.oldNo} />
              <Gutter value={line.newNo} />
              <span className={cn("w-4 shrink-0 text-center select-none", signTone[line.kind])}>{sign[line.kind]}</span>
              <span className="pr-3 whitespace-pre">{line.text}</span>
            </div>
          ))}
        </div>
      ))}
    </details>
  )
}
