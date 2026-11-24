"use client"

import { useState, type FormEvent } from "react"

import { Button } from "@/components/ui/button"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"

/** The questions a waiting run needs before frame starts again. */
export function Waiting(props: {
  questions: string[]
  note: string
  onAnswer: (answers: { question: string; body: string }[]) => Promise<void>
  onReload: () => Promise<void>
}) {
  const [bodies, setBodies] = useState<string[]>(() => props.questions.map(() => ""))
  const [busy, setBusy] = useState(false)

  async function answer(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setBusy(true)
    try {
      await props.onAnswer(props.questions.map((question, index) => ({ question, body: bodies[index] ?? "" })))
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
    <form className="flex flex-col gap-3" onSubmit={(event) => void answer(event)}>
      <p className="text-pretty text-sm text-muted-foreground">
        Frame reads the story in the language it is written and carries what it already decides into this run. Answer here
        when the story does not say it.
      </p>
      {props.questions.map((question, index) => (
        <div className="flex flex-col gap-2" key={`${index}-${question}`}>
          <Label htmlFor={`answer-${index}`}>{question}</Label>
          <Textarea
            id={`answer-${index}`}
            value={bodies[index] ?? ""}
            onChange={(event) => {
              const next = [...bodies]
              next[index] = event.target.value
              setBodies(next)
            }}
            rows={4}
          />
        </div>
      ))}
      <div className="flex flex-wrap items-center gap-3">
        <Button type="submit" className="self-start" disabled={busy}>
          Answer and continue
        </Button>
        <Button type="button" variant="secondary" className="self-start" disabled={busy} onClick={() => void reload()}>
          Read the story again
        </Button>
        {props.note !== "" ? <p className="text-pretty text-sm">{props.note}</p> : null}
      </div>
    </form>
  )
}
