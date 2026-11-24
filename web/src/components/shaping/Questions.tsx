"use client"

import { useState } from "react"

import { Button } from "@/components/ui/button"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import type { ShapeQuestion } from "@/lib/types"
import { cn } from "@/lib/utils"
import { RefId } from "@/components/shaping/RefId"
import { PromptHeading } from "@/components/shell/PromptHeading"

/** The open questions. One submit sends every typed answer and every accepted assumption. */
export function Questions({
  questions,
  busy,
  onSubmit,
}: {
  questions: ShapeQuestion[]
  busy: boolean
  onSubmit: (
    answers: { question: string; text: string }[],
    assume: string[]
  ) => void
}) {
  const [typed, setTyped] = useState<Record<string, string>>({})
  const [accepted, setAccepted] = useState<string[]>([])
  const answers = questions.flatMap((question) => {
    const value = (typed[question.id] ?? "").trim()
    return value === "" ? [] : [{ question: question.id, text: value }]
  })
  const assume = accepted.filter(
    (id) => !answers.some((answer) => answer.question === id)
  )
  const ready = answers.length + assume.length > 0

  function toggle(id: string) {
    setAccepted((current) =>
      current.includes(id)
        ? current.filter((item) => item !== id)
        : [...current, id]
    )
  }

  return (
    <form
      className="flex flex-col gap-4"
      onSubmit={(event) => {
        event.preventDefault()
        if (ready && !busy) {
          onSubmit(answers, assume)
        }
      }}
    >
      <PromptHeading>open questions</PromptHeading>
      {questions.map((question) => {
        const isAssumed = assume.includes(question.id)
        return (
          <div
            key={question.id}
            className="flex flex-col gap-2 border border-border p-3"
          >
            <p className="text-pretty">
              <RefId id={question.id} />
              {question.text}
            </p>
            <p className="text-xs text-muted-foreground">{`Blocks ${question.blocks}`}</p>
            {question.assume !== "" ? (
              <p className="text-sm text-pretty text-hold">{`Assumption: ${question.assume}`}</p>
            ) : null}
            <Label
              htmlFor={`answer-${question.id}`}
              className="sr-only"
            >{`Answer ${question.id}`}</Label>
            <Textarea
              id={`answer-${question.id}`}
              rows={2}
              disabled={busy}
              value={typed[question.id] ?? ""}
              onChange={(event) =>
                setTyped((current) => ({
                  ...current,
                  [question.id]: event.target.value,
                }))
              }
            />
            {question.assume !== "" ? (
              <Button
                type="button"
                variant="secondary"
                size="sm"
                className={cn(
                  "self-start",
                  isAssumed && "border-primary text-primary"
                )}
                aria-pressed={isAssumed}
                aria-label={`Accept assumption for ${question.id}`}
                disabled={busy}
                onClick={() => toggle(question.id)}
              >
                Accept assumption
              </Button>
            ) : null}
          </div>
        )
      })}
      <Button type="submit" className="self-start" disabled={!ready || busy}>
        {busy ? "Thinking…" : "Send answers"}
      </Button>
      {busy ? (
        <p className="text-sm text-pretty text-muted-foreground">
          The agent is reading your answers. This can take a minute.
        </p>
      ) : null}
    </form>
  )
}
