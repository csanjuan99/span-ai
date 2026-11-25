"use client"

import { useEffect, useMemo, useState, type ReactNode } from "react"

import { Input } from "@/components/ui/input"
import { reasonText } from "@/lib/guards"
import { foldLog, readGateLog, seconds, type GateLog } from "@/lib/qa"
import { cn } from "@/lib/utils"

/** A text-only action in the log's toolbar. */
function LinkButton({
  onClick,
  strong = false,
  children,
}: {
  onClick: () => void
  strong?: boolean
  children: ReactNode
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        "text-sm text-primary underline-offset-4 hover:underline",
        strong && "font-medium"
      )}
    >
      {children}
    </button>
  )
}

/** A gate's whole log with its own line numbers, quiet stretches folded and a search over all of it. */
export function LogViewer({
  id,
  gate,
  onClose,
}: {
  id: string
  gate: string
  onClose: () => void
}) {
  const [q, setQ] = useState("")
  const [log, setLog] = useState<GateLog | null>(null)
  const [problem, setProblem] = useState("")
  const [opened, setOpened] = useState<ReadonlySet<string>>(new Set())
  const [hit, setHit] = useState(0)

  useEffect(() => {
    let live = true
    const timer = window.setTimeout(() => {
      readGateLog(id, gate, q)
        .then((body) => {
          if (live) {
            setLog(body)
            setProblem("")
          }
        })
        .catch((err: unknown) => {
          if (live) {
            setProblem(reasonText(err, "the log could not be read"))
          }
        })
    }, 200)
    return () => {
      live = false
      window.clearTimeout(timer)
    }
  }, [id, gate, q])

  // Folding a whole log is the costly part and the box re-renders on every keystroke of the
  // search, so the folded rows are rebuilt only when the log or the opened folds change.
  const items = useMemo(
    () => (log === null ? [] : foldLog(log.lines, log.matches, opened)),
    [log, opened]
  )
  const matches = log?.matches ?? []
  const hits = new Set(matches)
  const shown =
    matches.length === 0
      ? 0
      : ((hit % matches.length) + matches.length) % matches.length

  function openFold(id: string) {
    setOpened((now) => new Set(now).add(id))
  }

  function search(value: string) {
    setQ(value)
    setHit(0)
  }

  function revealMatch(step: number) {
    const next = (shown + step + matches.length) % matches.length
    setHit(next)
    const line = matches[next]
    if (line !== undefined) {
      reveal(line)
    }
  }

  function reveal(n: number) {
    const fold = items.find(
      (item) => item.kind === "fold" && n >= item.from && n <= item.to
    )
    if (fold !== undefined && fold.kind === "fold") {
      openFold(fold.id)
    }
    // The fold opens on the next render, so the line is not in the document until the frame after that.
    window.requestAnimationFrame(() => {
      window.requestAnimationFrame(() => {
        document
          .getElementById(`log-line-${n}`)
          ?.scrollIntoView({ block: "center" })
      })
    })
  }

  return (
    <section aria-label={`Log of ${gate}`} className="flex flex-col gap-3">
      <div className="flex flex-wrap items-center gap-3">
        <LinkButton onClick={onClose}>Overview</LinkButton>
        <h2 className="text-lg font-semibold">Log · {gate}</h2>
        {log !== null ? (
          <span className="text-xs text-muted-foreground tabular-nums">
            exit {log.exitCode} · {seconds(log.durationMs)} · {log.totalLines}{" "}
            lines
            {log.bytes > 0 ? ` · ${log.bytes} B` : ""}
          </span>
        ) : null}
      </div>
      <div className="flex flex-wrap items-center gap-2">
        <Input
          aria-label="Search the log"
          placeholder="Search the log"
          value={q}
          onChange={(event) => search(event.target.value)}
          className="max-w-sm"
        />
        {q !== "" ? (
          <LinkButton onClick={() => search("")}>Clear search</LinkButton>
        ) : null}
        {matches.length > 0 ? (
          <>
            <p
              role="status"
              className="text-xs text-muted-foreground tabular-nums"
            >
              {shown + 1} of {matches.length} matches
            </p>
            <LinkButton onClick={() => revealMatch(-1)}>Previous</LinkButton>
            <LinkButton onClick={() => revealMatch(1)}>Next</LinkButton>
          </>
        ) : null}
        {log !== null && log.firstError > 0 ? (
          <LinkButton strong onClick={() => reveal(log.firstError)}>
            Jump to first error: line {log.firstError}
          </LinkButton>
        ) : null}
      </div>
      {problem !== "" ? (
        <p role="alert" className="text-sm text-destructive">
          {problem}
        </p>
      ) : null}
      {log !== null && log.truncated ? (
        <p className="text-sm text-pretty text-hold">
          The start of this log was cut: lines {log.firstLine} to{" "}
          {log.firstLine + log.kept - 1} of {log.totalLines} are kept.
        </p>
      ) : null}
      <div className="overflow-x-auto rounded-[10px] border border-border bg-surface-code py-2 font-mono text-[13px] leading-5">
        {items.map((item) =>
          item.kind === "fold" ? (
            <button
              key={item.id}
              type="button"
              onClick={() => openFold(item.id)}
              className="block min-h-7 w-full bg-muted px-3.5 text-left text-primary"
            >
              Show {item.count} quiet lines ({item.from}–{item.to})
            </button>
          ) : (
            <div
              key={item.line.n}
              id={`log-line-${item.line.n}`}
              className={cn(
                "grid grid-cols-[60px_minmax(0,1fr)] pr-3.5 whitespace-pre",
                item.line.kind === "err"
                  ? "bg-fail-soft text-fail"
                  : hits.has(item.line.n)
                    ? "bg-accent-soft"
                    : "text-code-comment"
              )}
            >
              <span className="pr-2 text-right text-muted-foreground tabular-nums">
                {item.line.n}
              </span>
              <span>{item.line.text}</span>
            </div>
          )
        )}
      </div>
      <p className="text-xs text-muted-foreground">
        Terminal:{" "}
        <span className="font-mono">
          span qa log {id} {gate} --first-error
        </span>
      </p>
    </section>
  )
}
