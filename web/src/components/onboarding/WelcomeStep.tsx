"use client"

import { Button } from "@/components/ui/button"
import { Card, StepHead } from "@/components/onboarding/Frame"
import { CheckLine } from "@/components/onboarding/CheckLine"

const phases = [
  {
    name: "Frame",
    stages: "intake · shape · plan",
    text: "Span reads the story, narrows what to build and writes a plan. It asks you when the story is unclear.",
  },
  {
    name: "Build",
    stages: "implement",
    text: "An agent edits files and commits on its own branch. Your main branch is not touched here.",
  },
  {
    name: "Prove",
    stages: "verify · review · qa",
    text: "Your checks run, you read the diff, and the behavior is tested against the story.",
  },
  {
    name: "Ship",
    stages: "merge · deploy · close",
    text: "The branch merges, deploys if you set that up, and the story closes in Jira.",
  },
]

const safe = [
  "Every run works on its own branch. Your main branch changes only at merge.",
  "Each stage ends with advance, hold or stop. Span never skips a hold.",
  "You can pause after a stage, rewind, or stop now at any time.",
]

/** Step 1: what Span does and what stays safe. */
export function WelcomeStep({
  onNext,
  onSkip,
}: {
  onNext: () => void
  onSkip: () => void
}) {
  return (
    <div className="flex flex-col gap-6">
      <StepHead
        step="Step 1 of 3 · about 3 minutes"
        title="Span takes a story from idea to merged code"
        lead="Agents do the work. You direct and review. Span holds the run and tells you when it needs a person."
      />
      <div className="grid gap-3 sm:grid-cols-2">
        {phases.map((phase) => (
          <Card key={phase.name} title={phase.name} note={phase.stages}>
            <p className="text-sm text-muted-foreground">{phase.text}</p>
          </Card>
        ))}
      </div>
      <Card title="What stays safe">
        <ul className="flex flex-col gap-2">
          {safe.map((line) => (
            <CheckLine key={line} tone="pass">
              {line}
            </CheckLine>
          ))}
        </ul>
      </Card>
      <div className="flex flex-wrap gap-2">
        <Button onClick={onNext}>Continue: connect your project</Button>
        <Button variant="ghost" onClick={onSkip}>
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
