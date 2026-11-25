"use client"

import { stepGlyph } from "@/components/onboarding/glyph"
import { Brand } from "@/components/shell/Brand"
import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"

const steps = ["What Span does", "Connect", "Your first run"] as const

function Mark({ at, step }: { at: number; step: number }) {
  const glyph = stepGlyph(step > at, step === at)
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
        <Brand />
      </div>
      <ol aria-label="Setup steps" className="hidden items-center sm:flex">
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
      <Button
        type="button"
        size="sm"
        variant="ghost"
        className="ml-auto h-11 sm:ml-0 sm:h-8"
        onClick={onSkip}
      >
        Skip setup
      </Button>
    </header>
  )
}

/** On a phone the three steps fold into one line and a bar of three segments. */
export function MobileSteps({ step }: { step: number }) {
  return (
    <div className="sm:hidden">
      <div className="flex items-baseline gap-3">
        <span className="text-xs font-medium tracking-wide text-muted-foreground">
          Step {step} of {steps.length}
        </span>
        <span className="text-sm font-semibold">{steps[step - 1]}</span>
      </div>
      <div className="mt-3 flex gap-[3px]">
        {steps.map((name, index) => {
          const at = index + 1
          return (
            <span
              key={name}
              className="flex flex-1 flex-col items-center gap-0.5"
            >
              <i
                className={cn(
                  "block h-1.5 w-full rounded-full border",
                  step > at && "border-pass bg-pass",
                  step === at && "border-primary bg-primary",
                  step < at && "border-input bg-muted"
                )}
              />
              <b
                aria-hidden="true"
                className={cn(
                  "font-mono text-xs leading-4 font-semibold",
                  step > at && "text-pass",
                  step === at && "text-primary",
                  step < at && "text-muted-foreground"
                )}
              >
                {stepGlyph(step > at, step === at)}
              </b>
            </span>
          )
        })}
      </div>
    </div>
  )
}
