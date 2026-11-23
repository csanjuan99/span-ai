"use client"

import { useState } from "react"

import { Button } from "@/components/ui/button"
import { deleteRun } from "@/lib/api"

/**
 * Deletes one run's record after a second click. Only the record goes: the branch, the pull
 * request and any worktree stay, so the confirmation says so.
 */
export function DeleteRun({ id, label, onDeleted }: { id: string; label: string; onDeleted: (id: string) => void }) {
  const [asking, setAsking] = useState(false)
  const [busy, setBusy] = useState(false)
  const [problem, setProblem] = useState("")

  async function confirm() {
    setBusy(true)
    setProblem("")
    try {
      await deleteRun(id)
      onDeleted(id)
    } catch (err: unknown) {
      setProblem(err instanceof Error ? err.message : "the run was not deleted")
      setAsking(false)
    } finally {
      setBusy(false)
    }
  }

  if (!asking) {
    return (
      <span className="inline-flex flex-wrap items-baseline gap-2">
        <Button type="button" variant="link" size="xs" aria-label={`Delete ${label}`} onClick={() => setAsking(true)}>
          Delete
        </Button>
        {problem !== "" ? <span className="text-sm text-destructive">{problem}</span> : null}
      </span>
    )
  }
  return (
    <span className="inline-flex flex-wrap items-baseline gap-2 text-sm" role="group" aria-label={`Delete ${label}?`}>
      <span className="text-hold">Delete this run&apos;s record? The branch and any worktree stay.</span>
      <Button type="button" variant="destructive" size="xs" disabled={busy} onClick={() => void confirm()}>
        Yes, delete
      </Button>
      <Button type="button" variant="link" size="xs" disabled={busy} onClick={() => setAsking(false)}>
        Keep
      </Button>
    </span>
  )
}
