import { DiffTotals } from "@/components/diff-totals/DiffTotals"
import { splitPath, totals, type DiffFile } from "@/lib/diff"

const letter: Record<DiffFile["status"], string> = { added: "A", modified: "M", deleted: "D", renamed: "R" }

/** The files of a finished change, one line each, so the size of it can be read without opening the diff. */
export function FilesChanged({ files }: { files: DiffFile[] }) {
  if (files.length === 0) {
    return null
  }
  const sum = totals(files)
  return (
    <section aria-label="Files changed" className="flex flex-col gap-2 rounded-xl border border-border bg-card p-4 shadow-sm">
      <h2 className="flex flex-wrap items-baseline gap-x-3 text-sm font-medium">
        Files changed
        <span className="text-xs font-normal text-muted-foreground tabular-nums">
          <DiffTotals files={sum.files} added={sum.added} removed={sum.removed} />
        </span>
      </h2>
      <p className="text-xs text-muted-foreground">A added · M modified · D deleted · R renamed</p>
      <ul className="flex flex-col divide-y divide-border text-sm">
        {files.map((file) => {
          const { dir, name } = splitPath(file.path)
          return (
            <li key={file.path} className="flex items-baseline gap-3 py-1.5">
              <span className="w-4 shrink-0 font-mono text-xs text-muted-foreground" title={file.status}>
                {letter[file.status]}
              </span>
              <span className="min-w-0 flex-1 break-words font-mono text-xs">
                <span className="text-muted-foreground">{dir}</span>
                {name}
                {file.status === "renamed" ? <span className="text-muted-foreground"> · from {file.oldPath}</span> : null}
              </span>
              <span className="shrink-0 font-mono text-xs tabular-nums">
                <span className="text-pass">+{file.added}</span> <span className="text-fail">−{file.removed}</span>
              </span>
            </li>
          )
        })}
      </ul>
    </section>
  )
}
