"use client"

import { UndoIcon } from "lucide-react"

import { ActionPanel } from "@/components/shaping/session-ui"
import { Button } from "@/components/ui/button"
import type { UndoPlan } from "@/lib/shape-flows"

/**
 * What undoing a step changes, said before it is pressed. A step others depend on offers
 * both ways: only this one is refused by the server, so the choice is to undo them together.
 */
export function UndoPanel({
  plan,
  busy,
  onUndo,
  onCancel,
}: {
  plan: UndoPlan
  busy: boolean
  onUndo: (cascade: boolean) => void
  onCancel: () => void
}) {
  const dependents = plan.dependents
  return (
    <ActionPanel
      title={`Undo ${plan.target.id} · ${plan.target.summary}`}
      icon={UndoIcon}
      rows={[
        {
          label: "Now",
          body: plan.now,
        },
        { label: "After", body: plan.after },
        {
          label: "Changes",
          body: (
            <ul className="list-disc pl-4">
              {plan.changes.map((line) => (
                <li key={line}>{line}</li>
              ))}
            </ul>
          ),
        },
        { label: "Kept", body: plan.kept },
        { label: "Next", body: plan.next },
      ]}
    >
      {dependents.length > 0 ? (
        <div
          role="note"
          className="rounded-[10px] border border-border p-3 text-[13px]"
        >
          <b>
            {dependents.length} later{" "}
            {dependents.length === 1 ? "step depends" : "steps depend"} on this
            one:
          </b>
          <ul className="mt-1 list-disc pl-4">
            {dependents.map((step) => (
              <li key={step.id}>
                <span className="font-mono">{step.id}</span> {step.summary}
              </li>
            ))}
          </ul>
        </div>
      ) : null}
      <div className="flex flex-wrap gap-2">
        {dependents.length === 0 ? (
          <Button type="button" disabled={busy} onClick={() => onUndo(false)}>
            Undo {plan.target.id}
          </Button>
        ) : (
          <Button type="button" disabled={busy} onClick={() => onUndo(true)}>
            Undo {plan.target.id} and {dependents.length} that depend on it
          </Button>
        )}
        <Button type="button" variant="ghost" onClick={onCancel}>
          Cancel
        </Button>
      </div>
    </ActionPanel>
  )
}
