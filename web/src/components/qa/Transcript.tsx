"use client"

import { useEffect, useState } from "react"

import { PillToggle } from "@/components/qa/PillToggle"
import { Button } from "@/components/ui/button"
import { reasonText } from "@/lib/guards"
import { readTranscript, type TranscriptView } from "@/lib/qa"

/** What the agent printed in a stage, as the blocks it printed. The agent's output is plain text, so no tool calls are shown. */
export function Transcript({ id }: { id: string }) {
  const [stage, setStage] = useState("")
  const [from, setFrom] = useState(0)
  const [view, setView] = useState<TranscriptView | null>(null)
  const [problem, setProblem] = useState("")

  useEffect(() => {
    let live = true
    readTranscript(id, stage, from)
      .then((body) => {
        if (live) {
          setView(body)
          setProblem("")
        }
      })
      .catch((err: unknown) => {
        if (live) {
          setProblem(reasonText(err, "the transcript could not be read"))
        }
      })
    return () => {
      live = false
    }
  }, [id, stage, from])

  if (problem !== "") {
    return (
      <p role="alert" className="text-sm text-muted-foreground">
        {problem}
      </p>
    )
  }
  if (view === null) {
    return null
  }
  const end = view.from + view.turns.length
  return (
    <section aria-label="Transcript" className="flex flex-col gap-3">
      <div className="flex flex-wrap items-center gap-2">
        {view.stages.map((item) => (
          <PillToggle
            key={item.stage}
            on={view.stage === item.stage}
            onClick={() => {
              setStage(item.stage)
              setFrom(0)
            }}
          >
            {item.stage === "" ? "Whole run" : item.stage}
          </PillToggle>
        ))}
      </div>
      <p className="text-xs text-muted-foreground tabular-nums">
        {view.total} turns · {view.lines} lines
        {view.redacted > 0 ? ` · ${view.redacted} hidden values` : ""}
      </p>
      {view.note !== "" ? (
        <p className="text-sm text-pretty text-muted-foreground">{view.note}</p>
      ) : null}
      <ol className="flex flex-col overflow-hidden rounded-[14px] border border-border bg-card">
        {view.turns.map((turn) => (
          <li
            key={turn.n}
            className="flex flex-col gap-2 border-t border-border px-5 py-4 first:border-t-0"
          >
            <span className="text-xs text-muted-foreground tabular-nums">
              Turn {turn.n}
            </span>
            <pre className="font-sans text-sm leading-6 whitespace-pre-wrap">
              {turn.text}
            </pre>
          </li>
        ))}
      </ol>
      <div className="flex gap-2">
        {view.from > 0 ? (
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={() => setFrom(Math.max(0, view.from - 10))}
          >
            Earlier turns
          </Button>
        ) : null}
        {end < view.total ? (
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={() => setFrom(end)}
          >
            Later turns
          </Button>
        ) : null}
      </div>
      <p className="text-xs text-muted-foreground">
        Terminal:{" "}
        <span className="font-mono">
          span transcript {id} --stage {view.stage}
        </span>
      </p>
    </section>
  )
}
