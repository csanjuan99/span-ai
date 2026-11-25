"use client"

import { useState } from "react"
import { ArrowRightIcon } from "lucide-react"

import { Button } from "@/components/ui/button"
import { ActionPanel, Panel, twoColumns } from "@/components/shaping/session-ui"
import { MapRules } from "@/components/shaping/MapRules"
import { SessionSummary } from "@/components/shaping/SessionSummary"
import { SourceDetail } from "@/components/shaping/SourceDetail"
import { sourceIcons } from "@/components/shaping/source-icons"
import type { ShapeSession, ShapeSource } from "@/lib/types"
import { headingText } from "@/components/shaping/styles"

const legend = [
  { kind: "prompt", text: "your idea." },
  { kind: "answer", text: "a question and the answer you gave (A1, A2…)." },
  { kind: "code", text: "a file and line, as the agent read it." },
  { kind: "context", text: "a note from the product context." },
] as const

/** The example map: every rule and example with its source, then the way on. */
export function MapBoard({
  session,
  next,
  onContinue,
}: {
  session: ShapeSession
  /** What the next step is called, for the button and the box. */
  next: "split" | "context" | "seal"
  onContinue: () => void
}) {
  const [pick, setPick] = useState<{ id: string; source: ShapeSource } | null>(
    null
  )
  const assumed = session.examples.filter(
    (example) =>
      example.source.kind === "answer" &&
      session.questions.some(
        (q) => q.id === example.source.ref && q.status === "assumed"
      )
  ).length
  const nextText = {
    split:
      "The agent proposed splitting the story. You decide on the next step.",
    context: "You settle the product-context notes the agent proposed.",
    seal: "The story is ready to seal.",
  }[next]
  return (
    <div className={twoColumns}>
      <div className="flex min-w-0 flex-col gap-4">
        <div>
          <h2 className={headingText}>Example map</h2>
          <p className="text-sm text-muted-foreground">
            {`${session.rules.length} ${session.rules.length === 1 ? "rule" : "rules"}, ${session.examples.length} ${session.examples.length === 1 ? "example" : "examples"}. Every item shows its source.`}
            {assumed > 0
              ? ` ${assumed} ${assumed === 1 ? "example rests" : "examples rest"} on an assumption.`
              : ""}
          </p>
        </div>
        {session.rules.length === 0 ? (
          <p className="text-sm text-muted-foreground">No rules yet.</p>
        ) : null}
        <MapRules
          session={session}
          picked={pick?.id}
          onPick={(id, source) => setPick({ id, source })}
        />
      </div>
      <aside className="flex min-w-0 flex-col gap-4">
        <ActionPanel
          title="When you continue"
          icon={ArrowRightIcon}
          rows={[
            {
              label: "Needs",
              body: "Check that each rule and example reads right.",
            },
            {
              label: "Changes",
              body: "Nothing yet. The map is a draft until you seal.",
            },
            { label: "Next", body: nextText },
          ]}
        >
          <Button type="button" className="h-10 w-full" onClick={onContinue}>
            Looks right: continue
          </Button>
        </ActionPanel>
        {pick !== null ? (
          <SourceDetail session={session} id={pick.id} source={pick.source} />
        ) : null}
        <Panel className="p-4">
          <div className="mb-2 text-sm font-semibold">
            Where each item comes from
          </div>
          {legend.map(({ kind, text }) => {
            const Icon = sourceIcons[kind]
            return (
              <div
                key={kind}
                className="grid grid-cols-[20px_minmax(0,1fr)] gap-2 py-1"
              >
                <Icon className="mt-0.5 size-4" aria-hidden="true" />
                <span className="text-[13px] leading-5">
                  <b>{kind}</b> · {text}
                </span>
              </div>
            )
          })}
          <p className="mt-2 text-xs text-muted-foreground">
            Click any source to see the exact reference.
          </p>
        </Panel>
        <SessionSummary session={session} />
      </aside>
    </div>
  )
}
