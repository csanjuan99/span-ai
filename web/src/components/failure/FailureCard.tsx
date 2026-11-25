import { FileTextIcon, TerminalIcon } from "lucide-react"

import type { Failure } from "@/lib/failure"

/**
 * What the check printed, so a person does not have to guess: the reason as a heading, then the
 * command, its exit code and the tail of its output. The tail is short on purpose: the full
 * output is at the log path, and the next build prompt gets the same tail.
 */
export function FailureCard({ failure }: { failure: Failure }) {
  const command = failure.command
  const printed = (command?.output ?? "").trim()
  return (
    <section className="flex flex-col gap-4" aria-label="Failing check">
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
        <div className="flex flex-col gap-2">
          <div className="flex flex-wrap items-center gap-2">
            <TerminalIcon className="size-4" aria-hidden="true" />
            <span className="text-sm font-semibold">Failing output</span>
            {command.logPath !== undefined && command.logPath !== "" ? (
              <span className="ml-auto inline-flex items-center gap-1.5 text-xs text-muted-foreground select-all">
                <FileTextIcon className="size-3.5" aria-hidden="true" />
                {command.logPath}
              </span>
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
              <div className="px-3.5 text-muted-foreground">{command.error}</div>
            ) : null}
            {printed !== "" ? (
              <pre
                aria-label="Command output"
                className="max-h-64 overflow-y-auto px-3.5 break-words whitespace-pre-wrap text-muted-foreground"
              >
                {printed}
              </pre>
            ) : null}
          </div>
        </div>
      ) : null}
    </section>
  )
}
