"use client"

import { useState, type FormEvent } from "react"

import { Check, CircleHelp, Play, RotateCw } from "lucide-react"

import { Consequences } from "@/components/status-block/Consequences"
import { whereKeptRows } from "@/components/status-block/whereKept"
import { Button } from "@/components/ui/button"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import { quietButton, wideButton } from "@/lib/button-class"
import { designField, quietDisabled } from "@/lib/field-class"
import { cn } from "@/lib/utils"

/** The state chip of one question: answered, or still needing its answer. */
function QuestionState({ answered }: { answered: boolean }) {
  return (
    <span
      className={cn(
        "inline-flex h-5 shrink-0 items-center gap-1 rounded-full px-2 text-xs font-medium",
        answered ? "bg-pass-soft text-pass" : "bg-hold-soft text-hold"
      )}
    >
      {answered ? (
        <Check className="size-3" aria-hidden="true" />
      ) : (
        <CircleHelp className="size-3" aria-hidden="true" />
      )}
      {answered ? "Answered" : "Needs an answer"}
    </span>
  )
}

/** The questions a waiting run needs before frame starts again. */
export function Waiting(props: {
  questions: string[]
  note: string
  where?: string
  kept?: string
  onAnswer: (answers: { question: string; body: string }[]) => Promise<unknown>
  onReload: () => Promise<unknown>
}) {
  const [bodies, setBodies] = useState<string[]>(() =>
    props.questions.map(() => "")
  )
  const [busy, setBusy] = useState(false)
  // The engine refuses an empty answer, so every question is required: say so before the request fails.
  const open = bodies.filter((body) => body.trim() === "").length
  const firstOpen = props.questions.findIndex(
    (_, index) => (bodies[index] ?? "").trim() === ""
  )
  const total = props.questions.length

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
      className="flex flex-col gap-6 lg:grid lg:grid-cols-[minmax(0,1fr)_300px]"
      onSubmit={(event) => void answer(event)}
    >
      <div className="flex min-w-0 flex-col gap-4">
        <div className="flex flex-col gap-2">
          <h3 className="text-lg font-semibold text-balance">
            Frame needs {total} {total === 1 ? "answer" : "answers"} before it
            can plan
          </h3>
          <p className="text-sm text-pretty text-muted-foreground">
            Frame reads the story in the language it is written and carries what
            it already decides into this run. Answer here when the story does
            not say it. A short answer is enough.
          </p>
        </div>
        {props.questions.map((question, index) => {
          const answered = (bodies[index] ?? "").trim() !== ""
          return (
            <div
              className="flex flex-col gap-2 rounded-[14px] border border-border p-4"
              key={`${index}-${question}`}
            >
              <div className="flex flex-wrap items-center justify-between gap-x-3 gap-y-1">
                <Label htmlFor={`answer-${index}`} className="text-sm">
                  <span>
                    <span aria-hidden="true">{index + 1}. </span>
                    {question}
                  </span>
                </Label>
                <QuestionState answered={answered} />
              </div>
              <p className="text-xs text-muted-foreground">
                Required. The agent will not guess this one.
              </p>
              <Textarea
                id={`answer-${index}`}
                aria-label={question}
                value={bodies[index] ?? ""}
                onChange={(event) => {
                  const next = [...bodies]
                  next[index] = event.target.value
                  setBodies(next)
                }}
                className={designField}
                rows={3}
                aria-required="true"
              />
            </div>
          )
        })}
      </div>
      <div className="flex min-w-0 flex-col gap-4">
        <Consequences
          title="When you send"
          icon={CircleHelp}
          rows={[
            {
              label: "Changes",
              body: "Your answers are saved to the ledger. No code is written in frame.",
            },
            ...whereKeptRows(props.where, props.kept),
            {
              label: "Next",
              body: "Frame resumes with your answers, then hands over to build.",
            },
          ]}
        />
        <Button
          type="submit"
          className={cn(wideButton, quietDisabled)}
          disabled={busy || open > 0}
        >
          <Play aria-hidden="true" />
          Answer and continue
        </Button>
        <p
          className="text-xs text-muted-foreground tabular-nums"
          aria-live="polite"
        >
          {open === 0
            ? "All answered"
            : `Answer question ${firstOpen + 1} to continue. ${total - open} of ${total} answered.`}
        </p>
        <Button
          type="button"
          variant="outline"
          className={cn(wideButton, quietButton)}
          disabled={busy}
          onClick={() => void reload()}
        >
          <RotateCw aria-hidden="true" />
          Read the story again
        </Button>
        {props.note !== "" ? (
          <p className="text-sm text-pretty">{props.note}</p>
        ) : null}
      </div>
    </form>
  )
}
