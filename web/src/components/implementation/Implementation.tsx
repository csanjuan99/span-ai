import { DiffFileCard } from "@/components/implementation/DiffFileCard"
import { parseDiff, parseStat, totals, type DiffFile } from "@/lib/diff"

/** Files the stat lists that the patch did not reach, because the patch was cut. */
function missingFromPatch(stat: string, shown: DiffFile[]): DiffFile[] {
  const known = new Set(shown.map((file) => file.path))
  return parseStat(stat)
    .filter((row) => !known.has(row.path))
    .map((row) => ({
      path: row.path,
      oldPath: row.path,
      status: "modified" as const,
      binary: false,
      added: row.plus,
      removed: row.minus,
      hunks: [],
    }))
}

/** How many files open by default. A long patch starts closed so the list stays scannable. */
const openFiles = 3

/** Files changed from base to HEAD as a diff, and the agent transcript while build is open. */
export function Implementation({
  stat,
  patch,
  truncated,
  transcript,
}: {
  stat: string
  patch: string
  truncated: boolean
  transcript: string[]
}) {
  const parsed = parseDiff(patch)
  const files = [...parsed, ...missingFromPatch(stat, parsed)]
  const sum = totals(files)
  return (
    <div className="flex flex-col gap-6">
      <section aria-label="Changes">
        <h2 className="flex flex-wrap items-baseline gap-x-3 text-balance text-sm">
          <span>
            <span className="text-primary">$ </span>git diff
          </span>
          {files.length > 0 ? (
            <span className="text-xs text-muted-foreground tabular-nums">
              {sum.files} {sum.files === 1 ? "file" : "files"} <span className="text-pass">+{sum.added}</span>{" "}
              <span className="text-destructive">−{sum.removed}</span>
            </span>
          ) : null}
        </h2>
        {files.length === 0 ? (
          <p className="mt-2 text-pretty text-muted-foreground">No commit on this run yet.</p>
        ) : (
          <div className="mt-3 flex flex-col gap-2">
            {files.map((file, index) => (
              <DiffFileCard key={file.path} file={file} defaultOpen={index < openFiles} />
            ))}
          </div>
        )}
        {truncated ? (
          <p className="mt-2 text-sm text-hold">The patch was cut at 256 KiB. Files past the cut show their counts only.</p>
        ) : null}
      </section>
      <section aria-label="Transcript">
        <h2 className="text-balance text-sm">
          <span className="text-primary">$ </span>transcript
        </h2>
        {transcript.length === 0 ? (
          <p className="mt-2 text-pretty text-muted-foreground">The agent has not written yet.</p>
        ) : (
          <div className="mt-2 max-h-96 overflow-auto rounded-md border border-border bg-card py-2 font-mono text-xs leading-5">
            {transcript.map((line, index) => (
              <div key={index} className="flex min-w-max gap-3 px-3">
                <span className="w-8 shrink-0 text-right text-muted-foreground/70 tabular-nums select-none">{index + 1}</span>
                <span className="whitespace-pre">{line}</span>
              </div>
            ))}
          </div>
        )}
      </section>
    </div>
  )
}
