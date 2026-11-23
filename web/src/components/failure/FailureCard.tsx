import type { Failure } from "@/lib/failure"
import { attemptLine } from "@/lib/failure"

/**
 * What the check printed, so a person does not have to guess. The tail is short on
 * purpose: the full output is at the log path, and the next build prompt gets the
 * same tail.
 */
export function FailureCard({ failure }: { failure: Failure }) {
  const command = failure.command
  const printed = (command?.output ?? "").trim()
  return (
    <section className="flex flex-col gap-3 border-t border-border pt-4" aria-label="Failing check">
      <div className="flex flex-wrap items-baseline gap-x-3">
        <span className="text-sm text-destructive">{failure.stage} failed</span>
        <span className="text-sm text-muted-foreground tabular-nums">{attemptLine(failure)}</span>
      </div>
      <p className="text-pretty">{failure.reason}</p>
      {failure.inconclusive ? (
        <p className="text-pretty text-sm text-hold">
          The same command passed and failed on this commit, so the result is not attributable to the change. Verify
          before editing.
        </p>
      ) : null}
      {command !== null ? (
        <p className="font-mono text-sm tabular-nums">
          <span className="text-primary" aria-hidden="true">
            ${" "}
          </span>
          {command.command} · exit {command.exitCode}
          {command.error !== undefined && command.error !== "" ? ` · ${command.error}` : ""}
        </p>
      ) : null}
      {printed !== "" ? (
        <pre
          aria-label="Command output"
          className="max-h-64 overflow-y-auto overflow-x-hidden whitespace-pre-wrap break-words border border-border bg-muted/40 p-3 font-mono text-xs leading-5"
        >
          {printed}
        </pre>
      ) : null}
      {command?.logPath !== undefined && command.logPath !== "" ? (
        <p className="text-sm text-muted-foreground tabular-nums select-all">{command.logPath}</p>
      ) : null}
    </section>
  )
}
