"use client"

import { CheckCheckIcon } from "lucide-react"

import { ActionPanel } from "@/components/shaping/session-ui"
import { Button } from "@/components/ui/button"
import type { BoardCard, BulkAction, BulkResult } from "@/lib/shape-flows"

/** The words of one action: its button, and what it does before it is pressed. */
const words: Record<
  BulkAction,
  { button: (n: number) => string; changes: string; kept: string }
> = {
  assume: {
    button: (n) =>
      `Answer ${n} ${n === 1 ? "session" : "sessions"} with the assumption`,
    changes: "Each open question becomes assumed, with what the agent chose.",
    kept: "You can still answer them later, until you seal.",
  },
  seal: {
    button: (n) => `Seal ${n} ${n === 1 ? "story" : "stories"}`,
    changes:
      "The stories are kept with their digest and cannot be changed afterwards.",
    kept: "Nothing else changes.",
  },
  archive: {
    button: (n) => `Archive ${n}`,
    changes: "The sessions leave the list and the board.",
    kept: "Their story and digest stay as they are. Show archived sessions to bring them back.",
  },
  unarchive: {
    button: (n) => `Unarchive ${n}`,
    changes: "The sessions show in the list and the board again.",
    kept: "Nothing else changes.",
  },
}

/** Which actions a selection allows: seal only when every chosen session is ready. */
export function allowed(chosen: BoardCard[]): BulkAction[] {
  const actions: BulkAction[] = []
  if (
    chosen.some(
      (card) => card.status === "interviewing" && card.openQuestions > 0
    )
  ) {
    actions.push("assume")
  }
  if (chosen.length > 0 && chosen.every((card) => card.status === "ready")) {
    actions.push("seal")
  }
  actions.push("archive")
  return actions
}

/**
 * The bar of a selection: what can be done to it, and, once one action is picked, what the
 * server says it would change in each session before anything changes.
 */
export function BulkBar({
  chosen,
  plan,
  action,
  busy,
  onPlan,
  onApply,
  onCancel,
  onClear,
}: {
  chosen: BoardCard[]
  plan: BulkResult | null
  action: BulkAction | null
  busy: boolean
  onPlan: (action: BulkAction) => void
  onApply: () => void
  onCancel: () => void
  onClear: () => void
}) {
  if (chosen.length === 0) {
    return null
  }
  const open = chosen.reduce((sum, card) => sum + card.openQuestions, 0)
  return (
    <div className="flex flex-col gap-3">
      <div
        role="region"
        aria-label="Bulk actions"
        className="flex flex-wrap items-center gap-3 rounded-[14px] border border-border bg-card p-3 shadow-sm"
      >
        <span className="text-[13px] font-semibold">
          {chosen.length} selected
        </span>
        <span className="text-[13px]">
          {open} open {open === 1 ? "question" : "questions"}
        </span>
        <span className="ml-auto flex flex-wrap gap-2">
          {allowed(chosen).map((name, index) => (
            <Button
              key={name}
              type="button"
              variant={index === 0 ? "default" : "outline"}
              disabled={busy}
              onClick={() => onPlan(name)}
            >
              {words[name].button(chosen.length)}
            </Button>
          ))}
          <Button type="button" variant="ghost" onClick={onClear}>
            Clear
          </Button>
        </span>
      </div>
      {plan !== null && action !== null ? (
        <ActionPanel
          title={`If you ${action === "assume" ? "answer with the assumption" : action}`}
          icon={CheckCheckIcon}
          rows={[
            {
              label: "Changes",
              body: (
                <ul className="flex flex-col gap-1">
                  {plan.items.map((item) => (
                    <li key={item.id}>
                      <span className="font-mono">{item.ref}</span>{" "}
                      {item.code !== ""
                        ? `cannot: ${item.message}`
                        : item.skipped
                          ? `nothing to do: ${item.message}`
                          : item.changed.join("; ")}
                    </li>
                  ))}
                </ul>
              ),
            },
            { label: "Kept", body: words[action].kept },
            { label: "Then", body: words[action].changes },
          ]}
        >
          <div className="flex flex-wrap gap-2">
            <Button
              type="button"
              disabled={busy || plan.done === 0}
              onClick={onApply}
            >
              {words[action].button(plan.done)}
            </Button>
            <Button type="button" variant="ghost" onClick={onCancel}>
              Cancel
            </Button>
          </div>
        </ActionPanel>
      ) : null}
    </div>
  )
}
