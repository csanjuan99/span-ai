"use client"

import { ArrowRightIcon } from "lucide-react"

import { FileStatusChip } from "@/components/files-changed/FileStatusChip"
import { splitPath, type DiffFile } from "@/lib/diff"
import { LineCounts } from "@/components/files-changed/LineCounts"

/**
 * The files of a finished change, one row each, so the size of it can be read without opening
 * the diff. When `onOpenDiff` is given the panel ends with a way to the full diff.
 */
export function FilesChanged({
  files,
  onOpenDiff,
}: {
  files: DiffFile[]
  onOpenDiff?: () => void
}) {
  if (files.length === 0) {
    return null
  }
  return (
    <section
      aria-label="Files changed"
      className="overflow-hidden rounded-[14px] border border-border bg-card shadow-sm"
    >
      <div className="px-4 pt-4 pb-3 sm:px-5">
        <h2 className="text-sm leading-[22px] font-semibold">Files changed</h2>
        <p className="text-xs text-muted-foreground">
          A added · M modified · D deleted · R renamed
        </p>
      </div>
      <ul>
        {files.map((file) => {
          const { dir, name } = splitPath(file.path)
          return (
            <li
              key={file.path}
              className="grid grid-cols-[24px_minmax(0,1fr)] items-center gap-x-3 gap-y-0.5 border-t border-border px-4 py-2.5 sm:grid-cols-[24px_minmax(0,1fr)_auto] sm:px-5"
            >
              <FileStatusChip status={file.status} />
              <span className="font-mono text-xs leading-[18px] [overflow-wrap:anywhere]">
                <span className="text-muted-foreground">{dir}</span>
                {name}
              </span>
              <span className="col-start-2 text-xs tabular-nums sm:col-start-auto">
                {file.status === "renamed" ? (
                  <span className="text-muted-foreground">
                    renamed from {file.oldPath}
                  </span>
                ) : (
                  <>
                    <LineCounts added={file.added} removed={file.removed} />
                  </>
                )}
              </span>
            </li>
          )
        })}
      </ul>
      {onOpenDiff !== undefined ? (
        <div className="border-t border-border px-4 py-3 sm:px-5">
          <button
            type="button"
            onClick={onOpenDiff}
            className="inline-flex items-center gap-1.5 text-[13px] text-primary hover:underline"
          >
            Open full diff
            <ArrowRightIcon className="size-3.5" aria-hidden="true" />
          </button>
        </div>
      ) : null}
    </section>
  )
}
