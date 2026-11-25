"use client"

import { useEffect, useState } from "react"
import { ArrowRightIcon, CheckIcon } from "lucide-react"
import { cn } from "cn"

import { previewShapeContext } from "@/lib/api"
import { reasonText } from "@/lib/guards"
import { Button } from "@/components/ui/button"
import {
  ActionPanel,
  CodeBlock,
  Panel,
  Pill,
  twoColumns,
} from "@/components/shaping/session-ui"
import { SessionSummary } from "@/components/shaping/SessionSummary"
import type { ShapeContextDiff, ShapeProposal, ShapeSession } from "@/lib/types"
import {
  headingText,
  lineNumberCell,
  wrapRow,
} from "@/components/shaping/styles"

type Decision = "accept" | "skip"

/** What the person decided for each note on this page. Nothing is written until they continue. */
type Decisions = Record<string, Decision>

function Note({
  proposal,
  decision,
  busy,
  onDecide,
  onUndo,
}: {
  proposal: ShapeProposal
  decision: Decision | undefined
  busy: boolean
  onDecide: (decision: Decision) => void
  onUndo: () => void
}) {
  const written = proposal.status === "accepted"
  const shown = written ? "accept" : decision
  return (
    <Panel
      aria-label={`Note ${proposal.id}`}
      className={cn(
        "flex flex-col gap-3 p-4 sm:px-5",
        shown === undefined && "border-input"
      )}
    >
      <div className={wrapRow}>
        <span className="font-mono text-xs">{proposal.section}</span>
        <span className="ml-auto">
          {shown === "accept" ? (
            <Pill tone="pass" small>
              <CheckIcon className="size-3" aria-hidden="true" />
              Accepted
            </Pill>
          ) : shown === "skip" ? (
            <Pill tone="mut" small>
              Skipped
            </Pill>
          ) : (
            <Pill tone="hold" small>
              Proposed
            </Pill>
          )}
        </span>
      </div>
      <p className="text-sm leading-[22px] text-pretty">{proposal.text}</p>
      <p className="text-xs text-muted-foreground">
        {written
          ? "Already in the product context."
          : "The agent proposed it from this session. Accepting adds it to the product context."}
      </p>
      {written ? null : (
        <div className="flex flex-wrap gap-2">
          {shown === undefined ? (
            <>
              <Button
                type="button"
                variant="outline"
                className="h-10"
                disabled={busy}
                aria-label={`Accept ${proposal.id}`}
                onClick={() => onDecide("accept")}
              >
                <CheckIcon aria-hidden="true" />
                Accept
              </Button>
              <Button
                type="button"
                variant="ghost"
                className="h-10"
                disabled={busy}
                aria-label={`Skip ${proposal.id}`}
                onClick={() => onDecide("skip")}
              >
                Skip
              </Button>
            </>
          ) : (
            <Button
              type="button"
              variant="ghost"
              className="h-10"
              disabled={busy}
              aria-label={`Undo ${proposal.id}`}
              onClick={onUndo}
            >
              Undo
            </Button>
          )}
        </div>
      )}
    </Panel>
  )
}

function Diff({
  diff,
  error,
}: {
  diff: ShapeContextDiff | null
  error: string
}) {
  if (error !== "") {
    return (
      <p role="alert" className="text-sm text-destructive">
        {error}
      </p>
    )
  }
  if (diff === null || diff.lines.length === 0) {
    return (
      <p className="text-sm text-muted-foreground">
        Accept a note to see what would be added.
      </p>
    )
  }
  return (
    <>
      <CodeBlock label="Context change">
        {diff.lines.map((line) => (
          <div
            key={line.n}
            className={cn(
              "grid grid-cols-[36px_16px_minmax(0,1fr)] pr-3.5 whitespace-pre",
              line.kind === "add" && "bg-diff-add"
            )}
          >
            <span className={lineNumberCell}>{line.n}</span>
            <span aria-hidden="true">{line.kind === "add" ? "+" : " "}</span>
            <span>{line.text}</span>
          </div>
        ))}
      </CodeBlock>
      <p className="text-xs text-muted-foreground">{`Product context of this folder: ${diff.path}`}</p>
    </>
  )
}

