"use client"

import { PlayIcon, ShieldCheckIcon } from "lucide-react"

import { Button } from "@/components/ui/button"

/** A small story a first run can start from, so nobody begins with a blank page. */
export const sampleBrief = {
  title: "Add a /healthz endpoint",
  text: "Add a /healthz endpoint\n\nReturns the build version and checks the database connection. Answers in under 200 ms.",
  detail:
    "Returns the build version and checks the database connection. Answers in under 200 ms.",
}

const phases = [
  {
    name: "Frame",
    does: "Span reads the story and writes a plan.",
    you: "You answer questions and pick options.",
  },
  {
    name: "Build",
    does: "An agent edits files on its own branch.",
    you: "You can leave. Span holds if something goes wrong.",
  },
  {
    name: "Prove",
    does: "Checks run and the behavior is tested.",
    you: "You read the diff and approve.",
  },
  {
    name: "Ship",
    does: "The branch merges and the story closes.",
    you: "You approve the merge.",
  },
]

/** The desk before any run exists: what Span does, a sample to start with, and how a run goes. */
export function EmptyDesk({
  onSample,
  onStory,
}: {
  onSample: () => void
  onStory: () => void
}) {
  return (
    <div className="flex flex-col gap-6">
      <section
        aria-label="Start your first run"
        className="flex flex-col items-start gap-4 rounded-2xl border border-border bg-card p-6 shadow-sm"
      >
        <span className="rounded-md bg-accent px-2 py-0.5 text-xs font-medium text-primary">
          Desk · no runs yet
        </span>
        <h2 className="max-w-3xl text-3xl font-semibold text-balance">
          Ship the story, not the chore
        </h2>
        <p className="max-w-2xl text-pretty text-muted-foreground">
          Start with a small sample. Span works on its own branch, asks you when
          it needs a person, and shows proof before anything merges.
        </p>
        <div className="flex w-full max-w-2xl flex-col gap-2 rounded-xl bg-muted p-4">
          <span className="text-xs tracking-wide text-muted-foreground uppercase">
            Sample brief
          </span>
          <span className="font-medium">{sampleBrief.title}</span>
          <p className="text-sm text-muted-foreground">{sampleBrief.detail}</p>
        </div>
        <div className="flex flex-wrap gap-2">
          <Button type="button" onClick={onSample}>
            <PlayIcon aria-hidden="true" />
            Start with the sample brief
          </Button>
          <Button type="button" variant="ghost" onClick={onStory}>
            Paste a Jira story
          </Button>
        </div>
        <p className="text-xs text-muted-foreground">
          What happens next: the sample fills the form below. Nothing runs until
          you press Configure and run.
        </p>
      </section>
      <section aria-label="How a run goes" className="flex flex-col gap-3">
        <div className="flex items-baseline gap-2">
          <h2 className="text-lg font-semibold">How a run goes</h2>
          <span className="text-xs text-muted-foreground">
            ten stages, four phases
          </span>
        </div>
        <ol className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
          {phases.map((phase, index) => (
            <li
              key={phase.name}
              className="flex flex-col gap-1.5 rounded-xl border border-border bg-card p-4 shadow-sm"
            >
              <span className="flex items-center gap-2 font-medium">
                <span
                  aria-hidden="true"
                  className="grid size-5.5 place-items-center rounded-full bg-accent font-mono text-xs text-primary"
                >
                  {index + 1}
                </span>
                {phase.name}
              </span>
              <span className="text-sm">{phase.does}</span>
              <span className="text-xs text-muted-foreground">
                <b>You:</b> {phase.you}
              </span>
            </li>
          ))}
        </ol>
        <p className="flex items-center gap-2 text-sm text-muted-foreground">
          <ShieldCheckIcon aria-hidden="true" className="size-4 shrink-0" />
          Runs never touch your main branch until merge. You can pause, rewind
          or stop at any time.
        </p>
      </section>
    </div>
  )
}
