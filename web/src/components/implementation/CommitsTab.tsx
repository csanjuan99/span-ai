"use client"

import { useEffect, useState } from "react"

import { cn } from "@/lib/utils"
import { DiffCode } from "@/components/implementation/DiffCode"
import { FileStatusChip } from "@/components/files-changed/FileStatusChip"
import { parseDiff, statusOf, type DiffFile } from "@/lib/diff"
import { reasonText } from "@/lib/guards"
import {
  readCommitDiff,
  type CommitDiff,
  type CommitStat,
} from "@/lib/implementation-api"
import { clockOf } from "@/lib/timeline"
import { LineCounts } from "@/components/files-changed/LineCounts"
import { ActionButton } from "@/components/implementation/ActionButton"
import { SideNote } from "@/components/implementation/SideNote"

function Lines({ commit }: { commit: CommitStat }) {
  return commit.renameOnly ? (
    <span className="text-xs text-muted-foreground">renamed</span>
  ) : (
    <span className="tabular-nums">
      <LineCounts added={commit.added} removed={commit.removed} />
    </span>
  )
}

/** What one commit changed and nothing before it: its files and their diffs. */
function CommitDetail({
  runId,
  commit,
  onBack,
}: {
  runId: string
  commit: CommitStat
  onBack: () => void
}) {
  const [diff, setDiff] = useState<CommitDiff | null>(null)
  const [note, setNote] = useState("")
  useEffect(() => {
    let closed = false
    void readCommitDiff(runId, commit.sha)
      .then((value) => {
        if (!closed) {
          setDiff(value)
        }
      })
      .catch((err: unknown) => {
        if (!closed) {
          setNote(reasonText(err, "the commit could not be read"))
        }
      })
    return () => {
      closed = true
    }
  }, [runId, commit.sha])
  const files: DiffFile[] = diff === null ? [] : parseDiff(diff.patch)
  return (
    <section
      aria-label={`Commit ${commit.sha.slice(0, 7)}`}
      className="flex flex-col gap-3 rounded-[14px] border border-border bg-card p-4 shadow-sm sm:p-5"
    >
      <div className="flex flex-wrap items-baseline gap-x-3 gap-y-1">
        <span className="font-mono text-xs">{commit.sha.slice(0, 7)}</span>
        <h2 className="min-w-0 flex-1 text-lg leading-[26px] font-semibold text-balance">
          {commit.subject}
        </h2>
      </div>
      <p className="text-[13px] text-muted-foreground">
        Only this commit. {commit.files} {commit.files === 1 ? "file" : "files"}
        , <Lines commit={commit} />.
        {commit.parent !== "" ? (
          <>
            {" "}
            Parent{" "}
            <span className="font-mono">{commit.parent.slice(0, 7)}</span>.
          </>
        ) : null}
      </p>
      {note !== "" ? (
        <p role="alert" className="text-sm text-destructive">
          {note}
        </p>
      ) : null}
      {diff === null && note === "" ? (
        <p className="text-sm text-muted-foreground">Reading the commit…</p>
      ) : null}
      {diff !== null && diff.truncated ? (
        <p className="text-sm text-hold">
          The patch was cut at 256 KiB. Files past the cut show their counts
          only.
        </p>
      ) : null}
      {diff?.files.map((entry) => {
        const parsed = files.find((file) => file.path === entry.path)
        return (
          <div key={entry.path} className="flex flex-col gap-2">
            <div className="flex items-center gap-2">
              <FileStatusChip status={statusOf(entry.status)} />
              <span className="min-w-0 flex-1 font-mono text-xs [overflow-wrap:anywhere]">
                {entry.path}
              </span>
              <span className="text-xs tabular-nums">
                <LineCounts added={entry.added} removed={entry.removed} />
              </span>
            </div>
            {parsed !== undefined &&
            parsed.hunks.length > 0 &&
            !entry.generated &&
            !entry.large ? (
              <DiffCode file={parsed} />
            ) : null}
          </div>
        )
      })}
      <button
        type="button"
        className="self-start text-[13px] font-medium text-primary hover:underline max-sm:min-h-11"
        onClick={onBack}
      >
        Back to whole branch
      </button>
    </section>
  )
}

