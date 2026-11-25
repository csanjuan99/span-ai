"use client"

import { cn } from "@/lib/utils"
import { ExternalLinkIcon, FileTextIcon, TerminalIcon } from "lucide-react"
import { useState } from "react"

import { getRunLog, type RunLog } from "@/lib/api"
import type { Failure } from "@/lib/failure"
import { highlight, type Tone } from "@/lib/output-lines"

const toneClass: Record<Tone, string> = {
  fail: "text-fail",
  name: "text-primary",
  muted: "text-muted-foreground",
  plain: "",
}

/** One line of output, with the failing marker, test name and detail drawn apart. */
function OutputLine({ line }: { line: string }) {
  return (
    <div className="px-3.5 whitespace-pre">
      {highlight(line).map((part, index) => (
        <span key={index} className={cn(toneClass[part.tone])}>
          {part.text}
        </span>
      ))}
    </div>
  )
}

/**
 * What the check printed, so a person does not have to guess: the reason as a heading, then the
 * command, its exit code and the tail of its output. The tail is short on purpose: the full
 * output is at the log path, and the next build prompt gets the same tail.
 */
export function FailureCard({ id, failure }: { id: string; failure: Failure }) {
  const command = failure.command
  const printed = (command?.output ?? "").trim()
  const shown = printed === "" ? [] : printed.split("\n")
  const logPath = command?.logPath ?? ""
  const total = command?.lines ?? 0
  const [log, setLog] = useState<RunLog | null>(null)
  const [problem, setProblem] = useState("")

  async function openLog() {
    setProblem("")
    try {
      setLog(await getRunLog(id, logPath))
    } catch (err: unknown) {
      setProblem(
        err instanceof Error ? err.message : "the log could not be read"
      )
    }
  }
  return (
    <section
      className="flex flex-col gap-4 max-lg:contents"
      aria-label="Failing check"
    >
      <div className="flex flex-col gap-2">
        <h2 className="text-lg leading-[26px] font-semibold text-pretty">
          {failure.reason}
        </h2>
        <p className="text-sm text-pretty text-muted-foreground">
          {failure.stage} ran and failed. The stages before it are sealed and
          kept.
        </p>
      </div>
      {failure.inconclusive ? (
        <p className="text-sm text-pretty text-hold">
          The same command passed and failed on this commit, so the result is
          not attributable to the change. Verify before editing.
        </p>
      ) : null}
      {command !== null ? (
        <div className="flex flex-col gap-2 max-lg:order-1">
          <div className="flex flex-wrap items-center gap-2">
            <TerminalIcon className="size-4" aria-hidden="true" />
            <span className="text-sm font-semibold">Failing output</span>
            {total > shown.length ? (
              <span className="text-xs text-muted-foreground">
                {shown.length} of {total} lines
              </span>
            ) : null}
            {logPath !== "" ? (
              <button
                type="button"
                onClick={() => void openLog()}
                className="ml-auto inline-flex items-center gap-1.5 text-[13px] text-primary underline-offset-4 hover:underline"
              >
                Open full log
                <ExternalLinkIcon className="size-3.5" aria-hidden="true" />
              </button>
            ) : null}
          </div>
          <div className="overflow-x-auto rounded-[10px] border border-border bg-surface-code py-2.5 font-mono text-[13px] leading-5">
            <div className="flex flex-wrap justify-between gap-x-4 px-3.5">
              <span>
                <span className="text-primary" aria-hidden="true">
                  ${" "}
                </span>
                {command.command}
              </span>
              <span className="text-fail">exit {command.exitCode}</span>
            </div>
            {command.error !== undefined && command.error !== "" ? (
              <div className="px-3.5 text-muted-foreground">
                {command.error}
              </div>
            ) : null}
            {shown.length > 0 ? (
              <div
                role="group"
                aria-label="Command output"
                className="max-h-64 overflow-y-auto"
              >
                {shown.map((line, index) => (
                  <OutputLine key={index} line={line} />
                ))}
              </div>
            ) : null}
          </div>
          {problem !== "" ? (
            <p className="text-sm text-pretty text-destructive">{problem}</p>
          ) : null}
          {log !== null ? (
            <div className="flex flex-col gap-1">
              <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                <FileTextIcon className="size-3.5" aria-hidden="true" />
                <span className="select-all">{log.path}</span>
                <span>
                  ·{" "}
                  {log.truncated
                    ? `the last ${log.lines} lines`
                    : `${log.lines} lines`}
                </span>
              </div>
              <pre
                aria-label="Full log"
                className="max-h-96 overflow-auto rounded-[10px] border border-border bg-surface-code px-3.5 py-2.5 font-mono text-[13px] leading-5 whitespace-pre-wrap"
              >
                {log.text}
              </pre>
            </div>
          ) : null}
        </div>
      ) : null}
    </section>
  )
}
