import type { Statement } from "@/lib/types"

function tone(verdict: string): string {
  if (verdict === "stop") {
    return "text-destructive"
  }
  if (verdict === "hold") {
    return "text-hold"
  }
  return "text-pass"
}

/** One sealed statement, printed as a shell record. */
export function Ledger({ statements }: { statements: Statement[] }) {
  if (statements.length === 0) {
    return <p className="text-pretty text-muted-foreground">No record yet. It appears when this stage finishes.</p>
  }
  return (
    <ol className="flex flex-col">
      {statements.map((item) => (
        <li key={`${item.stage}-${item.iteration}-${item.digest ?? item.reason}`} className="border-b border-border py-3">
          <div className="flex flex-wrap items-baseline gap-x-3 gap-y-1">
            <span>
              <span className="text-primary">$ </span>
              {item.stage}
            </span>
            <span className={tone(item.verdict)}>{item.verdict}</span>
            <span className="text-sm text-muted-foreground tabular-nums">iteration {item.iteration}</span>
          </div>
          <p className="mt-2 text-pretty">{item.reason}</p>
          {item.parentDigest ? <p className="mt-2 text-xs text-muted-foreground">parent {item.parentDigest}</p> : null}
          {item.gitSHA ? <p className="text-xs text-muted-foreground">git {item.gitSHA}</p> : null}
          {item.changeURL ? (
            <a className="text-sm text-primary underline-offset-4 hover:underline" href={item.changeURL}>
              {item.changeURL}
            </a>
          ) : null}
          {item.commands.length > 0 ? (
            <ul className="mt-2 flex flex-col gap-1 text-sm">
              {item.commands.map((command) => (
                <li key={command.command} className="tabular-nums">
                  {command.command} · exit {command.exitCode}
                  {command.error ? ` · ${command.error}` : ""}
                </li>
              ))}
            </ul>
          ) : null}
        </li>
      ))}
    </ol>
  )
}
