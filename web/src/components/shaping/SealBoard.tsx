"use client"

import { LockIcon } from "lucide-react"
import { cn } from "@/lib/utils"

import { Button } from "@/components/ui/button"
import {
  ActionPanel,
  Panel,
  Pill,
  toneText,
  twoColumns,
} from "@/components/shaping/session-ui"
import { SessionSummary } from "@/components/shaping/SessionSummary"
import { StoryView } from "@/components/shaping/StoryView"
import { contextCounts, sealChecks } from "@/lib/shape-session"
import type { ShapeSession } from "@/lib/types"
import { headingText, wrapRow } from "@/components/shaping/styles"

/** The story before it is sealed: what it says, what sealing locks in, and the one button that does it. */
export function SealBoard({
  session,
  busy,
  canGoBack,
  onSeal,
  onBack,
  onRevisit,
}: {
  session: ShapeSession
  busy: boolean
  canGoBack: boolean
  onSeal: () => void
  onBack: () => void
  onRevisit: () => void
}) {
  const checks = sealChecks(session, contextCounts(session).proposed)
  const hasAssumption = session.questions.some(
    (question) => question.status === "assumed"
  )
  return (
    <div className={twoColumns}>
      <Panel
        aria-label="Story"
        className="flex min-w-0 flex-col gap-3 p-4 sm:p-5"
      >
        <div className={wrapRow}>
          <h2 className={headingText}>The story</h2>
          <Pill tone="hold" small>
            Draft · not sealed
          </Pill>
        </div>
        <StoryView story={session.story} />
      </Panel>
      <aside className="flex min-w-0 flex-col gap-4">
        <ActionPanel
          title="If you seal"
          icon={LockIcon}
          rows={[
            {
              label: "Changes",
              body: "The story becomes read-only. Its digest is computed from the markdown.",
            },
            {
              label: "Where",
              body: "Saved in this session. Nothing goes to Jira or your repo.",
            },
            {
              label: "Cannot undo",
              body: "A sealed story cannot change. To change it, start a new session from it.",
            },
            {
              label: "Next",
              body: "Then you can create a run or preview it in Jira.",
            },
          ]}
        >
          <Button
            type="button"
            className="h-10 w-full"
            disabled={busy}
            onClick={onSeal}
          >
            <LockIcon aria-hidden="true" />
            Seal story: cannot be changed
          </Button>
          {canGoBack ? (
            <Button
              type="button"
              variant="ghost"
              className="h-10 w-full"
              disabled={busy}
              onClick={onBack}
            >
              Back to context
            </Button>
          ) : null}
        </ActionPanel>
        <Panel aria-label="Checklist" className="p-4">
          <div className="mb-1 text-sm font-semibold">Before you seal</div>
          <ul>
            {checks.map((check) => (
              <li
                key={check.id}
                className="grid grid-cols-[16px_minmax(0,1fr)] gap-2 py-1"
              >
                <span
                  aria-hidden="true"
                  className={cn(
                    "font-mono font-semibold",
                    toneText[check.ok ? "pass" : "hold"]
                  )}
                >
                  {check.ok ? "+" : "!"}
                </span>
                <span className="text-[13px] leading-5">
                  <b className={toneText[check.ok ? "pass" : "hold"]}>
                    {check.title}
                  </b>{" "}
                  {check.detail}
                </span>
              </li>
            ))}
          </ul>
          {hasAssumption ? (
            <Button
              type="button"
              variant="ghost"
              className="mt-2 h-10"
              disabled={busy}
              onClick={onRevisit}
            >
              Answer an assumption instead
            </Button>
          ) : null}
        </Panel>
        <SessionSummary session={session} />
      </aside>
    </div>
  )
}
