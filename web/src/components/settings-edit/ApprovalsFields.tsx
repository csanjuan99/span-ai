"use client"

import { ChangedBadge } from "./ChangedBadge"
import { defaultDraft, valueOf } from "./draft"
import type { EditProps } from "./fields"

const stages = ["open", "merge", "promote", "plan", "conflict"] as const

const choices: Record<(typeof stages)[number], string[]> = {
  open: ["auto", "hold"],
  merge: ["auto", "hold"],
  promote: ["auto", "hold"],
  plan: ["auto", "conditional", "hold"],
  conflict: ["hold", "auto"],
}

const modeLabel: Record<string, string> = {
  auto: "Runs on its own",
  hold: "Asks me",
  conditional: "Asks me when the story is deep",
}

/** Which stages stop and wait for a yes, editable. Saved to span.yaml. */
export function ApprovalsFields({ props }: { props: EditProps }) {
  return (
    <>
      {stages.map((stage) => {
        const value = valueOf(props.config, props.draft, stage)
        const changed = value !== defaultDraft[stage]
        return (
          <div
            key={stage}
            className="grid grid-cols-[96px_minmax(0,1fr)] items-center gap-3 border-t border-border py-2 first:border-t-0 sm:grid-cols-[120px_minmax(0,1fr)_auto]"
          >
            <span className="text-[13px] font-medium">{stage}</span>
            <select
              aria-label={`${stage} approval`}
              className="h-8 w-full rounded-[10px] border border-input bg-card px-2.5 text-[13px] outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 disabled:opacity-50 sm:max-w-72"
              disabled={props.disabled}
              value={value}
              onChange={(event) => props.onEdit(stage, event.target.value)}
            >
              {choices[stage].map((mode) => (
                <option key={mode} value={mode}>
                  {modeLabel[mode]}
                </option>
              ))}
            </select>
            <span className="col-start-2 sm:col-start-3">
              {changed ? (
                <ChangedBadge>Changed</ChangedBadge>
              ) : (
                <span className="text-xs text-muted-foreground">Default</span>
              )}
            </span>
          </div>
        )
      })}
    </>
  )
}
