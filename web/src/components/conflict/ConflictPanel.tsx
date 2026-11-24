"use client"

import { useState } from "react"

import { abortConflict, chooseConflict, hintConflict } from "@/lib/api"
import { canApply, choicePairs, conflictSignature, remaining, type ConflictChoices } from "@/lib/conflict"
import type { Conflict } from "@/lib/types"
import { AbortMerge } from "@/components/conflict/AbortMerge"
import { ConflictGroupSection } from "@/components/conflict/ConflictGroupSection"
import { Button } from "@/components/ui/button"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"

/** An action the page sent and the server has not answered on the socket yet. */
type PendingKind = "apply" | "hint" | "abort"

/** What the panel body shows: the server state, or the pending action that replaces it. */
type View = "choosing" | "proposing" | "applying" | "aborting" | "failed"

const calmLines: Record<Exclude<View, "choosing" | "failed">, string> = {
  proposing: "The agent is reading both sides…",
  applying: "Applying your choice…",
  aborting: "Aborting the merge…",
}

function viewOf(conflict: Conflict, waiting: PendingKind | null): View {
  if (waiting === "apply") {
    return "applying"
  }
  if (waiting === "hint") {
    return "proposing"
  }
  return waiting === "abort" ? "aborting" : conflict.state
}

/** Choices the server already recorded. Nothing is preselected otherwise. */
function recorded(conflict: Conflict): ConflictChoices {
  return Object.fromEntries(conflict.groups.filter((group) => group.chosen !== "").map((group) => [group.id, group.chosen]))
}

function leftLine(left: number): string {
  if (left === 0) {
    return "Every group has a choice."
  }
  return `${String(left)} ${left === 1 ? "group" : "groups"} left to choose`
}

/**
 * The merge-conflict phase on the span page. Nothing is written to the branch until the
 * person picked an option for every group and applied; a sent action shows as pending
 * until the socket reports the next state, so a second click cannot send it twice.
 */
export function ConflictPanel({ id, conflict }: { id: string; conflict: Conflict }) {
  const [choices, setChoices] = useState<ConflictChoices>(() => recorded(conflict))
  const [pending, setPending] = useState<{ kind: PendingKind; signature: string } | null>(null)
  const [error, setError] = useState("")
  const signature = conflictSignature(conflict)
  const waiting = pending !== null && pending.signature === signature ? pending.kind : null
  const view = viewOf(conflict, waiting)

  async function send(kind: PendingKind, post: () => Promise<unknown>, fallback: string) {
    setError("")
    setPending({ kind, signature })
    try {
      await post()
    } catch (err: unknown) {
      setPending(null)
      setError(err instanceof Error ? err.message : fallback)
    }
  }

  const abort = () => void send("abort", () => abortConflict(id), "the merge was not aborted")

  return (
    <section aria-labelledby="conflict-title" className="flex flex-col gap-3">
      <h2 id="conflict-title" className="text-balance text-sm text-hold">
        Merge conflict
      </h2>
      <p className="text-sm text-muted-foreground">
        Merging <span className="font-mono text-foreground">{conflict.base}</span> into this branch ·{" "}
        <span>{conflict.state}</span>
      </p>
      {conflict.note !== "" ? <p className="text-pretty text-sm">{conflict.note}</p> : null}
      {view === "choosing" ? (
        <Choosing
          conflict={conflict}
          choices={choices}
          onChoose={(group, option) => setChoices({ ...choices, [group]: option })}
          onApply={() => void send("apply", () => chooseConflict(id, choicePairs(conflict, choices)), "the choice was not applied")}
          onHint={(text) => void send("hint", () => hintConflict(id, text), "the hint was not sent")}
          onAbort={abort}
        />
      ) : null}
      {view === "failed" ? <AbortMerge onConfirm={abort} /> : null}
      {view !== "choosing" && view !== "failed" ? (
        <p className="text-pretty text-sm text-muted-foreground" aria-live="polite">
          {calmLines[view]}
        </p>
      ) : null}
      {error !== "" ? (
        <p role="alert" className="text-pretty text-sm text-destructive">
          {error}
        </p>
      ) : null}
    </section>
  )
}

function Choosing({
  conflict,
  choices,
  onChoose,
  onApply,
  onHint,
  onAbort,
}: {
  conflict: Conflict
  choices: ConflictChoices
  onChoose: (group: string, option: string) => void
  onApply: () => void
  onHint: (text: string) => void
  onAbort: () => void
}) {
  const left = remaining(conflict, choices)
  return (
    <>
      {conflict.groups.map((group) => (
        <ConflictGroupSection
          key={group.id}
          group={group}
          chosen={choices[group.id] ?? ""}
          onChoose={(option) => onChoose(group.id, option)}
        />
      ))}
      <div className="flex flex-wrap items-center gap-3 border-t border-border pt-4">
        <Button type="button" disabled={!canApply(conflict, choices)} onClick={onApply}>
          Apply resolution
        </Button>
        <p className="text-sm text-muted-foreground tabular-nums" aria-live="polite">
          {leftLine(left)}
        </p>
      </div>
      <Hint onHint={onHint} />
      <AbortMerge onConfirm={onAbort} />
    </>
  )
}

function Hint({ onHint }: { onHint: (text: string) => void }) {
  const [text, setText] = useState("")
  return (
    <div className="flex flex-col gap-2">
      <Label htmlFor="conflict-hint">Hint for the agent</Label>
      <Textarea
        id="conflict-hint"
        rows={3}
        placeholder="For example: keep the new rounding, the base only renamed the variable."
        value={text}
        onChange={(event) => setText(event.target.value)}
      />
      <Button
        type="button"
        variant="secondary"
        className="self-start"
        disabled={text.trim() === ""}
        onClick={() => onHint(text.trim())}
      >
        Ask the agent again
      </Button>
    </div>
  )
}
