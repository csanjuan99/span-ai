"use client"

import { useState } from "react"
import { CheckCircle2Icon, PlayIcon } from "lucide-react"

import { Button } from "@/components/ui/button"
import { CopyChip } from "@/components/ledger/CopyChip"
import {
  ActionPanel,
  Panel,
  Pill,
  twoColumns,
} from "@/components/shaping/session-ui"
import { MapRules } from "@/components/shaping/MapRules"
import { SessionSummary } from "@/components/shaping/SessionSummary"
import { StoryView } from "@/components/shaping/StoryView"
import { whenLabel } from "@/lib/clock"
import { TabStrip } from "@/components/shaping/TabStrip"
import type { ShapeSession } from "@/lib/types"
import { monoCaption, noteText } from "@/components/shaping/styles"
import { touchWideButton } from "@/lib/button-class"
import { labelled } from "@/lib/shape-session"

type Tab = "story" | "map" | "questions" | "context"

const tabs: { value: Tab; label: string }[] = [
  { value: "story", label: "Story" },
  { value: "map", label: "Example map" },
  { value: "questions", label: "Questions" },
  { value: "context", label: "Context" },
]

function Questions({ session }: { session: ShapeSession }) {
  if (session.questions.length === 0) {
    return (
      <p className="text-sm text-muted-foreground">The agent asked nothing.</p>
    )
  }
  return (
    <ul className="flex flex-col gap-3">
      {session.questions.map((question) => (
        <li
          key={question.id}
          className="flex flex-col gap-1 rounded-[10px] border border-border p-3"
        >
          <div className="flex flex-wrap items-center gap-2 text-sm font-semibold">
            <span className={monoCaption}>{question.id}</span>
            {question.text}
            <Pill tone={question.status === "assumed" ? "hold" : "pass"} small>
              {question.status === "assumed"
                ? "Assumed"
                : labelled("Answered", question.answerId)}
            </Pill>
          </div>
          <p className={noteText}>{question.answer}</p>
        </li>
      ))}
    </ul>
  )
}

function Notes({ session }: { session: ShapeSession }) {
  if (session.proposals.length === 0) {
    return (
      <p className="text-sm text-muted-foreground">
        The agent proposed no context notes.
      </p>
    )
  }
  return (
    <ul className="flex flex-col gap-3">
      {session.proposals.map((note) => (
        <li
          key={note.id}
          className="flex flex-col gap-1 rounded-[10px] border border-border p-3"
        >
          <div className="flex items-center gap-2">
            <span className="font-mono text-xs">{note.section}</span>
            <Pill tone={note.status === "accepted" ? "pass" : "mut"} small>
              {note.status === "accepted" ? "Accepted" : "Not added"}
            </Pill>
          </div>
          <p className="text-[13px] leading-5">{note.text}</p>
        </li>
      ))}
    </ul>
  )
}

/**
 * A sealed story: read-only, with its digest and what can be done with it. The same page
 * shows right after sealing and when the session is opened again later.
 */
export function SealedBoard({
  session,
  busy,
  onRun,
  onPreview,
  onFork,
}: {
  session: ShapeSession
  busy: boolean
  onRun: () => void
  onPreview: () => void
  onFork: () => void
}) {
  const [tab, setTab] = useState<Tab>("story")
  const assumed = session.questions.filter(
    (question) => question.status === "assumed"
  )
  const digest = `sha256:${session.digest}`
  const when = whenLabel(session.sealedAt)
  return (
    <div className={twoColumns}>
      <div className="flex min-w-0 flex-col gap-4">
        <Panel className="flex items-center gap-3 border-pass bg-pass-soft px-4 py-2.5 text-pass sm:px-5">
          <CheckCircle2Icon className="size-4 shrink-0" aria-hidden="true" />
          <span className="text-[13px] leading-5">
            <b>
              {when !== "" ? `Sealed ${when}. ` : "Sealed. "}This story cannot
              change.
            </b>{" "}
            Nothing on this page is editable. To change it, start a new session
            from it.
          </span>
        </Panel>
        <Panel aria-label="Sealed session" className="flex flex-col gap-3 p-4">
          <TabStrip
            label="Sealed session"
            value={tab}
            options={tabs}
            onChange={setTab}
          />
          {tab === "story" ? <StoryView story={session.story} /> : null}
          {tab === "map" ? (
            <div className="flex flex-col gap-4">
              <MapRules session={session} />
            </div>
          ) : null}
          {tab === "questions" ? <Questions session={session} /> : null}
          {tab === "context" ? <Notes session={session} /> : null}
        </Panel>
      </div>
      <aside className="flex min-w-0 flex-col gap-4">
        <Panel aria-label="Digest" className="flex flex-col gap-2 p-4">
          <div className="text-sm font-semibold">Digest</div>
          <div className="rounded-[10px] bg-muted px-3 py-2.5 font-mono text-xs leading-[18px] break-all">
            {`${digest.slice(0, 27)}…`}
          </div>
          <CopyChip label="digest" value={digest}>
            Copy digest
          </CopyChip>
          <p className="text-xs text-muted-foreground">
            {assumed.length > 0
              ? `${assumed.length} ${assumed.length === 1 ? "assumption was" : "assumptions were"} sealed in: ${assumed.map((question) => question.id).join(" and ")}. `
              : ""}
            Any change to the markdown would give a different digest.
          </p>
        </Panel>
        <ActionPanel
          title="What you can do"
          icon={PlayIcon}
          rows={[
            {
              label: "Run",
              body: "Starts a run from this story on a new branch. Nothing in your repo changes until implement.",
            },
            {
              label: "Jira",
              body:
                session.jira !== null
                  ? `${session.jira.key} is linked. Open it, or nothing to create.`
                  : "Shows exactly what would be created. Nothing is sent until you confirm.",
            },
            {
              label: "New version",
              body: "Starts a new session from this story. The sealed one stays.",
            },
          ]}
        >
          <Button
            type="button"
            className={touchWideButton}
            disabled={busy}
            onClick={onRun}
          >
            <PlayIcon aria-hidden="true" />
            Create run
          </Button>
          {session.jira !== null ? (
            <a
              className="inline-flex h-11 w-full items-center justify-center rounded-[10px] border border-input bg-card text-sm font-medium md:h-10"
              href={session.jira.url}
            >
              {`Open ${session.jira.key}`}
            </a>
          ) : (
            <Button
              type="button"
              variant="outline"
              className={touchWideButton}
              disabled={busy}
              onClick={onPreview}
            >
              Preview in Jira
            </Button>
          )}
          <Button
            type="button"
            variant="ghost"
            className={touchWideButton}
            disabled={busy}
            onClick={onFork}
          >
            New session from this story
          </Button>
        </ActionPanel>
        <SessionSummary session={session} />
      </aside>
    </div>
  )
}
