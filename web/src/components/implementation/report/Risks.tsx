import { levelTone, originLabel } from "@/components/implementation/report/tone"
import { cn } from "cn"
import type { ReportedRisk } from "@/lib/featuredoc"
import type { Risk } from "@/lib/report"

const shown = 4

function Files({ files }: { files: string[] }) {
  if (files.length === 0) {
    return null
  }
  return (
    <ul className="mt-1.5 flex flex-wrap gap-1.5" aria-label="Files">
      {files.slice(0, shown).map((file) => (
        <li key={file} className="rounded-md border border-border px-1.5 py-0.5 font-mono text-xs break-all">
          {file}
        </li>
      ))}
      {files.length > shown ? <li className="px-1 py-0.5 text-xs text-muted-foreground">+{files.length - shown} more</li> : null}
    </ul>
  )
}

/** Every risk Span can see, worst first, each with the files it points at. */
export function Risks({ risks, reported }: { risks: Risk[]; reported: ReportedRisk[] }) {
  const own = reported.filter((item) => !item.none)
  const saidNone = reported.some((item) => item.none)
  return (
    <section aria-label="Risks" className="flex flex-col gap-2">
      <h3 className="text-sm">
        <span className="text-primary">$ </span>risks
        <span className="ml-3 text-xs text-muted-foreground tabular-nums">{risks.length + own.length}</span>
      </h3>
      {risks.length + own.length === 0 ? (
        <p className="text-pretty text-sm text-muted-foreground">
          No signal found{saidNone ? ", and the agent says it looked and found none" : ""}. That is the absence of signals, not proof of safety.
        </p>
      ) : (
        <ul className="flex flex-col gap-2">
          {risks.map((item) => {
            const tone = levelTone[item.level]
            return (
              <li key={item.id} className={cn("rounded-md border p-3", tone.border)}>
                <div className="flex flex-wrap items-baseline gap-x-2.5 gap-y-0.5">
                  <span className={cn("text-xs font-medium uppercase", tone.text)}>{tone.label}</span>
                  <span>{item.title}</span>
                  <span className="ml-auto text-xs text-muted-foreground">{originLabel[item.origin]}</span>
                </div>
                <p className="mt-1 text-pretty text-sm text-muted-foreground">{item.detail}</p>
                <Files files={item.files} />
              </li>
            )
          })}
          {own.map((item) => (
            <li key={item.text} className="rounded-md border border-border p-3">
              <div className="flex flex-wrap items-baseline gap-x-2.5">
                <span className="text-xs font-medium text-primary uppercase">agent</span>
                <span className="text-pretty">{item.text}</span>
                <span className="ml-auto text-xs text-muted-foreground">{originLabel.reported}</span>
              </div>
            </li>
          ))}
        </ul>
      )}
    </section>
  )
}
