"use client"

import { useState, type FormEvent } from "react"

import { Button } from "@/components/ui/button"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"

/** The questions a waiting run needs before frame starts again. */
export function Waiting(props: {
  questions: string[]
  note: string
  onAnswer: (answers: { question: string; body: string }[]) => Promise<unknown>
  onReload: () => Promise<unknown>
}) {
  const [bodies, setBodies] = useState<string[]>(() =>
    props.questions.map(() => "")
  )
  const [busy, setBusy] = useState(false)
  // The engine refuses an empty answer, so every question is required: say so before the request fails.
  const open = bodies.filter((body) => body.trim() === "").length

  async function answer(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setBusy(true)
    try {
      await props.onAnswer(
        props.questions.map((question, index) => ({
          question,
          body: bodies[index] ?? "",
        }))
      )
    } finally {
      setBusy(false)
    }
  }

  async function reload() {
    setBusy(true)
    try {
      await props.onReload()
    } finally {
      setBusy(false)
    }
  }

  return (
    <form
      className="flex flex-col gap-3"
      onSubmit={(event) => void answer(event)}
    >
      <p className="text-sm text-pretty text-muted-foreground">
        Frame reads the story in the language it is written and carries what it
        already decides into this run. Answer here when the story does not say
        it.
      </p>
      {props.questions.map((question, index) => (
        <div className="flex flex-col gap-2" key={`${index}-${question}`}>
          <div className="flex flex-wrap items-baseline justify-between gap-x-3">
            <Label htmlFor={`answer-${index}`}>{question}</Label>
            <span className="text-xs text-muted-foreground">Required</span>
          </div>
          <Textarea
            id={`answer-${index}`}
            value={bodies[index] ?? ""}
            onChange={(event) => {
              const next = [...bodies]
              next[index] = event.target.value
              setBodies(next)
            }}
            rows={4}
            aria-required="true"
          />
        </div>
      ))}
      <div className="flex flex-wrap items-center gap-3">
        <Button
          type="submit"
          className="self-start"
          disabled={busy || open > 0}
        >
          Answer and continue
        </Button>
        <Button
          type="button"
          variant="secondary"
          className="self-start"
          disabled={busy}
          onClick={() => void reload()}
        >
          Read the story again
        </Button>
        <p
          className="text-sm text-muted-foreground tabular-nums"
          aria-live="polite"
        >
          {open === 0
            ? "All answered"
            : `${props.questions.length - open} of ${props.questions.length} answered`}
        </p>
        {props.note !== "" ? (
          <p className="text-sm text-pretty">{props.note}</p>
        ) : null}
      </div>
    </form>
  )
}
