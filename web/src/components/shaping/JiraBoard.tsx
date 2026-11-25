"use client"

import { useState } from "react"
import {
  CheckCircle2Icon,
  ExternalLinkIcon,
  PlayIcon,
  ShieldAlertIcon,
} from "lucide-react"

import { useBusyAction } from "@/hooks/use-busy-action"
import { ApiError, createShapeIssue, previewShapeIssue } from "@/lib/api"
import { reasonText } from "@/lib/guards"
import { issuePayload } from "@/lib/shape-session"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import {
  ActionPanel,
  CodeBlock,
  Eyebrow,
  Panel,
  Pill,
  twoColumns,
} from "@/components/shaping/session-ui"
import { SessionSummary } from "@/components/shaping/SessionSummary"
import { StoryView } from "@/components/shaping/StoryView"
import type { ShapePreview, ShapeSession } from "@/lib/types"
import {
  errorText,
  headingText,
  labelText,
  wrapRow,
} from "@/components/shaping/styles"

const leads: Record<string, string> = {
  jira_backlog_disabled:
    "The backlog is not enabled on this board, so Span creates nothing.",
  jira_board_ambiguous:
    "This project has more than one Kanban board. Type a board id.",
  jira_not_in_backlog:
    "The issue exists in Jira but could not be moved into the backlog.",
}

/** What the person reads when Jira refuses: the case in plain words, then the server message. */
export function jiraFailure(err: unknown): string {
  if (err instanceof ApiError) {
    const lead = leads[err.code]
    return lead === undefined ? err.message : `${lead} ${err.message}`
  }
  return reasonText(err, "Jira did not answer")
}

function Row({
  label,
  children,
}: {
  label: string
  children: React.ReactNode
}) {
  return (
    <div className="grid gap-2 border-t border-border py-3 sm:grid-cols-[160px_minmax(0,1fr)] sm:gap-4">
      <Eyebrow className="pt-0.5">{label}</Eyebrow>
      <div className="text-[13px] leading-5 break-words">{children}</div>
    </div>
  )
}

/**
 * Jira for a sealed story: the lookup and the exact request first, a confirm button that is the
 * only thing that sends it, and the issue once it exists. A refusal keeps the story and the preview.
 */
