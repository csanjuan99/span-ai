"use client"

import { configDefaults } from "@/lib/api"

import { defaultDraft, savedDraft, valueOf } from "./draft"
import { FieldRow } from "./FieldRow"
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
        return (
          <FieldRow
            key={stage}
            label={stage}
            fallback={modeLabel[configDefaults.stages[stage]] ?? ""}
            changed={value !== defaultDraft[stage]}
            unsaved={value !== savedDraft(props.config)[stage]}
          >
            <select
              aria-label={`${stage} approval`}
              className="h-9 w-full rounded-lg border border-input bg-transparent px-2.5 text-sm outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 disabled:opacity-50 sm:max-w-72 dark:bg-input/30"
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
          </FieldRow>
        )
      })}
    </>
  )
}
