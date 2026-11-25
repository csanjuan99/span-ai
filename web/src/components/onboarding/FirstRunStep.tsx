"use client"

import { Button } from "@/components/ui/button"
import { sampleBrief } from "@/components/desk/EmptyDesk"
import { Card, StepHead } from "@/components/onboarding/Frame"
import { DryLook } from "@/components/onboarding/DryLook"

/** Step 3: a sample brief, a look at what each stage does, and the Start button. */
export function FirstRunStep({
  folder,
  onStart,
  onBack,
  onSkip,
}: {
  folder: string
  onStart: () => void
  onBack: () => void
  onSkip: () => void
}) {
  return (
    <div className="flex flex-col gap-6">
      <StepHead
        step="Step 3 of 3"
        title="Your first run"
        lead="Start with a small brief we filled in. Look at what each stage will do first, then press Start."
      />
      <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(0,1.4fr)]">
        <div className="flex flex-col gap-4">
          <Card title="Sample brief" note="Prefilled">
            <p className="font-medium">{sampleBrief.title}</p>
            <textarea
              aria-label="Brief"
              readOnly
              rows={4}
              value={sampleBrief.detail}
              className="rounded-lg border border-border bg-background px-3 py-2 text-sm"
            />
            {folder !== "" ? (
              <p className="text-xs text-muted-foreground">
                Runs in {folder} on a new branch.
              </p>
            ) : null}
          </Card>
          <Card title="When you press Start">
            <dl className="flex flex-col gap-2 text-sm">
              <div>
                <dt className="font-medium">Changes</dt>
                <dd className="text-muted-foreground">
                  Creates a run branch. No file changes until build.
                </dd>
              </div>
              <div>
                <dt className="font-medium">Kept</dt>
                <dd className="text-muted-foreground">
                  Everything you have now. Nothing is overwritten.
                </dd>
              </div>
              <div>
                <dt className="font-medium">Next</dt>
                <dd className="text-muted-foreground">
                  Intake reads the brief. If something is unclear, the run holds
                  and asks you.
                </dd>
              </div>
            </dl>
          </Card>
          <Button onClick={onStart}>Start the first run</Button>
          <div className="flex gap-2">
            <Button variant="outline" onClick={onBack}>
              Back
            </Button>
            <Button variant="ghost" onClick={onSkip}>
              Skip for now
            </Button>
          </div>
        </div>
        <DryLook />
      </div>
    </div>
  )
}
