"use client"

import { ChevronLeftIcon, ChevronRightIcon, CopyIcon } from "lucide-react"
import { useState } from "react"

import { cn } from "@/lib/utils"
import { CheckField } from "@/components/list-controls/CheckField"
import { CopyChip } from "@/components/ledger/CopyChip"
import { DiffCode } from "@/components/implementation/DiffCode"
import { FileStatusChip } from "@/components/files-changed/FileStatusChip"
import type { FileFlags } from "@/components/implementation/FilesTable"
import { limitLines } from "@/lib/diff-fold"
import { parseDiff, splitPath, type DiffFile } from "@/lib/diff"
import { readFileDiff, type DecisionEntry } from "@/lib/implementation-api"
import { reasonText } from "@/lib/guards"
import { LineCounts } from "@/components/files-changed/LineCounts"
import { ActionButton } from "@/components/implementation/ActionButton"
import { SideNote } from "@/components/implementation/SideNote"

/** How many lines of a very large file the reader sees when they ask for a part of it. */
const firstLines = 300

type Why = "generated" | "large" | "binary"

function whyHeld(flags: FileFlags | undefined): Why | null {
  if (flags === undefined) {
    return null
  }
  if (flags.binary) {
    return "binary"
  }
  return flags.generated ? "generated" : flags.large ? "large" : null
}

const heldCopy: Record<
  Why,
  { title: string; body: (file: DiffFile) => string }
> = {
  generated: {
    title: "Generated",
    body: (file) =>
      `A tool wrote this file. ${file.added + file.removed} lines changed. Collapsed because nobody reads it line by line.`,
  },
  large: {
    title: "Too large",
    body: (file) =>
      `${file.added + file.removed} lines changed. Showing them all would freeze the page. The first ${firstLines} lines are available.`,
  },
  binary: {
    title: "Binary",
    body: () => "Binary file. A line diff does not apply.",
  },
}

/** The diff of one file, or the card that says why it is not shown and how to see it anyway. */
function FileBody({
  runId,
  file,
  flags,
}: {
  runId: string
  file: DiffFile
  flags: FileFlags | undefined
}) {
  const held = whyHeld(flags)
  const [shown, setShown] = useState<DiffFile | null>(null)
  const [note, setNote] = useState("")
  const [busy, setBusy] = useState(false)
  async function reveal(part: boolean) {
    setBusy(true)
    setNote("")
    try {
      const view = await readFileDiff(runId, file.path, { force: true })
      const parsed = parseDiff(view.patch)[0] ?? { ...file, hunks: [] }
      setShown(part ? limitLines(parsed, firstLines) : parsed)
    } catch (err: unknown) {
      setNote(reasonText(err, "the diff could not be read"))
    } finally {
      setBusy(false)
    }
  }
  if (held === null || shown !== null) {
    const body = shown ?? file
    return body.hunks.length === 0 ? (
      <p className="rounded-[10px] border border-border p-4 text-sm text-muted-foreground">
        {file.status === "renamed"
          ? "Renamed with no content change."
          : "The patch did not carry this file's lines."}
      </p>
    ) : (
      <DiffCode file={body} />
    )
  }
  const copy = heldCopy[held]
  return (
    <div
      role="status"
      className="flex flex-col gap-3 rounded-[10px] border border-border bg-muted p-4"
    >
      <p className="text-sm font-semibold">{copy.title}</p>
      <p className="text-sm text-pretty text-muted-foreground">
        {copy.body(file)}
      </p>
      {note !== "" ? <p className="text-sm text-destructive">{note}</p> : null}
      <div className="flex flex-wrap gap-2">
        {held === "large" ? (
          <ActionButton disabled={busy} onClick={() => void reveal(true)}>
            Show first {firstLines} lines
          </ActionButton>
        ) : null}
        {held !== "binary" ? (
          <ActionButton disabled={busy} onClick={() => void reveal(false)}>
            Show the diff anyway
          </ActionButton>
        ) : null}
      </div>
    </div>
  )
}

