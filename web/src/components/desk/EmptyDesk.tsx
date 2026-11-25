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
  folder,
  onSample,
  onStory,
}: {
  /** The folder a run would work in; "." is the folder Span was started in. */
  folder: string
  onSample: () => void
  onStory: () => void
}) {
  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-col gap-1">
        <h1 className="text-2xl leading-8 font-semibold tracking-tight">
          Desk
        </h1>
        <p className="text-[13px] leading-5 text-muted-foreground">
          No runs yet
        </p>
      </div>
      <section
        aria-label="Start your first run"
        className="flex flex-col items-start gap-4 rounded-[20px] border border-border bg-card p-6 shadow-sm md:gap-6 md:p-12"
      >
        <span className="inline-flex h-6 items-center rounded-full bg-accent px-2.5 text-xs leading-4 font-medium tracking-wide text-primary">
          Desk · no runs yet
        </span>
        <h2 className="max-w-[720px] text-[30px] leading-[34px] font-semibold tracking-tight text-balance md:text-[40px] md:leading-[44px]">
          Ship the story, not the chore
        </h2>
        <p className="max-w-[640px] text-base leading-[26px] text-pretty text-muted-foreground">
          Start with a small sample. Span works on its own branch, asks you when
          it needs a person, and shows proof before anything merges.
        </p>
        <div className="flex w-full max-w-[640px] flex-col gap-2 rounded-[14px] bg-muted p-4">
          <span className="text-xs leading-4 font-medium text-muted-foreground">
            Sample brief
          </span>
          <span className="text-sm leading-[22px] font-semibold">
            {sampleBrief.title}
          </span>
          <p className="text-[13px] leading-5 text-muted-foreground">
            {sampleBrief.detail}
          </p>
          <p className="text-[13px] leading-5 text-muted-foreground">
            Runs in{" "}
            <span className="font-mono text-xs">
              {folder === "." ? "the folder Span was started in" : folder}
            </span>
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-3">
          <Button
            type="button"
            className="h-11 rounded-[10px] px-4 md:h-10"
            onClick={onSample}
          >
            <PlayIcon aria-hidden="true" />
            Start with the sample brief
          </Button>
          <Button
            type="button"
            variant="ghost"
            className="h-11 rounded-[10px] px-4 md:h-10"
            onClick={onStory}
          >
            Paste a Jira story
          </Button>
        </div>
        <p className="text-xs leading-4 text-muted-foreground">
          What happens next: Start creates a branch and runs intake. Nothing
          else changes.
        </p>
      </section>
      <section aria-label="How a run goes" className="flex flex-col gap-3">
        <div className="flex items-center gap-2">
          <h2 className="text-lg font-semibold">How a run goes</h2>
          <span className="text-xs leading-4 text-muted-foreground">
            ten stages, four phases
          </span>
        </div>
        <ol className="grid gap-3 md:grid-cols-4 md:gap-4">
          {phases.map((phase, index) => (
            <li
              key={phase.name}
              className="flex flex-col gap-2 rounded-[14px] border border-border bg-card p-4 shadow-sm"
            >
              <span className="flex items-center gap-2 text-lg leading-[26px] font-semibold">
                <span
                  aria-hidden="true"
                  className="font-mono font-semibold text-primary"
                >
                  {index + 1}
                </span>
                {phase.name}
              </span>
              <span className="text-[13px] leading-5">{phase.does}</span>
              <span className="text-xs leading-4 text-muted-foreground">
                {phase.you}
              </span>
            </li>
          ))}
        </ol>
        <p className="flex items-center gap-2 text-[13px] leading-5 text-muted-foreground">
          <ShieldCheckIcon aria-hidden="true" className="size-4 shrink-0" />
          Runs never touch your main branch until merge. You can pause, rewind
          or stop at any time.
        </p>
      </section>
    </div>
  )
}
