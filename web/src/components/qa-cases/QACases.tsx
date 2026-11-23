import type { QACase } from "@/lib/types"

function tone(result: string): string {
  if (result === "fail") {
    return "text-destructive"
  }
  if (result === "unverified" || result === "") {
    return "text-hold"
  }
  return "text-pass"
}

/** One row per acceptance case. A missing declaration stays unverified and names no file. */
export function QACases({
  skipped,
  lines,
  command,
  exitCode,
}: {
  skipped: boolean
  lines: QACase[]
  command: string
  exitCode: number
}) {
  if (skipped) {
    return <p className="text-pretty text-muted-foreground">This story has no acceptance lines, so the gate was skipped.</p>
  }
  return (
    <div className="flex flex-col gap-3">
      {command !== "" ? (
        <p className="text-sm tabular-nums">
          <span className="text-primary" aria-hidden="true">
            ${" "}
          </span>
          <span>
            {command} · exit {exitCode}
          </span>
        </p>
      ) : null}
      <ul className="flex flex-col">
        {lines.map((line) => (
          <li key={line.id} className="border-b border-border py-3">
            <div className="flex flex-wrap items-baseline gap-x-3">
              <span className="tabular-nums">{line.id}</span>
              <span className={tone(line.result)}>{line.result === "" ? "unverified" : line.result}</span>
            </div>
            <p className="mt-2 text-pretty">{line.text}</p>
            {line.result !== "unverified" && line.file !== "" ? (
              <p className="mt-2 text-sm text-muted-foreground tabular-nums">
                {line.file}:{line.line}
              </p>
            ) : null}
          </li>
        ))}
      </ul>
    </div>
  )
}
