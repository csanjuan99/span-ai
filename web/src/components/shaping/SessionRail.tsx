import { cn } from "@/lib/utils"

import { Panel } from "@/components/shaping/session-ui"
import type { SessionStep } from "@/lib/shape-session"
import {
  captionText,
  headingText,
  smallTitleText,
} from "@/components/shaping/styles"

const markTone: Record<SessionStep["place"], string> = {
  done: "bg-pass-soft text-pass",
  current: "bg-primary text-primary-foreground",
  waiting: "border border-input bg-muted text-muted-foreground",
}

const markOf: Record<SessionStep["place"], string> = {
  done: "+",
  current: ">",
  waiting: "·",
}

/** The eight steps of a session. A wide screen gets a timeline, a narrow one a bar and the step named. */
export function SessionRail({ steps }: { steps: SessionStep[] }) {
  const at = Math.max(
    steps.findIndex((step) => step.place === "current"),
    0
  )
  return (
    <Panel aria-label="Steps" className="px-4 py-4 sm:px-5">
      <ol className="hidden md:flex">
        {steps.map((step) => (
          <li
            key={step.name}
            aria-current={step.place === "current" ? "step" : undefined}
            className={cn(
              "relative flex min-w-0 flex-1 flex-col pr-2",
              "after:absolute after:top-[11px] after:right-2 after:left-8 after:h-0.5 after:rounded-sm after:bg-border last:after:hidden",
              step.place === "done" && "after:bg-pass"
            )}
          >
            <span
              aria-hidden="true"
              className={cn(
                "relative z-10 mb-1.5 grid size-6 place-items-center rounded-full font-mono text-[13px] font-semibold",
                markTone[step.place]
              )}
            >
              {markOf[step.place]}
            </span>
            <span
              className={cn(
                smallTitleText,
                step.place === "current" && "font-semibold text-primary",
                step.place === "waiting" && "font-normal text-muted-foreground"
              )}
            >
              {step.name}
            </span>
            <span className={captionText}>
              {step.place === "current" ? (
                <span className="inline-flex h-5 items-center rounded-full bg-accent px-2 font-medium tracking-wide text-primary">
                  {step.note}
                </span>
              ) : (
                step.note
              )}
            </span>
          </li>
        ))}
      </ol>
      <div className="md:hidden">
        <div className="flex flex-wrap items-center gap-2.5">
          <span className="text-xs font-medium text-muted-foreground">
            Step {at + 1} of {steps.length}
          </span>
          <span className={headingText}>{steps[at]?.name}</span>
        </div>
        <div className="mt-3 flex gap-[3px]">
          {steps.map((step) => (
            <i
              key={step.name}
              className={cn(
                "block h-1.5 flex-1 rounded-full border border-input bg-muted",
                step.place === "done" && "border-pass bg-pass",
                step.place === "current" && "border-primary bg-primary"
              )}
            />
          ))}
        </div>
      </div>
    </Panel>
  )
}
