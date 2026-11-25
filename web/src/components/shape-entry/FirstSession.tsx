"use client"

import { PlayIcon } from "lucide-react"
import type { FormEvent } from "react"

import { IdeaField } from "@/components/shape-entry/IdeaField"
import { Button } from "@/components/ui/button"
import {
  captionText,
  errorText,
  headingText,
  noteText,
  wrapRowWide,
} from "@/components/shaping/styles"

/** An idea a first session can start from, so nobody begins with a blank box. */
export const exampleIdea =
  "Customers should see why a webhook delivery failed and be able to retry it themselves."

const steps = [
  { name: "You describe", does: "A sentence or a page. Rough is fine." },
  {
    name: "The agent interviews",
    does: "It reads code and asks only what blocks.",
  },
  {
    name: "You check the map",
    does: "Rules and examples, each with its source.",
  },
  {
    name: "You seal",
    does: "The story becomes read-only. Then run it or send it to Jira.",
  },
]

/** The page before any session exists: what shaping is, an idea box, and how it goes. */
export function FirstSession({
  idea,
  busy,
  note,
  onIdea,
  onStart,
}: {
  idea: string
  busy: boolean
  note: string
  onIdea: (idea: string) => void
  onStart: () => void
}) {
  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!busy && idea.trim() !== "") {
      onStart()
    }
  }
  return (
    <>
      <section
        aria-label="Start your first session"
        className="flex flex-col items-start gap-4 rounded-[20px] border border-border bg-card p-6 shadow-sm md:gap-6 md:p-12"
      >
        <span className="inline-flex h-6 items-center rounded-full bg-accent px-2.5 text-xs leading-4 font-medium tracking-wide text-primary">
          Shape a story · no sessions yet
        </span>
        <h2 className="max-w-[760px] text-[30px] leading-[34px] font-semibold tracking-tight md:text-[40px] md:leading-[44px]">
          Turn a rough idea into a story you can run
        </h2>
        <p className="max-w-[640px] text-base leading-[26px] text-pretty text-muted-foreground">
          Describe what you want. An agent interviews you, asks only what
          blocks, and shows where every rule comes from. Nothing is created
          until you say so.
        </p>
        <form
          onSubmit={submit}
          className="flex w-full max-w-[720px] flex-col gap-4 md:gap-6"
        >
          <IdeaField
            id="shape-first-idea"
            rows={4}
            value={idea}
            onChange={onIdea}
          />
          <div className={wrapRowWide}>
            <Button
              type="submit"
              className="h-11 w-full rounded-[10px] px-4 md:h-10 md:w-auto"
              disabled={busy || idea.trim() === ""}
            >
              <PlayIcon aria-hidden="true" />
              {busy ? "Thinking…" : "Start interview"}
            </Button>
            <Button
              type="button"
              variant="ghost"
              className="h-11 w-full rounded-[10px] px-4 md:h-10 md:w-auto"
              onClick={() => onIdea(exampleIdea)}
            >
              Try an example idea
            </Button>
          </div>
        </form>
        {note !== "" ? (
          <p role="alert" className={errorText}>
            {note}
          </p>
        ) : null}
        <p className={captionText}>
          What happens next: the agent reads your idea and the code in the repo.
          It writes nothing to your repo or Jira.
        </p>
      </section>
      <div>
        <h2 className="mb-3 text-lg leading-[26px] font-semibold">
          How shaping goes
        </h2>
        <ol className="grid gap-3 md:grid-cols-4 md:gap-4">
          {steps.map((step, index) => (
            <li
              key={step.name}
              className="flex flex-col gap-2 rounded-[14px] border border-border bg-card p-4 shadow-sm"
            >
              <div className="flex items-center gap-2">
                <span
                  aria-hidden="true"
                  className="font-mono font-semibold text-primary"
                >
                  {index + 1}
                </span>
                <span className={headingText}>{step.name}</span>
              </div>
              <p className={noteText}>{step.does}</p>
            </li>
          ))}
        </ol>
      </div>
    </>
  )
}
