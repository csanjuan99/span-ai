"use client"

import { useState } from "react"

import { Button } from "@/components/ui/button"
import { archiveShape, deleteShape } from "@/lib/api"
import { reasonText } from "@/lib/guards"
import type { ShapeSession } from "@/lib/types"

/**
 * Archive or restore a session, and delete one that is not sealed after a second click.
 * A sealed story is a record of a decision, so it can only be archived.
 */
export function SessionTidy({
  session,
  workspace,
  title,
  onChanged,
}: {
  session: ShapeSession
  workspace: string
  /** What the buttons name for screen readers. */
  title: string
  onChanged: () => void
}) {
  const [asking, setAsking] = useState(false)
  const [busy, setBusy] = useState(false)
  const [problem, setProblem] = useState("")
  const archived = session.archived === true

  async function change(work: () => Promise<unknown>, fallback: string) {
    setBusy(true)
    setProblem("")
    try {
      await work()
      onChanged()
    } catch (err: unknown) {
      setProblem(reasonText(err, fallback))
      setAsking(false)
    } finally {
      setBusy(false)
    }
  }

  if (asking) {
    return (
      <span
        role="group"
        aria-label={`Delete ${title}?`}
        className="mt-1 inline-flex flex-wrap items-baseline gap-2 text-xs"
      >
        <span className="text-hold">Delete this session and what it kept?</span>
        <Button
          type="button"
          variant="destructive"
          size="xs"
          disabled={busy}
          onClick={() =>
            void change(
              () => deleteShape(session.id, workspace),
              "the session was not deleted"
            )
          }
        >
          Yes, delete
        </Button>
        <Button
          type="button"
          variant="link"
          size="xs"
          disabled={busy}
          onClick={() => setAsking(false)}
        >
          Keep
        </Button>
      </span>
    )
  }
  return (
    <span className="mt-1 inline-flex flex-wrap items-baseline gap-2">
      <Button
        type="button"
        variant="link"
        size="xs"
        disabled={busy}
        aria-label={`${archived ? "Restore" : "Archive"} ${title}`}
        onClick={() =>
          void change(
            () => archiveShape(session.id, workspace, !archived),
            "the session was not changed"
          )
        }
      >
        {archived ? "Restore" : "Archive"}
      </Button>
      {session.status !== "sealed" ? (
        <Button
          type="button"
          variant="link"
          size="xs"
          aria-label={`Delete ${title}`}
          onClick={() => setAsking(true)}
        >
          Delete
        </Button>
      ) : null}
      {problem !== "" ? (
        <span role="alert" className="text-xs text-destructive">
          {problem}
        </span>
      ) : null}
    </span>
  )
}
