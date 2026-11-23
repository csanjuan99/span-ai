import { useState, type ReactNode } from "react"

import { Button } from "@/components/ui/button"
import { activeStep, doneCount, type SetupStep, type StepId } from "@/lib/setup"
import { cn } from "@/lib/utils"

const hiddenKey = "span.setup.hidden.v1"

function wasHidden(): boolean {
  try {
    return window.localStorage.getItem(hiddenKey) === "1"
  } catch {
    return false
  }
}

function remember(hidden: boolean): void {
  try {
    if (hidden) {
      window.localStorage.setItem(hiddenKey, "1")
    } else {
      window.localStorage.removeItem(hiddenKey)
    }
  } catch {
    // Without storage the band simply comes back on the next load.
  }
}

/**
 * What a first run still needs, read from the desk's own state. The first open step is
 * expanded; any step can be opened. It never blocks the form below, and it collapses to one
 * line that stays reachable.
 */
export function Setup({
  steps,
  bodies,
  onGo,
}: {
  steps: SetupStep[]
  /** Extra content a step shows when it is the open one. */
  bodies: Partial<Record<StepId, ReactNode>>
  /** Moves the person to the field a step is about. */
  onGo: (id: StepId) => void
}) {
  const [hidden, setHidden] = useState(wasHidden)
  // null follows the first open step; "none" is a person closing every step on purpose.
  const [opened, setOpened] = useState<StepId | "none" | null>(null)
  const done = doneCount(steps)
  const next = activeStep(steps)
  const open = opened === "none" ? null : (opened ?? next?.id ?? null)

  function toggleBand(value: boolean) {
    setHidden(value)
    remember(value)
  }

  if (hidden) {
    return (
      <section aria-label="Setup" className="flex flex-wrap items-center gap-3 border-b border-border pb-3 text-sm">
        <h2 className="text-balance">
          <span className="text-primary">$ </span>setup <span className="tabular-nums text-muted-foreground">{done} of {steps.length}</span>
        </h2>
        <Button type="button" variant="link" size="xs" onClick={() => toggleBand(false)}>
          Show setup
        </Button>
      </section>
    )
  }

  return (
    <section aria-label="Setup" className="flex flex-col border border-border">
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-border px-3 py-2">
        <h2 className="text-balance text-sm">
          <span className="text-primary">$ </span>setup <span className="tabular-nums text-muted-foreground">{done} of {steps.length}</span>
        </h2>
        <Button type="button" variant="link" size="xs" onClick={() => toggleBand(true)}>
          Hide setup
        </Button>
      </div>
      <ol className="flex flex-col divide-y divide-border">
        {steps.map((step) => {
          const isOpen = open === step.id
          const body = bodies[step.id]
          const current = next?.id === step.id
          return (
            <li key={step.id} className="flex flex-col gap-2 px-3 py-2">
              <div className="flex flex-wrap items-baseline gap-x-3 gap-y-1">
                <span aria-hidden className={cn("tabular-nums", step.done ? "text-pass" : current ? "text-primary" : "text-muted-foreground")}>
                  {step.done ? "[x]" : "[ ]"}
                </span>
                <button
                  type="button"
                  aria-expanded={isOpen}
                  className={cn("text-left underline-offset-4 hover:underline", current ? "text-primary" : "text-foreground")}
                  onClick={() => setOpened(isOpen ? "none" : step.id)}
                >
                  <span className="sr-only">{step.done ? "Done: " : "To do: "}</span>
                  {step.title}
                </button>
                <span className="text-pretty text-sm text-muted-foreground">{step.detail}</span>
              </div>
              {isOpen ? (
                <div className="flex flex-col gap-3 pl-8">
                  {body}
                  {step.id === "folder" || step.id === "agent" || step.id === "story" ? (
                    <Button type="button" variant="secondary" size="sm" className="self-start" onClick={() => onGo(step.id)}>
                      {step.id === "folder" ? "Go to the folder field" : step.id === "agent" ? "Go to the agent" : "Go to the story"}
                    </Button>
                  ) : null}
                </div>
              ) : null}
            </li>
          )
        })}
      </ol>
    </section>
  )
}
