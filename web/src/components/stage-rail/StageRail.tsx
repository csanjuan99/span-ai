import { cn } from "@/lib/utils"

import { stages } from "@/lib/types"

type Place = "done" | "current" | "stopped" | "waiting"

const markTone: Record<Place, string> = {
  done: "bg-pass-soft text-pass",
  current: "bg-primary text-primary-foreground",
  stopped: "border border-fail bg-fail-soft text-fail",
  waiting: "border border-input bg-muted text-muted-foreground",
}

const markOf: Record<Place, string> = {
  done: "+",
  current: ">",
  stopped: ">",
  waiting: "·",
}

const stateWord: Record<string, string> = {
  running: "running",
  hold: "on hold",
  stop: "stopped",
}

const badgeTone: Record<string, string> = {
  running: "bg-accent text-primary",
  waiting: "bg-hold-soft text-hold",
  hold: "bg-hold-soft text-hold",
  stop: "bg-fail-soft text-fail",
}

function Badge({ verdict }: { verdict: string }) {
  return (
    <span
      className={cn(
        "inline-flex h-5 items-center rounded-full px-2 text-xs font-medium tracking-wide whitespace-nowrap",
        badgeTone[verdict]
      )}
    >
      {stateWord[verdict] ?? verdict}
    </span>
  )
}

/**
 * The ten stages. On a wide screen a timeline with a connector between the stages; on a narrow
 * one a segmented bar with the current stage named above it. The mark stays (`+` sealed, `>`
 * current, `·` waiting) so color never carries the state alone, and a sealed stage shows how
 * long it took when that is known.
 */
export function StageRail({
  stage,
  verdict,
  sealed = [],
  durations = {},
}: {
  stage: string
  verdict: string
  sealed?: string[]
  durations?: Record<string, string>
}) {
  const done = new Set(sealed)
  const places = stages.map((name): Place => {
    // A finished run has no current stage: its last one is sealed like the rest.
    if (name === stage && verdict !== "done") {
      return verdict === "stop" ? "stopped" : "current"
    }
    return done.has(name) || verdict === "done" ? "done" : "waiting"
  })
  const at = Math.max(stages.indexOf(stage as (typeof stages)[number]), 0)
  const sealedCount = places.filter((place) => place === "done").length
  const waitingCount = places.filter((place) => place === "waiting").length
  return (
    <section
      aria-label="Stages"
      className="rounded-[14px] border border-border bg-card px-4 py-4 shadow-sm sm:px-5"
    >
      <ol className="hidden md:flex">
        {stages.map((name, index) => {
          const place = places[index] ?? "waiting"
          const current = place === "current" || place === "stopped"
          return (
            <li
              key={name}
              aria-current={current ? "step" : undefined}
              className={cn(
                "relative flex min-w-0 flex-1 flex-col pr-2",
                "after:absolute after:top-[11px] after:right-2 after:left-8 after:h-0.5 after:rounded-sm after:bg-border last:after:hidden",
                place === "done" && "after:bg-pass"
              )}
            >
              <span
                aria-hidden="true"
                className={cn(
                  "relative z-10 mb-1.5 grid size-6 place-items-center rounded-full font-mono text-[13px] font-semibold",
                  markTone[place]
                )}
              >
                {markOf[place]}
              </span>
              <span
                className={cn(
                  "text-[13px] leading-5 font-medium",
                  current && "font-semibold text-primary",
                  place === "waiting" && "font-normal text-muted-foreground"
                )}
              >
                {name}
              </span>
              <span className="text-xs leading-4 text-muted-foreground">
                {current ? (
                  <Badge verdict={verdict} />
                ) : place === "done" ? (
                  durations[name] !== undefined ? (
                    `sealed · ${durations[name]}`
                  ) : (
                    "sealed"
                  )
                ) : (
                  "waiting"
                )}
              </span>
            </li>
          )
        })}
      </ol>
      <div className="md:hidden">
        <div className="flex flex-wrap items-center gap-2.5">
          <span className="text-xs font-medium text-muted-foreground">
            Stage {at + 1} of {stages.length}
          </span>
          <span className="text-lg leading-[26px] font-semibold">
            {stage || stages[0]}
          </span>
          {verdict !== "done" ? <Badge verdict={verdict} /> : null}
        </div>
        <div className="mt-3 flex gap-[3px]">
          {stages.map((name, index) => {
            const place = places[index] ?? "waiting"
            return (
              <span
                key={name}
                className="flex flex-1 flex-col items-center gap-0.5"
              >
                <i
                  className={cn(
                    "block h-1.5 w-full rounded-full border border-input bg-muted",
                    place === "done" && "border-pass bg-pass",
                    place === "current" && "border-primary bg-primary",
                    place === "stopped" && "border-fail bg-fail"
                  )}
                />
                <b
                  aria-hidden="true"
                  className={cn(
                    "font-mono text-xs leading-4 font-semibold text-muted-foreground",
                    place === "done" && "text-pass",
                    place === "current" && "text-primary",
                    place === "stopped" && "text-fail"
                  )}
                >
                  {markOf[place]}
                </b>
              </span>
            )
          })}
        </div>
        <p className="mt-2 text-xs text-muted-foreground">
          {sealedCount} sealed · {verdict === "done" ? 0 : 1} current ·{" "}
          {waitingCount} waiting
        </p>
        <details className="mt-2 border-t border-border">
          <summary className="flex min-h-11 cursor-pointer items-center py-2 text-[13px] select-none">
            All {stages.length} stages
          </summary>
          <ol className="flex flex-col gap-1 pb-1 font-mono text-xs">
            {stages.map((name, index) => (
              <li key={name} className="flex items-center gap-2">
                <span
                  aria-hidden="true"
                  className={cn(
                    "grid size-5 place-items-center rounded-full",
                    markTone[places[index] ?? "waiting"]
                  )}
                >
                  {markOf[places[index] ?? "waiting"]}
                </span>
                {name}
                {durations[name] !== undefined && places[index] === "done" ? (
                  <span className="text-muted-foreground">
                    {durations[name]}
                  </span>
                ) : null}
              </li>
            ))}
          </ol>
        </details>
      </div>
    </section>
  )
}
