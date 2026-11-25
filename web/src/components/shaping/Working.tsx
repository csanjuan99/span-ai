"use client"

import { LoaderIcon } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Consequences } from "@/components/status-block/Consequences"
import {
  Panel,
  Pill,
  twoColumns,
  Eyebrow,
} from "@/components/shaping/session-ui"
import { SessionAgent } from "@/components/shaping/SessionAgent"
import { SessionSummary } from "@/components/shaping/SessionSummary"
import { questionCounts } from "@/lib/shape-session"
import type { ShapeSession } from "@/lib/types"
import { headingText, labelText } from "@/components/shaping/styles"

/**
 * The interview while the agent works, or before it has drafted anything. The turn runs on the
 * server inside one request, so there is no live progress to show: the counts are what was
 * saved before it started, and the page says so.
 */
export function Working({
  session,
  running,
  busy,
  onSession,
  onDraft,
}: {
  session: ShapeSession
  running: boolean
  busy: boolean
  onSession: (next: ShapeSession) => void
  onDraft: () => void
}) {
  const questions = questionCounts(session)
  const found: [string, number, string][] = [
    ["Rules", session.rules.length, "saved so far"],
    ["Examples", session.examples.length, "saved so far"],
    ["Questions", questions.open, "waiting for you"],
  ]
  return (
    <div className={twoColumns}>
      <div className="flex min-w-0 flex-col gap-4">
        <Panel aria-label="Needs you" className="overflow-hidden">
          <div className="flex items-center gap-2 bg-accent px-4 py-2.5 text-primary sm:px-5">
            <LoaderIcon
              className={
                running
                  ? "size-4 animate-spin motion-reduce:animate-none"
                  : "size-4"
              }
              aria-hidden="true"
            />
            <span className={labelText}>Interviewing</span>
          </div>
          <div className="flex flex-col gap-3 p-4 sm:p-6">
            <h2 className={headingText}>
              {running
                ? "The agent is reading and drafting"
                : "The agent has not drafted a story yet"}
            </h2>
            <p className="text-sm leading-[22px] text-pretty text-muted-foreground">
              {running
                ? "Nothing to do yet. The turn runs on the server and this page shows the result when it ends. If you leave, the session is saved and shows on your list."
                : "No question is open and there is no story. Run the next turn and the agent drafts the rules, the examples and its questions."}
            </p>
            {running ? (
              <Pill tone="acc" small>
                working
              </Pill>
            ) : (
              <Button
                type="button"
                className="h-10 self-start"
                disabled={busy}
                onClick={onDraft}
              >
                Draft the story
              </Button>
            )}
          </div>
        </Panel>
        <Panel aria-label="Found so far" className="grid sm:grid-cols-3">
          {found.map(([label, value, hint], index) => (
            <div
              key={label}
              className={
                index > 0
                  ? "flex flex-col gap-1 border-t border-border p-4 sm:border-t-0 sm:border-l"
                  : "flex flex-col gap-1 p-4"
              }
            >
              <Eyebrow>{label}</Eyebrow>
              <div className="text-lg font-semibold tabular-nums">{value}</div>
              <div className="text-xs text-muted-foreground">{hint}</div>
            </div>
          ))}
        </Panel>
      </div>
      <aside className="flex min-w-0 flex-col gap-4">
        <Panel className="p-4">
          <Consequences
            title="What happens next"
            icon={LoaderIcon}
            rows={[
              {
                label: "Next",
                body: "If something blocks, the session asks you. If nothing does, it goes straight to the example map.",
              },
              {
                label: "Kept",
                body: "Rules and examples are saved when the turn ends.",
              },
              {
                label: "Changes",
                body: "Nothing in your repo or Jira. The interview is read-only and Span checks it.",
              },
            ]}
          />
        </Panel>
        <SessionSummary session={session} />
        <SessionAgent session={session} onSession={onSession} />
      </aside>
    </div>
  )
}
