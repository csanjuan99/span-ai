"use client"

import { useState } from "react"

import { useBusyAction } from "@/hooks/use-busy-action"
import { reasonText } from "@/lib/guards"

import { ApiError, createShapeIssue, previewShapeIssue } from "@/lib/api"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import type { ShapePreview, ShapeSession } from "@/lib/types"
import { PromptHeading } from "@/components/shell/PromptHeading"

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

function PreviewCard({ preview }: { preview: ShapePreview }) {
  return (
    <div className="flex flex-col gap-2 border border-border p-3 text-sm">
      <p>
        <span className="text-muted-foreground">Project </span>
        {preview.project}
      </p>
      <p>
        <span className="text-muted-foreground">Board </span>
        <span>{preview.board.name}</span>
      </p>
      <p>
        <span className="text-muted-foreground">Backlog statuses </span>
        {preview.backlogStatuses.map((status) => (
          <span key={status} className="mr-2">
            {status}
          </span>
        ))}
      </p>
      <p>{preview.summary}</p>
      <pre className="whitespace-pre-wrap text-muted-foreground">
        {preview.description}
      </pre>
    </div>
  )
}

/** Preview, then create. A session that already has a key shows the link and nothing else. */
export function JiraPanel({
  session,
  onSession,
  onReload,
}: {
  session: ShapeSession
  onSession: (next: ShapeSession) => void
  onReload: () => Promise<void>
}) {
  const [project, setProject] = useState("")
  const [board, setBoard] = useState("")
  const [preview, setPreview] = useState<ShapePreview | null>(null)
  const [note, setNote] = useState("")

  const { busy, run: guarded } = useBusyAction({
    onStart: () => setNote(""),
    onError: (err) => {
      setNote(jiraFailure(err))
      if (err instanceof ApiError && err.code === "jira_not_in_backlog") {
        void onReload()
      }
    },
  })

  const created = session.jira
  return (
    <section className="flex flex-col gap-3">
      <PromptHeading>jira</PromptHeading>
      {created !== null ? (
        <p className="text-sm">
          <a
            className="text-primary underline-offset-4 hover:underline"
            href={created.url}
          >
            {created.key}
          </a>{" "}
          <span className="text-muted-foreground">{created.status}</span>
        </p>
      ) : (
        <>
          <p className="text-sm text-pretty text-muted-foreground">
            Span creates the story only in the backlog of the project's Kanban
            board, unassigned. Check it first.
          </p>
          <div className="flex flex-wrap items-end gap-3">
            <div className="flex flex-col gap-2">
              <Label htmlFor="jira-project">Project key</Label>
              <Input
                id="jira-project"
                className="w-40"
                value={project}
                onChange={(event) => setProject(event.target.value)}
              />
            </div>
            <div className="flex flex-col gap-2">
              <Label htmlFor="jira-board">Board id (optional)</Label>
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
              variant="secondary"
              disabled={busy || project.trim() === ""}
              onClick={() =>
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
            >
              Preview
            </Button>
          </div>
          {preview !== null ? (
            <>
              <PreviewCard preview={preview} />
              <Button
                type="button"
                className="self-start"
                disabled={busy}
                onClick={() =>
                  void guarded(async () => {
                    onSession(
                      await createShapeIssue(
                        session.id,
                        session.workspace,
                        preview.project,
                        board
                      )
                    )
                  })
                }
              >
                Create in Jira
              </Button>
            </>
          ) : null}
        </>
      )}
      {note !== "" ? (
        <p role="alert" className="text-sm text-pretty text-destructive">
          {note}
        </p>
      ) : null}
    </section>
  )
}
