"use client"

import { GitMerge, MessageSquare } from "lucide-react"
import { useState, type ReactNode } from "react"

import { abortConflict, chooseConflict, hintConflict } from "@/lib/api"
import {
  canApply,
  choicePairs,
  conflictFileCount,
  conflictSignature,
  remaining,
  type ConflictChoices,
} from "@/lib/conflict"
import type { Conflict } from "@/lib/types"
import { AbortMerge } from "@/components/conflict/AbortMerge"
import { ConflictGroupSection } from "@/components/conflict/ConflictGroupSection"
import { Button } from "@/components/ui/button"
import { quietButton, wideButton } from "@/lib/button-class"
import { Consequences } from "@/components/status-block/Consequences"
import { whereKeptRows } from "@/components/status-block/whereKept"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import { cn } from "@/lib/utils"

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
  return Object.fromEntries(
    conflict.groups
      .filter((group) => group.chosen !== "")
      .map((group) => [group.id, group.chosen])
  )
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
 * until the socket reports the next state, so a second click cannot send it twice. `where`
 * and `kept` fill the box on the right when the page knows them.
 */
export function ConflictPanel({
  id,
  conflict,
  where,
  kept,
}: {
  id: string
  conflict: Conflict
  where?: string
  kept?: string
}) {
  const [choices, setChoices] = useState<ConflictChoices>(() =>
    recorded(conflict)
  )
  const [pending, setPending] = useState<{
    kind: PendingKind
    signature: string
  } | null>(null)
  const [error, setError] = useState("")
  const signature = conflictSignature(conflict)
  const waiting =
    pending !== null && pending.signature === signature ? pending.kind : null
  const view = viewOf(conflict, waiting)
  const files = conflictFileCount(conflict)

  async function send(
    kind: PendingKind,
    post: () => Promise<unknown>,
    fallback: string
  ) {
    setError("")
    setPending({ kind, signature })
    try {
      await post()
    } catch (err: unknown) {
      setPending(null)
      setError(err instanceof Error ? err.message : fallback)
    }
  }

  const abort = () =>
    void send("abort", () => abortConflict(id), "the merge was not aborted")

  const heading = (
    <div className="flex flex-col gap-2">
      <h2 id="conflict-title" className="text-lg font-semibold text-balance">
        Merging {conflict.base} into this branch hits a conflict in {files}{" "}
        {files === 1 ? "file" : "files"}
      </h2>
      <p className="text-sm text-pretty text-muted-foreground">
        <span className="font-mono text-foreground">{conflict.base}</span> into
        this branch: {files} {files === 1 ? "file" : "files"},{" "}
        {conflict.groups.length}{" "}
        {conflict.groups.length === 1 ? "group" : "groups"}.
        {view === "choosing"
          ? " Choose how each group resolves. The result is shown under each choice."
          : ""}
      </p>
    </div>
  )

  return (
    <section aria-labelledby="conflict-title" className="flex flex-col gap-4">
      {view === "choosing" ? null : heading}
      {conflict.note !== "" ? (
        <p className="text-sm text-pretty">{conflict.note}</p>
      ) : null}
      {view === "choosing" ? (
        <Choosing
          conflict={conflict}
          choices={choices}
          where={where}
          kept={kept}
          heading={heading}
          onChoose={(group, option) =>
            setChoices({ ...choices, [group]: option })
          }
          onApply={() =>
            void send(
              "apply",
              () => chooseConflict(id, choicePairs(conflict, choices)),
              "the choice was not applied"
            )
          }
          onHint={(text) =>
            void send(
              "hint",
              () => hintConflict(id, text),
              "the hint was not sent"
            )
          }
          onAbort={abort}
        />
      ) : null}
      {view === "failed" ? <AbortMerge onConfirm={abort} /> : null}
      {view !== "choosing" && view !== "failed" ? (
        <p
          className="text-sm text-pretty text-muted-foreground"
          aria-live="polite"
        >
          {calmLines[view]}
        </p>
      ) : null}
      {error !== "" ? (
        <p role="alert" className="text-sm text-pretty text-destructive">
          {error}
        </p>
      ) : null}
    </section>
  )
}

/** The words for what applying does, from the option each group chose. */
function appliedLine(conflict: Conflict, choices: ConflictChoices): string {
  const parts = conflict.groups.flatMap((group, at) => {
    const option = group.options.find((item) => item.id === choices[group.id])
    return option === undefined
      ? []
      : [`group ${at + 1}: ${option.title.toLowerCase()}`]
  })
  return parts.length === 0
    ? "Nothing is chosen yet."
    : `A merge commit on the branch resolves ${conflictFileCount(conflict)} files. ${parts.join("; ")}.`
}

function Choosing({
  conflict,
  choices,
  where,
  kept,
  heading,
  onChoose,
  onApply,
  onHint,
  onAbort,
}: {
  conflict: Conflict
  choices: ConflictChoices
  where?: string
  kept?: string
  /** The title and intro of the panel: they sit above the groups, left of the box that says what applying does. */
  heading: ReactNode
  onChoose: (group: string, option: string) => void
  onApply: () => void
  onHint: (text: string) => void
  onAbort: () => void
}) {
  const left = remaining(conflict, choices)
  const made = conflict.groups.length - left
  return (
    <div className="flex flex-col gap-6 lg:grid lg:grid-cols-[minmax(0,1fr)_300px]">
      <div className="flex min-w-0 flex-col gap-4">
        {heading}
        {conflict.groups.map((group, at) => (
          <ConflictGroupSection
            key={group.id}
            group={group}
            index={at}
            total={conflict.groups.length}
            chosen={choices[group.id] ?? ""}
            onChoose={(option) => onChoose(group.id, option)}
          />
        ))}
      </div>
      <div className="flex min-w-0 flex-col gap-4">
        <Consequences
          title={`If you apply ${made} ${made === 1 ? "choice" : "choices"}`}
          icon={GitMerge}
          rows={[
            { label: "Changes", body: appliedLine(conflict, choices) },
            ...whereKeptRows(where, kept),
            {
              label: "Next",
              body: `The run goes on from the merge. ${conflict.base} is not touched.`,
            },
          ]}
        />
        <Button
          type="button"
          className={wideButton}
          disabled={!canApply(conflict, choices)}
          onClick={onApply}
        >
          <GitMerge aria-hidden="true" />
          Apply resolution
        </Button>
        <p
          className="text-sm text-muted-foreground tabular-nums"
          aria-live="polite"
        >
          {leftLine(left)}
        </p>
        <div className="flex flex-col gap-3 pt-1">
          <p className="text-xs font-medium text-muted-foreground">
            Other ways out
          </p>
          <Hint onHint={onHint} />
          <AbortMerge base={conflict.base} onConfirm={onAbort} />
        </div>
      </div>
    </div>
  )
}

function Hint({ onHint }: { onHint: (text: string) => void }) {
  const [open, setOpen] = useState(false)
  const [text, setText] = useState("")
  return (
    <div className="flex flex-col gap-1.5">
      <Button
        type="button"
        variant="outline"
        className={cn(wideButton, quietButton)}
        aria-expanded={open}
        onClick={() => setOpen(!open)}
      >
        <MessageSquare aria-hidden="true" />
        Leave a hint for the agent
      </Button>
      <p className="text-xs text-muted-foreground">
        The agent proposes a new resolution. Nothing is applied until you
        choose.
      </p>
      {open ? (
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
      ) : null}
    </div>
  )
}