/** The commits of the branch, newest first, and what a picked commit changed on its own. */
export function CommitsTab({
  runId,
  commits,
  base,
  branch,
  fileCount,
  added,
  removed,
  onOpenBranchDiff,
  note,
}: {
  runId: string
  commits: CommitStat[]
  base: string
  branch: string
  fileCount: number
  added: number
  removed: number
  onOpenBranchDiff: () => void
  /** Why the list could not be read, when it could not. */
  note: string
}) {
  const [picked, setPicked] = useState("")
  const commit = commits.find((item) => item.sha === picked)
  return (
    <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_300px] xl:items-start">
      <div className="flex min-w-0 flex-col gap-4">
        <section
          aria-label="Commits on the branch"
          className="overflow-hidden rounded-[14px] border border-border bg-card shadow-sm"
        >
          <div className="flex flex-wrap items-center gap-2 px-4 py-4 sm:px-5">
            <h2 className="text-lg leading-[26px] font-semibold">
              Commits on the branch
            </h2>
            <span className="text-xs text-muted-foreground">
              Newest first
              {branch === "" ? (
                ""
              ) : (
                <>
                  {" "}
                  · <span className="font-mono">{branch}</span>
                </>
              )}
            </span>
          </div>
          {note !== "" ? (
            <p className="px-5 pb-4 text-sm text-hold">{note}</p>
          ) : null}
          {commits.length === 0 && note === "" ? (
            <p className="px-5 pb-4 text-sm text-muted-foreground">
              No commit on this run yet.
            </p>
          ) : null}
          {commits.length > 0 ? (
            <div className="grid grid-cols-[72px_minmax(0,1fr)] items-center gap-x-3 border-t border-border bg-muted px-4 py-1.5 text-xs font-medium text-muted-foreground max-sm:hidden sm:grid-cols-[72px_minmax(0,1fr)_56px_72px_110px] sm:px-5">
              <span>Commit</span>
              <span>Subject</span>
              <span>Time</span>
              <span>Files</span>
              <span>Lines</span>
            </div>
          ) : null}
          {commits.map((item) => (
            <button
              key={item.sha}
              type="button"
              aria-pressed={item.sha === picked}
              className={cn(
                "grid min-h-[52px] w-full grid-cols-[72px_minmax(0,1fr)] items-center gap-x-3 gap-y-0.5 border-t border-border px-4 py-2 text-left hover:bg-muted focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none sm:grid-cols-[72px_minmax(0,1fr)_56px_72px_110px] sm:px-5",
                item.sha === picked && "bg-accent shadow-accent-edge"
              )}
              onClick={() => setPicked(item.sha === picked ? "" : item.sha)}
            >
              <span className="font-mono text-xs">{item.sha.slice(0, 7)}</span>
              <span className="min-w-0 text-[13px] leading-5 text-pretty">
                {item.subject}
              </span>
              <span className="text-xs text-muted-foreground tabular-nums max-sm:col-start-2">
                {clockOf(item.time)}
              </span>
              <span className="text-xs text-muted-foreground max-sm:hidden">
                {item.files} {item.files === 1 ? "file" : "files"}
              </span>
              <span className="text-[13px] max-sm:col-start-2">
                <Lines commit={item} />
              </span>
            </button>
          ))}
        </section>
        {commit !== undefined ? (
          <CommitDetail
            runId={runId}
            commit={commit}
            onBack={() => setPicked("")}
          />
        ) : null}
      </div>
      <aside className="flex min-w-0 flex-col gap-4">
        {commit === undefined ? (
          <SideNote
            title="Whole branch"
            rows={[
              {
                term: "Shows",
                body: (
                  <>
                    All {commits.length}{" "}
                    {commits.length === 1 ? "commit" : "commits"} against{" "}
                    {base === "" ? (
                      "its base"
                    ) : (
                      <span className="font-mono">{base}</span>
                    )}
                    .
                  </>
                ),
              },
              {
                term: "Files",
                body: (
                  <>
                    {fileCount} {fileCount === 1 ? "file" : "files"},{" "}
                    <LineCounts added={added} removed={removed} />.
                  </>
                ),
              },
              {
                term: "Next",
                body: "Pick a commit to read only what it changed.",
              },
            ]}
          >
            <ActionButton className="mt-3" onClick={onOpenBranchDiff}>
              Open the whole-branch diff
            </ActionButton>
          </SideNote>
        ) : (
          <SideNote
            title="This commit only"
            rows={[
              {
                term: "Shows",
                body: (
                  <>
                    What changed in{" "}
                    <span className="font-mono">{commit.sha.slice(0, 7)}</span>{" "}
                    and nothing before it.
                  </>
                ),
              },
              {
                term: "Not shown",
                body: `Changes from the other ${commits.length - 1} ${commits.length - 1 === 1 ? "commit" : "commits"}. Switch to the whole branch to see them together.`,
              },
            ]}
          />
        )}
      </aside>
    </div>
  )
}
