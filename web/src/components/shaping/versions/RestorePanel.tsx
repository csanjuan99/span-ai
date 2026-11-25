"use client"

import { HistoryIcon } from "lucide-react"

import { ActionPanel } from "@/components/shaping/session-ui"
import { Button } from "@/components/ui/button"
import {
  tallyText,
  type Revision,
  type RevisionDiff,
} from "@/lib/shape-versions"

/** What leaves the draft when a revision is restored: the diff from it to where the session is now. */
function leaves(back: RevisionDiff | null): string {
  if (back === null) {
    return "Restoring brings the session back to that revision."
  }
  const parts = [
    ["Rules", back.rules],
    ["Examples", back.examples],
    ["Questions", back.questions],
  ] as const
  const said = parts
    .filter(([, t]) => t.added + t.removed + t.changed > 0)
    .map(([name, t]) => `${name}: ${tallyText(t)}`)
  return said.length === 0
    ? "The map is the same; only the story text comes back."
    : `Since then: ${said.join("; ")}.`
}

/**
 * What a restore or a new session from a revision will do, before it is pressed. A sealed
 * story cannot change, so there the restore is off and "new session" is the way forward.
 */
export function RestorePanel({
  target,
  last,
  back,
  sealed,
  busy,
  onRestore,
  onFork,
  onPickOther,
}: {
  target: Revision
  last: Revision
  /** The diff from the target to the current revision, when they differ. */
  back: RevisionDiff | null
  sealed: boolean
  busy: boolean
  onRestore: () => void
  onFork: () => void
  onPickOther: () => void
}) {
  const next = `r${last.n + 1}`
  const isCurrent = target.id === last.id
  return (
    <ActionPanel
      title={sealed ? "What you can do" : `If you restore ${target.id}`}
      icon={HistoryIcon}
      rows={
        sealed
          ? [
              { label: "Compare", body: "Any two revisions, read-only." },
              { label: "Restore", body: "Off. The session is sealed." },
              {
                label: "Instead",
                body: `Start a new session from ${target.id}. The sealed one stays.`,
              },
            ]
          : [
              {
                label: "Changes",
                body: `Adds ${next} with the content of ${target.id}. ${leaves(back)}`,
              },
              {
                label: "Kept",
                body: `r1 to ${last.id} stay in the history. Nothing is deleted. Notes already written to the product context stay written.`,
              },
              {
                label: "Cannot",
                body: "After sealing, nothing can be restored. A sealed story cannot change.",
              },
              {
                label: "Next",
                body: "You answer what the revision left open, or keep the assumptions.",
              },
            ]
      }
    >
      {sealed ? (
        <>
          <Button className="w-full" size="lg" disabled={busy} onClick={onFork}>
            New session from {target.id}
          </Button>
          <Button
            className="w-full"
            size="lg"
            variant="outline"
            disabled
            aria-disabled="true"
          >
            Restore {target.id} as {next}: sealed
          </Button>
        </>
      ) : (
        <>
          <Button
            className="w-full"
            size="lg"
            disabled={busy || isCurrent}
            onClick={onRestore}
          >
            {isCurrent
              ? `${target.id} is already current`
              : `Restore ${target.id} as ${next}`}
          </Button>
          <Button
            className="w-full"
            size="lg"
            variant="ghost"
            disabled={busy}
            onClick={onFork}
          >
            New session from {target.id}
          </Button>
        </>
      )}
      <Button
        className="w-full"
        size="lg"
        variant="ghost"
        onClick={onPickOther}
      >
        Pick other revisions
      </Button>
    </ActionPanel>
  )
}