/**
 * The notes the agent proposes for the product context. Accepting or skipping is staged here;
 * the file is written, and the skipped notes dropped, when the person continues.
 */
export function ContextBoard({
  session,
  busy,
  onContinue,
}: {
  session: ShapeSession
  busy: boolean
  onContinue: (accept: string[], skip: string[]) => void
}) {
  const [decisions, setDecisions] = useState<Decisions>({})
  const notes = session.proposals
  const staged = (decision: Decision) =>
    notes
      .filter(
        (note) => note.status === "proposed" && decisions[note.id] === decision
      )
      .map((note) => note.id)
  const accept = staged("accept")
  const skip = staged("skip")
  const accepted =
    notes.filter((note) => note.status === "accepted").length + accept.length
  const pending = notes.filter(
    (note) => note.status === "proposed" && decisions[note.id] === undefined
  ).length
  const key = accept.join(",")
  const [loaded, setLoaded] = useState<{
    key: string
    diff: ShapeContextDiff | null
    error: string
  } | null>(null)

  useEffect(() => {
    if (key === "") {
      return
    }
    let live = true
    void previewShapeContext(session.id, session.workspace, key.split(","))
      .then((diff) => {
        if (live) {
          setLoaded({ key, diff, error: "" })
        }
      })
      .catch((err: unknown) => {
        if (live) {
          setLoaded({
            key,
            diff: null,
            error: reasonText(err, "the preview could not be read"),
          })
        }
      })
    return () => {
      live = false
    }
  }, [key, session.id, session.workspace])

  const current = key !== "" && loaded?.key === key ? loaded : null
  const set = (id: string, decision: Decision | undefined) =>
    setDecisions((now) => {
      const rest = Object.fromEntries(
        Object.entries(now).filter(([other]) => other !== id)
      )
      return decision === undefined ? rest : { ...rest, [id]: decision }
    })

  return (
    <div className={twoColumns}>
      <div className="flex min-w-0 flex-col gap-4">
        <div>
          <h2 className={headingText}>
            {`${notes.length} ${notes.length === 1 ? "note" : "notes"} for your product context`}
          </h2>
          <p className="text-sm text-muted-foreground">
            Accepted notes help every later session in this repo. Skipped ones
            are dropped.
          </p>
        </div>
        <div className="flex flex-col gap-4">
          {notes.map((note) => (
            <Note
              key={note.id}
              proposal={note}
              decision={decisions[note.id]}
              busy={busy}
              onDecide={(decision) => set(note.id, decision)}
              onUndo={() => set(note.id, undefined)}
            />
          ))}
        </div>
        <Panel className="flex flex-col gap-2 p-4">
          <div className="text-sm font-semibold">What will be added</div>
          <Diff
            diff={current === null ? null : current.diff}
            error={current === null ? "" : current.error}
          />
        </Panel>
      </div>
      <aside className="flex min-w-0 flex-col gap-4">
        <ActionPanel
          title="When you continue"
          icon={ArrowRightIcon}
          rows={[
            {
              label: "Changes",
              body: "Accepted notes are added to the product context. Nothing else.",
            },
            { label: "Kept", body: "Skipped notes are dropped, not saved." },
            { label: "Next", body: "The story is ready to seal." },
          ]}
        >
          <Button
            type="button"
            className="h-10 w-full"
            disabled={busy}
            onClick={() => onContinue(accept, skip)}
          >
            Continue to seal
          </Button>
          <p className="text-xs text-muted-foreground">
            {`${accepted} accepted, ${pending} still proposed.`}
            {pending > 0 ? " They are dropped if you continue now." : ""}
          </p>
        </ActionPanel>
        <SessionSummary session={session} />
      </aside>
    </div>
  )
}
