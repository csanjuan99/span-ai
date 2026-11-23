import { failed } from "@/lib/ledger"
import type { StatementCommand } from "@/lib/types"

/** One command the stage ran: its line, its exit, and for a failure what it printed. */
function CommandRow({ command }: { command: StatementCommand }) {
  const bad = failed(command)
  const printed = (command.output ?? "").trim()
  const detail = bad && (printed !== "" || (command.error ?? "") !== "" || (command.logPath ?? "") !== "")
  return (
    <li className="overflow-hidden rounded-md border border-border bg-muted/40">
      <div className="flex flex-wrap items-center gap-x-3 gap-y-1 px-2.5 py-1.5 font-mono text-xs">
        <span className="text-primary" aria-hidden="true">
          $
        </span>
        <span className="min-w-0 flex-1 break-all text-foreground">{command.command}</span>
        {command.band ? <span className="text-muted-foreground">{command.band}</span> : null}
        <span className={bad ? "text-destructive tabular-nums" : "text-pass tabular-nums"}>exit {command.exitCode}</span>
      </div>
      {detail ? (
        <details open className="border-t border-border">
          <summary className="cursor-pointer px-2.5 py-1 text-xs text-muted-foreground hover:text-foreground focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none">
            what it printed
          </summary>
          {command.error ? <p className="px-2.5 pb-1 text-xs text-destructive">{command.error}</p> : null}
          {printed !== "" ? (
            <pre className="max-h-56 overflow-auto px-2.5 pb-2 text-xs leading-5 whitespace-pre-wrap break-words">{printed}</pre>
          ) : null}
          {command.logPath ? <p className="px-2.5 pb-2 text-xs text-muted-foreground break-all">full log: {command.logPath}</p> : null}
        </details>
      ) : null}
    </li>
  )
}

/** The commands of one statement, as terminal lines. */
export function CommandBlock({ commands }: { commands: StatementCommand[] }) {
  if (commands.length === 0) {
    return null
  }
  return (
    <ul className="mt-2 flex flex-col gap-1.5" aria-label="Commands run">
      {commands.map((command) => (
        <CommandRow key={`${command.command}-${command.band ?? ""}`} command={command} />
      ))}
    </ul>
  )
}
