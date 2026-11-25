import { levelTone } from "@/components/implementation/report/tone"
import { cn } from "@/lib/utils"
import type { Risk } from "@/lib/report"
import type { RunDetails } from "@/lib/run-details"

const panel =
  "flex flex-col gap-2 rounded-[14px] border border-border bg-card p-4 shadow-sm sm:p-5"

/** The two things worth reading after a run is over: what could still hurt, and how it was run. */
export function FinishedRecord({
  risks,
  details,
}: {
  risks: Risk[]
  details: RunDetails
}) {
  return (
    <>
      {risks.length > 0 ? (
        <section aria-label="Risks noted" className={panel}>
          <h2 className="text-sm font-semibold">
            Risks noted · {risks.length}
          </h2>
          <ul className="flex flex-col gap-1.5 text-sm">
            {risks.map((risk) => (
              <li key={risk.id} className="flex gap-2">
                <span
                  className={cn(
                    "shrink-0 font-mono text-xs",
                    levelTone[risk.level].text
                  )}
                  aria-hidden="true"
                >
                  !
                </span>
                <span className="min-w-0 text-pretty">
                  {risk.title}
                  <span className="text-muted-foreground">
                    {" "}
                    · {levelTone[risk.level].label}
                  </span>
                </span>
              </li>
            ))}
          </ul>
        </section>
      ) : null}
      {details.agents.length > 0 || details.time !== null ? (
        <section aria-label="Run details" className={panel}>
          <h2 className="text-sm font-semibold">Run details</h2>
          <dl className="grid grid-cols-[64px_minmax(0,1fr)] gap-x-2.5 text-[13px]">
            {details.agents.length > 0 ? (
              <>
                <dt className="border-t border-border py-2 text-muted-foreground">
                  Agents
                </dt>
                <dd className="border-t border-border py-2 font-mono text-xs">
                  {details.agents.join(", ")}
                </dd>
              </>
            ) : null}
            {details.time !== null ? (
              <>
                <dt className="border-t border-border py-2 text-muted-foreground">
                  Time
                </dt>
                <dd className="border-t border-border py-2 tabular-nums">
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
