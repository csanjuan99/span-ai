"use client"

import { useState } from "react"

import { cn } from "@/lib/utils"
import { FilterChips } from "@/components/list-controls/FilterChips"
import { CheckField } from "@/components/list-controls/CheckField"
import { FileStatusChip } from "@/components/files-changed/FileStatusChip"
import type { DiffFile } from "@/lib/diff"
import { purposeLabels } from "@/lib/overview"
import { kindOf, type Kind } from "@/lib/report"
import { LineCounts } from "@/components/files-changed/LineCounts"

/** What the server knows about a file beyond its patch: whether the person viewed it, and what kind of file it is. */
export interface FileFlags {
  viewed: boolean
  generated: boolean
  large: boolean
  binary: boolean
}

/** The files of the change with a filter by purpose and a viewed box per file. */
export function FilesTable({
  files,
  flags,
  onOpen,
  onViewed,
  current = "",
}: {
  files: DiffFile[]
  flags: Record<string, FileFlags>
  onOpen: (path: string) => void
  onViewed: (path: string, viewed: boolean) => void
  /** The file open in the diff, highlighted. */
  current?: string
}) {
  const [filter, setFilter] = useState<Kind | "all">("all")
  const kinds = (
    [
      "source",
      "test",
      "docs",
      "config",
      "deps",
      "ci",
      "generated",
      "other",
    ] as const
  ).filter((kind) => files.some((file) => kindOf(file.path) === kind))
  const shown = files.filter(
    (file) => filter === "all" || kindOf(file.path) === filter
  )
  const viewed = files.filter(
    (file) => flags[file.path]?.viewed === true
  ).length
  return (
    <section
      aria-label="Files changed"
      className="overflow-hidden rounded-[14px] border border-border bg-card shadow-sm"
    >
      <div className="flex flex-wrap items-center gap-2 px-4 py-4 sm:px-5">
        <h2 className="text-lg leading-[26px] font-semibold">Files changed</h2>
        <span className="text-xs text-muted-foreground">
          A added · M modified · D deleted · R renamed
        </span>
        <span className="ml-auto text-xs">
          {viewed} of {files.length} viewed
        </span>
      </div>
      {kinds.length > 1 ? (
        <div className="px-4 pb-3">
          <FilterChips
            label="Filter files"
            value={filter}
            onChange={setFilter}
            chips={(["all", ...kinds] as const).map((kind) => ({
              key: kind,
              label: kind === "all" ? "All" : purposeLabels[kind],
              count:
                kind === "all"
                  ? files.length
                  : files.filter((file) => kindOf(file.path) === kind).length,
            }))}
          />
        </div>
      ) : null}
      <div className="grid grid-cols-[28px_minmax(0,1fr)_44px] items-center gap-3 border-t border-border bg-muted px-4 py-1.5 text-xs font-medium text-muted-foreground sm:grid-cols-[28px_minmax(0,1fr)_150px_44px] sm:px-5">
        <span />
        <span>Path</span>
        <span className="max-sm:hidden">Lines</span>
        <span>Viewed</span>
      </div>
      {shown.map((file) => (
        <div
          key={file.path}
          className={cn(
            "grid min-h-[60px] grid-cols-[28px_minmax(0,1fr)_44px] items-center gap-x-3 border-t border-border px-4 py-2 sm:min-h-[52px] sm:grid-cols-[28px_minmax(0,1fr)_150px_44px] sm:px-5",
            current === file.path && "bg-accent shadow-accent-edge"
          )}
        >
          <FileStatusChip status={file.status} />
          <button
            type="button"
            className="min-w-0 text-left font-mono text-xs leading-[18px] [overflow-wrap:anywhere] hover:underline focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none"
            onClick={() => onOpen(file.path)}
          >
            {file.path}
            {file.status === "renamed" ? (
              <span className="block font-sans text-xs text-muted-foreground">
                renamed from <span className="font-mono">{file.oldPath}</span>
              </span>
            ) : null}
          </button>
          <span className="text-[13px] tabular-nums max-sm:col-start-2 max-sm:row-start-2">
            {file.status === "renamed" && file.added + file.removed === 0 ? (
              <span className="text-xs text-muted-foreground">
                renamed, no content change
              </span>
            ) : file.binary ? (
              <span className="text-xs text-muted-foreground">binary</span>
            ) : (
              <>
                <LineCounts added={file.added} removed={file.removed} />
              </>
            )}
          </span>
          <CheckField
            className="inline-flex size-11 items-center justify-center max-sm:col-start-3 max-sm:row-span-2 max-sm:row-start-1 sm:-my-0"
            inputClassName="size-5 accent-primary"
            checked={flags[file.path]?.viewed === true}
            ariaLabel={`Viewed ${file.path}`}
            onChange={(viewed) => onViewed(file.path, viewed)}
          />
        </div>
      ))}
    </section>
  )
}