export function JiraBoard({
  session,
  busyOutside,
  onSession,
  onReload,
  onBack,
  onRun,
}: {
  session: ShapeSession
  busyOutside: boolean
  onSession: (next: ShapeSession) => void
  onReload: () => Promise<void>
  onBack: () => void
  onRun: () => void
}) {
  const [project, setProject] = useState("")
  const [board, setBoard] = useState("")
  const [preview, setPreview] = useState<ShapePreview | null>(null)
  const [failure, setFailure] = useState("")
  const { busy, run: guarded } = useBusyAction({
    onStart: () => setFailure(""),
    onError: (err) => {
      setFailure(jiraFailure(err))
      if (err instanceof ApiError && err.code === "jira_not_in_backlog") {
        void onReload()
      }
    },
  })
  const created = session.jira
  const locked = busy || busyOutside

  function load() {
    void guarded(async () => {
      setPreview(
        await previewShapeIssue(
          session.id,
          session.workspace,
          project.trim(),
          board
        )
      )
    })
  }

  function create(shown: ShapePreview) {
    void guarded(async () => {
      onSession(
        await createShapeIssue(
          session.id,
          session.workspace,
          shown.project,
          board
        )
      )
    })
  }

  const story = (
    <Panel
      aria-label="Story"
      className="flex min-w-0 flex-col gap-3 p-4 sm:p-5"
    >
      <div className={wrapRow}>
        <h2 className={headingText}>The story</h2>
        <Pill tone="pass" small>
          Sealed · read-only
        </Pill>
      </div>
      <StoryView story={session.story} />
    </Panel>
  )

  if (created !== null) {
    return (
      <div className="flex flex-col gap-4">
        <Panel className="flex flex-wrap items-center gap-3 border-pass bg-pass-soft px-4 py-2.5 text-pass sm:px-5">
          <CheckCircle2Icon className="size-4 shrink-0" aria-hidden="true" />
          <span className="text-[13px] leading-5">
            <b>{`Created ${created.key}.`}</b>
            {created.status !== "" ? ` It is in ${created.status}.` : ""}
          </span>
        </Panel>
        {failure !== "" ? (
          <p role="alert" className={errorText}>
            {failure}
          </p>
        ) : null}
        <div className={twoColumns}>
          {story}
          <aside className="flex min-w-0 flex-col gap-4">
            <Panel aria-label="Jira link" className="flex flex-col gap-3 p-4">
              <div className="text-sm font-semibold">Jira</div>
              <Row label="Key">
                <span className="font-mono text-xs">{created.key}</span>
              </Row>
              {created.status !== "" ? (
                <Row label="Status">
                  <Pill tone="mut" small>
                    {created.status}
                  </Pill>
                </Row>
              ) : null}
              <Row label="Link">
                <a
                  className="text-primary underline-offset-4 hover:underline"
                  href={created.url}
                >
                  {created.url.replace(/^https?:\/\//, "")}
                </a>
              </Row>
              <Button
                type="button"
                className="h-10 w-full"
                disabled={locked}
                onClick={onRun}
              >
                <PlayIcon aria-hidden="true" />
                Create run
              </Button>
              <a
                className="inline-flex h-10 w-full items-center justify-center gap-2 rounded-[10px] border border-input bg-card text-sm font-medium"
                href={created.url}
              >
                Open in Jira
                <ExternalLinkIcon className="size-4" aria-hidden="true" />
              </a>
              <Button
                type="button"
                variant="ghost"
                className="h-10 w-full"
                onClick={onBack}
              >
                Back to story
              </Button>
            </Panel>
            <SessionSummary session={session} />
          </aside>
        </div>
      </div>
    )
  }

  return (
    <div className={twoColumns}>
      <div className="flex min-w-0 flex-col gap-4">
        <Panel aria-label="Jira preview" className="flex flex-col p-4 sm:p-5">
          <div className="mb-2 flex flex-wrap items-center gap-2">
            <h2 className={headingText}>Jira preview</h2>
            <Pill tone="pass" small>
              Nothing sent yet
            </Pill>
          </div>
          <p className="mb-3 text-sm text-pretty text-muted-foreground">
            Span creates the story only in the backlog of the project's Kanban
            board, unassigned. Check it first.
          </p>
          <div className="flex flex-wrap items-end gap-3 pb-3">
            <div className="flex flex-col gap-1.5">
              <label htmlFor="jira-project" className={labelText}>
                Project key
              </label>
              <Input
                id="jira-project"
                className="w-40"
                value={project}
                onChange={(event) => setProject(event.target.value)}
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <label htmlFor="jira-board" className={labelText}>
                Board id (optional)
              </label>
              <Input
                id="jira-board"
                className="w-40"
                inputMode="numeric"
                value={board}
                onChange={(event) => setBoard(event.target.value)}
              />
            </div>
            <Button
              type="button"
              variant="outline"
              className="h-10"
              disabled={locked || project.trim() === ""}
              onClick={load}
            >
              Preview
            </Button>
          </div>
          {preview !== null ? (
            <>
              <Row label="Project">{preview.project}</Row>
              <Row label="Board">{preview.board.name}</Row>
              <Row label="Issue type">Story</Row>
              <Row label="Status">
                <ul className="flex flex-col gap-1">
                  {preview.backlogStatuses.map((status) => (
                    <li key={status}>{status}</li>
                  ))}
                </ul>
                <p className="mt-1.5 text-xs text-muted-foreground">
                  Backlog statuses of this board. Span lands the issue in one of
                  them, never in To Do or In Progress.
                </p>
              </Row>
              <Row label="Summary">{preview.summary}</Row>
              <Eyebrow className="mt-3 mb-1.5">Exact payload</Eyebrow>
              <CodeBlock label="Payload">
                {issuePayload(preview)
                  .split("\n")
                  .map((line, index) => (
                    <div key={index} className="px-3.5 whitespace-pre-wrap">
                      {line}
                    </div>
                  ))}
              </CodeBlock>
              <p className="mt-1.5 text-xs text-muted-foreground">
                The description is sent as an Atlassian document made from this
                markdown.
              </p>
            </>
          ) : null}
        </Panel>
        {failure !== "" ? (
          <Panel role="alert" className="overflow-hidden">
            <div className="flex items-center gap-2 bg-fail-soft px-4 py-2.5 text-fail sm:px-5">
              <ShieldAlertIcon className="size-4" aria-hidden="true" />
              <span className={labelText}>Jira refused</span>
            </div>
            <div className="flex flex-col gap-3 p-4 sm:p-6">
              <p className="text-sm leading-[22px] text-pretty">{failure}</p>
              <p className="text-sm leading-[22px] text-pretty text-muted-foreground">
                Nothing was created. The story is sealed and the preview is
                unchanged.
              </p>
              <Button
                type="button"
                className="h-10 self-start"
                onClick={() => {
                  setPreview(null)
                  setFailure("")
                }}
              >
                Choose another project
              </Button>
            </div>
          </Panel>
        ) : null}
      </div>
      <aside className="flex min-w-0 flex-col gap-4">
        <ActionPanel
          title="If you create it"
          icon={CheckCircle2Icon}
          rows={[
            {
              label: "Changes",
              body:
                preview === null
                  ? "Creates 1 issue. No other Jira field changes."
                  : `Creates 1 issue in ${preview.project}. No other Jira field changes.`,
            },
            {
              label: "Where",
              body:
                preview === null
                  ? "The backlog of the project's Kanban board."
                  : `${preview.project} · ${preview.board.name}`,
            },
            {
              label: "Kept",
              body: "The sealed story and its digest do not change.",
            },
            {
              label: "Next",
              body: "The key and link are saved on the story. You can still create a run.",
            },
          ]}
        >
          <Button
            type="button"
            className="h-10 w-full"
            disabled={locked || preview === null}
            onClick={() => preview !== null && create(preview)}
          >
            Create in Jira
          </Button>
          <Button
            type="button"
            variant="ghost"
            className="h-10 w-full"
            onClick={onBack}
          >
            Back to story
          </Button>
        </ActionPanel>
        <SessionSummary session={session} />
      </aside>
    </div>
  )
}
