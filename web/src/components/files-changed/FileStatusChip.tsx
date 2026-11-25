import type { FileStatus } from "@/lib/diff"
import { cn } from "@/lib/utils"

const marks: Record<FileStatus, { letter: string; className: string }> = {
  added: { letter: "A", className: "bg-pass-soft text-pass" },
  modified: { letter: "M", className: "bg-accent text-primary" },
  deleted: { letter: "D", className: "bg-fail-soft text-fail" },
  renamed: { letter: "R", className: "bg-hold-soft text-hold" },
}

/** The letter that says what happened to a file, in the colour of that kind of change. */
export function FileStatusChip({ status }: { status: FileStatus }) {
  const mark = marks[status]
  return (
    <span
      title={status}
      className={cn(
        "grid size-[22px] shrink-0 place-items-center rounded-md font-mono text-xs font-semibold",
        mark.className
      )}
    >
      <span aria-hidden="true">{mark.letter}</span>
      <span className="sr-only">{status}</span>
    </span>
  )
}
