import { levelTone } from "@/components/implementation/report/tone"
import { cn } from "cn"
import type { Risk } from "@/lib/report"
import type { RunDetails } from "@/lib/run-details"

/** The two things worth reading after a run is over: what could still hurt, and how it was run. */
export function FinishedRecord({ risks, details }: { risks: Risk[]; details: RunDetails }) {
  return (
    <>
      {risks.length > 0 ? (
        <section aria-label="Risks noted" className="flex flex-col gap-2">
          <h2 className="text-sm font-medium">Risks noted · {risks.length}</h2>
          <ul className="flex flex-col gap-1.5 text-sm">
            {risks.map((risk) => (
              <li key={risk.id} className="flex gap-2">
                <span className={cn("shrink-0 font-mono text-xs", levelTone[risk.level].text)} aria-hidden="true">
                  !
                </span>
                <span className="min-w-0 text-pretty">
                  {risk.title}
                  <span className="text-muted-foreground"> · {levelTone[risk.level].label}</span>
                </span>
              </li>
            ))}
          </ul>
        </section>
      ) : null}
      {details.agents.length > 0 || details.time !== null ? (
        <section aria-label="Run details" className="flex flex-col gap-2">
          <h2 className="text-sm font-medium">Run details</h2>
          <dl className="grid gap-x-6 gap-y-1 text-sm sm:grid-cols-[auto_1fr]">
            {details.agents.length > 0 ? (
              <>
                <dt className="text-muted-foreground">Agents</dt>
                <dd className="font-mono text-xs">{details.agents.join(", ")}</dd>
              </>
            ) : null}
            {details.time !== null ? (
              <>
                <dt className="text-muted-foreground">Time</dt>
                <dd className="tabular-nums">
                  {details.time.took}, started {details.time.started}
                </dd>
              </>
            ) : null}
          </dl>
        </section>
      ) : null}
    </>
  )
}
