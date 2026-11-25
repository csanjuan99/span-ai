"use client"

import { useState } from "react"
import { ArrowRightIcon } from "lucide-react"

import { Button } from "@/components/ui/button"
import { ActionPanel, Pill, twoColumns } from "@/components/shaping/session-ui"
import { QuestionCard, type Staged } from "@/components/shaping/QuestionCard"
import { SessionSummary } from "@/components/shaping/SessionSummary"
import type { ShapeSession } from "@/lib/types"
import { countText, headingText } from "@/components/shaping/styles"

function Group({
  tone,
  label,
  count,
}: {
  tone: "acc" | "pass"
  label: string
  count: number
}) {
  return (
    <div className="flex items-center gap-2 px-1 py-2">
      <Pill tone={tone}>{label}</Pill>
      <span className={countText}>{count}</span>
    </div>
  )
}

/**
 * The questions. Each one is settled here, by using its assumption or by typing an answer,
 * and one button sends them all so the agent runs a single turn. An assumed question can be
 * answered instead until the story is sealed.
 */
export function QuestionsBoard({
  session,
  busy,
  onSubmit,
}: {
  session: ShapeSession
  busy: boolean
  onSubmit: (
    answers: { question: string; text: string }[],
    assume: string[]
  ) => void
}) {
  const [staged, setStaged] = useState<Record<string, Staged>>({})
  const open = session.questions.filter(
    (question) => question.status === "open"
  )
  const settled = session.questions.filter(
    (question) => question.status !== "open"
  )
  const decided = Object.entries(staged)
  const missing = open.filter(
    (question) => staged[question.id] === undefined
  ).length
  const ready = decided.length > 0 && missing === 0

  function stage(id: string, next: Staged) {
    setStaged((current) => ({ ...current, [id]: next }))
  }

  function clear(id: string) {
    setStaged((current) =>
      Object.fromEntries(Object.entries(current).filter(([key]) => key !== id))
    )
  }

  function send() {
    const answers = decided.flatMap(([question, item]) =>
      item.kind === "answer" ? [{ question, text: item.text }] : []
    )
    const assume = decided.flatMap(([question, item]) =>
      item.kind === "assume" ? [question] : []
    )
    onSubmit(answers, assume)
  }

  const card = (question: ShapeSession["questions"][number]) => (
    <QuestionCard
      key={question.id}
      question={question}
      staged={staged[question.id]}
      busy={busy}
      onStage={(next) => stage(question.id, next)}
      onClear={() => clear(question.id)}
    />
  )

  return (
    <div className={twoColumns}>
      <div className="flex min-w-0 flex-col gap-4">
        <div className="flex flex-col gap-1">
          <h2 className={headingText}>
            {open.length > 0
              ? `${open.length} ${open.length === 1 ? "question blocks" : "questions block"} the story`
              : "Answer an assumption instead"}
          </h2>
          <p className="text-sm text-muted-foreground">
            Only what blocks is asked first. Each shows what the agent will
            assume if you skip it.
          </p>
        </div>
        {open.length > 0 ? (
          <>
            <Group tone="acc" label="Blocking now" count={open.length} />
            <div className="flex flex-col gap-4">{open.map(card)}</div>
          </>
        ) : null}
        {settled.length > 0 ? (
          <>
            <Group tone="pass" label="Settled" count={settled.length} />
            <div className="flex flex-col gap-4">{settled.map(card)}</div>
          </>
        ) : null}
      </div>
      <aside className="flex min-w-0 flex-col gap-4">
        <ActionPanel
          title="When you continue"
          icon={ArrowRightIcon}
          rows={[
            {
              label: "Needs",
              body:
                open.length > 0
                  ? `Settle the ${open.length} blocking ${open.length === 1 ? "question" : "questions"}. Use the assumption or answer.`
                  : "Pick the assumptions you want to answer instead.",
            },
            {
              label: "Changes",
              body: "Answers are saved with ids A1, A2… Assumptions are saved as assumed.",
            },
            {
              label: "Next",
              body: "The agent builds the example map from your answers.",
            },
          ]}
        >
          <Button
            type="button"
            className="h-10 w-full"
            disabled={!ready || busy}
            onClick={send}
          >
            {busy ? "Thinking…" : "Continue to example map"}
          </Button>
          <p className="text-xs text-muted-foreground">
            {ready
              ? "The agent reads your answers. This can take a minute."
              : missing > 0
                ? `Settle ${missing} more ${missing === 1 ? "question" : "questions"} to continue.`
                : "Choose something to send."}
          </p>
        </ActionPanel>
        <SessionSummary session={session} />
      </aside>
    </div>
  )
}
