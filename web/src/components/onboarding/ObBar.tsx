"use client"

import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"

const steps = ["What Span does", "Connect", "Your first run"] as const

function Mark({ at, step }: { at: number; step: number }) {
  const glyph = step > at ? "+" : step === at ? ">" : "·"
  return (
    <span
      aria-hidden="true"
      className={cn(
        "grid size-6 place-items-center rounded-full font-mono text-[13px] font-semibold",
        step > at && "bg-pass-soft text-pass",
        step === at && "bg-primary text-primary-foreground",
        step < at && "border border-input bg-muted text-muted-foreground"
      )}
    >
      {glyph}
    </span>
  )
}

/** The bar of the first-run flow: the brand, the three steps and where you are, and Skip setup. */
export function ObBar({ step, onSkip }: { step: number; onSkip: () => void }) {
  return (
    <header className="flex h-14 shrink-0 items-center gap-2 border-b border-border bg-card px-4 sm:gap-6 sm:px-10">
      <div className="flex items-center gap-2.5">
        <span
          aria-hidden="true"
          className="grid size-6 place-items-center rounded-md bg-primary font-mono text-[13px] font-semibold text-primary-foreground"
        >
          S
        </span>
        <span className="hidden text-sm font-semibold sm:inline">Span AI</span>
      </div>
      <ol aria-label="Setup steps" className="mx-auto flex items-center">
        {steps.map((name, index) => {
          const at = index + 1
          return (
            <li
              key={name}
              aria-current={step === at ? "step" : undefined}
              className={cn(
                "flex items-center gap-2 text-[13px] text-muted-foreground",
                index > 0 &&
                  "before:mx-1.5 before:h-0.5 before:w-4 before:rounded-sm before:bg-border sm:before:mx-3 sm:before:w-10",
                index > 0 && step >= at && "before:bg-pass",
                step === at && "font-semibold text-primary",
                step > at && "text-foreground"
              )}
            >
              <Mark at={at} step={step} />
              <span className={cn(step !== at && "hidden sm:inline")}>
                {name}
              </span>
            </li>
          )
        })}
      </ol>
      <Button type="button" size="sm" variant="ghost" onClick={onSkip}>
        Skip setup
      </Button>
    </header>
  )
}
