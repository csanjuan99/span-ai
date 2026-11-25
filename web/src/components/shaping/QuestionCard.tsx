"use client"

import { useState } from "react"
import { CheckIcon } from "lucide-react"
import { cn } from "cn"

import { Button } from "@/components/ui/button"
import { Textarea } from "@/components/ui/textarea"
import { Eyebrow, Panel, Pill, toneText } from "@/components/shaping/session-ui"
import type { ShapeQuestion } from "@/lib/types"
import {
  hintText,
  labelText,
  smallTitleText,
} from "@/components/shaping/styles"

/** What the person decided for a question on this page, before it is sent. */
export type Staged = { kind: "assume" } | { kind: "answer"; text: string }

function Assumption({ label, text }: { label: string; text: string }) {
  return (
    <div className="rounded-[10px] border border-dashed border-hold bg-hold-soft px-3 py-2.5">
      <Eyebrow className="text-hold">{label}</Eyebrow>
      <div className={smallTitleText}>{text}</div>
    </div>
  )
}

/** One question: what blocks, what the agent would assume, and how the person settles it. */
export function QuestionCard({
  question,
  staged,
  busy,
  onStage,
  onClear,
}: {
  question: ShapeQuestion
  staged: Staged | undefined
  busy: boolean
  onStage: (staged: Staged) => void
  onClear: () => void
}) {
  const [editing, setEditing] = useState(false)
  const [draft, setDraft] = useState("")
  const settled = question.status !== "open"
  const blocks =
    question.blocks === ""
      ? ""
      : `${settled ? "Was blocking" : "Blocked"}: ${question.blocks}`
  const mark = settled
    ? question.status === "answered"
      ? { text: "+", tone: "pass" as const }
      : { text: "!", tone: "hold" as const }
    : { text: "?", tone: "acc" as const }
  const badge =
    question.status === "answered" ? (
      <Pill tone="pass" small>
        <CheckIcon className="size-3" aria-hidden="true" />
        Answered by you
      </Pill>
    ) : question.status === "assumed" ? (
      <Pill tone="hold" small>
        Assumed
      </Pill>
    ) : staged !== undefined ? (
      <Pill tone="pass" small>
        <CheckIcon className="size-3" aria-hidden="true" />
        Ready to send
      </Pill>
    ) : (
      <Pill tone="acc" small>
        Open
      </Pill>
    )

  function save() {
    onStage({ kind: "answer", text: draft.trim() })
    setEditing(false)
  }

  function startEditing(initial: string) {
    setDraft(initial)
    setEditing(true)
  }

  return (
    <Panel
      aria-label={`Question ${question.id}`}
      className={cn(
        "flex flex-col gap-3 p-4 sm:px-5",
        !settled && "border-input"
      )}
    >
      <div className="flex items-start gap-2">
        <span
          className={cn(
            "shrink-0 font-mono font-semibold",
            toneText[mark.tone]
          )}
          aria-hidden="true"
        >
          {mark.text}
        </span>
        <span className="min-w-0 flex-1 text-sm font-semibold text-pretty">
          {question.text}
        </span>
        <span className="shrink-0">{badge}</span>
      </div>
      {blocks !== "" ? (
        <p className="text-xs text-muted-foreground">{blocks}</p>
      ) : null}
      {question.status === "answered" ? (
        <div className="rounded-[10px] bg-muted px-3 py-2.5">
          <Eyebrow>
            {`Your answer · ${question.answerId ?? ""}`.replace(/ · $/, "")}
          </Eyebrow>
          <div className="text-[13px] leading-5">{question.answer}</div>
        </div>
      ) : null}
      {question.status === "assumed" ? (
        <Assumption label="The agent assumed" text={question.answer} />
      ) : null}
      {!settled && question.assume !== "" ? (
        <Assumption
          label="If you do not answer, the agent will assume"
          text={question.assume}
        />
      ) : null}
      {staged?.kind === "answer" && !editing ? (
        <div className="rounded-[10px] bg-muted px-3 py-2.5">
          <Eyebrow>Your answer</Eyebrow>
          <div className="text-[13px] leading-5">{staged.text}</div>
        </div>
      ) : null}
      {staged?.kind === "assume" ? (
        <p className={hintText}>The assumption will be used.</p>
      ) : null}
      {editing ? (
        <div className="flex flex-col gap-1.5">
          <label htmlFor={`answer-${question.id}`} className={labelText}>
            Your answer
          </label>
          <Textarea
            id={`answer-${question.id}`}
            aria-label={`Your answer to ${question.id}`}
            rows={2}
            value={draft}
            onChange={(event) => setDraft(event.target.value)}
          />
        </div>
      ) : null}
      <div className="flex flex-wrap gap-2">
        {editing ? (
          <>
            <Button
              type="button"
              className="h-10"
              disabled={busy || draft.trim() === ""}
              aria-label={`Save answer for ${question.id}`}
              onClick={save}
            >
              Save answer
            </Button>
            <Button
              type="button"
              variant="ghost"
              className="h-10"
              onClick={() => setEditing(false)}
            >
              Cancel
            </Button>
          </>
        ) : staged !== undefined ? (
          <Button
            type="button"
            variant="ghost"
            className="h-10"
            aria-label={`Change ${question.id}`}
            onClick={onClear}
          >
            Change
          </Button>
        ) : (
          <>
            {question.status === "open" && question.assume !== "" ? (
              <Button
                type="button"
                variant="outline"
                className="h-10"
                disabled={busy}
                aria-label={`Use the assumption for ${question.id}`}
                onClick={() => onStage({ kind: "assume" })}
              >
                Use the assumption
              </Button>
            ) : null}
            {question.status !== "answered" ? (
              <Button
                type="button"
                variant={question.status === "open" ? "outline" : "ghost"}
                className="h-10"
                disabled={busy}
                aria-label={
                  question.status === "open"
                    ? `Answer ${question.id}`
                    : `Answer ${question.id} instead`
                }
                onClick={() => startEditing("")}
              >
                {question.status === "open" ? "Answer" : "Answer instead"}
              </Button>
            ) : null}
          </>
        )}
      </div>
    </Panel>
  )
}
