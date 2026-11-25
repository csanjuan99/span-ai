"use client"

import { CheckLine } from "@/components/onboarding/CheckLine"
import { Pill } from "@/components/onboarding/Pill"
import { Button } from "@/components/ui/button"
import { stageGroups } from "@/lib/onboarding"
import { cn } from "@/lib/utils"

const safe = [
  "Every run works on its own branch. Your main branch changes only at merge.",
  "Each stage ends with advance, hold or stop. Span never skips a hold.",
  "You can pause after a stage, rewind, or stop now at any time.",
]

// The example row: a run that has framed the work and is now building.
const sealed = ["intake", "frame"]
const current = "build"

function glyph(stage: string): string {
  return sealed.includes(stage) ? "+" : stage === current ? ">" : "·"
}

/** Step 1: what Span does, the ten stages, and what stays safe. */
export function WelcomeStep({
  onNext,
  onSkip,
}: {
  onNext: () => void
  onSkip: () => void
}) {
  return (
    <div className="mx-auto flex w-full max-w-[1200px] flex-col gap-6 sm:gap-8">
      <div className="flex max-w-[760px] flex-col gap-3">
        <Pill tone="acc">Step 1 of 3 · about 3 minutes</Pill>
        <h1 className="text-[30px] leading-[34px] font-semibold tracking-tight text-balance sm:text-[40px] sm:leading-[44px]">
          Span takes a story from idea to merged code
        </h1>
        <p className="text-base leading-[26px] text-muted-foreground">
          Agents do the work. You direct and review. Span holds the run and
          tells you when it needs a person.
        </p>
      </div>
      <section
        aria-label="The ten stages"
        className="rounded-[14px] border border-border bg-card p-4 shadow-sm sm:p-6"
      >
        <div className="flex flex-wrap gap-2 sm:grid sm:grid-cols-10">
          {stageGroups.map((group) => (
            <div
              key={group.name}
              className={cn(
                "mb-2 hidden border-b-2 border-input pb-1.5 text-xs font-semibold sm:block",
                group.span
              )}
            >
              {group.name}
            </div>
          ))}
          {stageGroups.flatMap((group) =>
            group.stages.map((stage) => (
              <span
                key={stage}
                className={cn(
                  "flex h-8 min-w-0 items-center gap-1.5 rounded-full border border-input bg-card px-2.5 text-xs font-medium whitespace-nowrap",
                  sealed.includes(stage) &&
                    "border-transparent bg-pass-soft text-pass",
                  stage === current &&
                    "border-transparent bg-primary text-primary-foreground",
                  glyph(stage) === "·" && "bg-muted text-muted-foreground"
                )}
              >
                <b aria-hidden="true" className="font-mono font-semibold">
                  {glyph(stage)}
                </b>
                {stage}
              </span>
            ))
          )}
        </div>
        <p className="mt-3 text-xs text-muted-foreground">
          Example: a run that has framed the work and is now building. + sealed
          · &gt; current · · waiting.
        </p>
      </section>
      <div className="grid gap-3 sm:grid-cols-2 sm:gap-4 lg:grid-cols-4">
        {stageGroups.map((group) => (
          <section
            key={group.name}
            className="flex flex-col gap-2 rounded-[14px] border border-border bg-card p-4 shadow-sm"
          >
            <div className="text-lg font-semibold">{group.name}</div>
            <div className="font-mono text-xs text-muted-foreground">
              {group.stages.join(" · ")}
            </div>
            <p className="text-[13px]">{group.text}</p>
          </section>
        ))}
      </div>
      <section className="rounded-[14px] border border-border bg-card px-4 py-4 shadow-sm sm:px-5">
        <div className="mb-2 text-sm font-semibold">What stays safe</div>
        <ul className="flex flex-col">
          {safe.map((line) => (
            <CheckLine key={line} tone="pass">
              {line}
            </CheckLine>
          ))}
        </ul>
      </section>
      <div className="flex flex-wrap items-center gap-3">
        <Button className="h-10" onClick={onNext}>
          Continue: connect your project
        </Button>
        <Button className="h-10" variant="ghost" onClick={onSkip}>
          Skip setup
        </Button>
      </div>
      <p className="text-xs text-muted-foreground">
        Nothing runs until you press Start in step 3. Your progress is saved, so
        you can pick this up from Desk later.
      </p>
    </div>
  )
}