/** The diff view: the files to pick from, one file's diff with viewed and previous/next, and why it changed. */
export function DiffTab({
  runId,
  files,
  flags,
  current,
  onSelect,
  onViewed,
  decisions,
}: {
  runId: string
  files: DiffFile[]
  flags: Record<string, FileFlags>
  current: string
  onSelect: (path: string) => void
  onViewed: (path: string, viewed: boolean) => void
  decisions: DecisionEntry[]
}) {
  const index = Math.max(
    0,
    files.findIndex((file) => file.path === current)
  )
  const file = files[index]
  if (file === undefined) {
    return (
      <p className="rounded-[14px] border border-border bg-card p-5 text-sm text-muted-foreground">
        No commit on this run yet, so there is no diff to read.
      </p>
    )
  }
  const { dir, name } = splitPath(file.path)
  const viewed = flags[file.path]?.viewed === true
  const seen = files.filter((item) => flags[item.path]?.viewed === true).length
  const next = files[index + 1]
  const why = decisions.filter((decision) =>
    decision.files.some(
      (path) => path === file.path || file.path.endsWith(`/${path}`)
    )
  )
  return (
    <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_300px] xl:items-start">
      <section
        aria-label="Changes"
        className="flex min-w-0 flex-col gap-3 rounded-[14px] border border-border bg-card p-4 shadow-sm sm:p-5"
      >
        <div className="flex flex-wrap items-center gap-x-3 gap-y-2">
          <FileStatusChip status={file.status} />
          <h2 className="min-w-0 flex-1 font-mono text-[13px] leading-5 [overflow-wrap:anywhere]">
            <span className="text-muted-foreground">{dir}</span>
            {name}
          </h2>
          <span className="text-[13px] tabular-nums">
            <LineCounts added={file.added} removed={file.removed} />
          </span>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <CheckField
            className="inline-flex h-9 items-center gap-2 rounded-lg border border-input px-3 text-[13px] max-sm:h-11"
            checked={viewed}
            label="Viewed"
            onChange={(next) => onViewed(file.path, next)}
          />
          <CopyChip
            label="path"
            value={file.path}
            className="rounded-lg border-input bg-card px-2.5 py-1.5 font-sans text-[13px] font-medium text-foreground"
          >
            <CopyIcon aria-hidden="true" className="size-3.5" />
            Copy path
          </CopyChip>
          <span className="ml-auto flex items-center gap-1 text-xs text-muted-foreground">
            File {index + 1} of {files.length}
            <button
              type="button"
              aria-label="Previous file"
              disabled={index === 0}
              className="grid size-9 place-items-center rounded-lg border border-input disabled:opacity-40 max-sm:size-11"
              onClick={() => onSelect(files[index - 1]?.path ?? file.path)}
            >
              <ChevronLeftIcon aria-hidden="true" className="size-4" />
            </button>
            <button
              type="button"
              aria-label="Next file"
              disabled={next === undefined}
              className="grid size-9 place-items-center rounded-lg border border-input disabled:opacity-40 max-sm:size-11"
              onClick={() => onSelect(next?.path ?? file.path)}
            >
              <ChevronRightIcon aria-hidden="true" className="size-4" />
            </button>
          </span>
        </div>
        <FileBody
          key={file.path}
          runId={runId}
          file={file}
          flags={flags[file.path]}
        />
      </section>
      <aside className="flex min-w-0 flex-col gap-4">
        <SideNote
          title="When you mark it viewed"
          rows={[
            {
              term: "Changes",
              body: `Nothing in the code. ${name} is marked viewed on this run.`,
            },
            {
              term: "Kept",
              body: "The diff, its decisions and your place in the list.",
            },
            {
              term: "Next",
              body:
                next === undefined
                  ? "That was the last file."
                  : `Opens ${splitPath(next.path).name}, file ${index + 2} of ${files.length}.`,
            },
          ]}
        />
        <section
          aria-label="Files"
          className="overflow-hidden rounded-[14px] border border-border bg-card shadow-sm"
        >
          <h3 className="px-4 py-3 text-[13px] font-semibold">
            Files · {seen} of {files.length} viewed
          </h3>
          {files.map((item) => (
            <button
              key={item.path}
              type="button"
              aria-current={item.path === file.path ? "true" : undefined}
              className={cn(
                "grid min-h-11 w-full grid-cols-[24px_minmax(0,1fr)_auto] items-center gap-2 border-t border-border px-4 py-2 text-left text-xs hover:bg-muted focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none",
                item.path === file.path && "bg-accent shadow-accent-edge"
              )}
              onClick={() => onSelect(item.path)}
            >
              <FileStatusChip status={item.status} />
              <span className="min-w-0 truncate font-mono">
                {splitPath(item.path).name}
              </span>
              {flags[item.path]?.viewed === true ? (
                <span className="text-pass">viewed</span>
              ) : (
                <span />
              )}
            </button>
          ))}
        </section>
        {why.length > 0 ? (
          <section
            aria-label="Why this changed"
            className="rounded-[14px] border border-border bg-card p-4 shadow-sm"
          >
            <h3 className="mb-2 text-[13px] font-semibold">Why this changed</h3>
            <ul className="flex flex-col gap-2 text-[13px] leading-5">
              {why.map((decision) => (
                <li key={decision.id} className="flex gap-2">
                  <span className="font-mono text-xs text-muted-foreground">
                    {decision.id}
                  </span>
                  <span className="min-w-0 text-pretty">
                    {decision.what}
                    {decision.assumed ? (
                      <span className="ml-1.5 rounded-full bg-hold-soft px-2 py-0.5 text-xs text-hold">
                        assumed
                      </span>
                    ) : null}
                  </span>
                </li>
              ))}
            </ul>
          </section>
        ) : null}
      </aside>
    </div>
  )
}
