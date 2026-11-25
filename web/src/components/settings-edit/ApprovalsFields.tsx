"use client"

import { defaultDraft, valueOf } from "./draft"
import type { EditProps } from "./fields"
import { StageRow } from "./StageRow"

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
          <StageRow
            key={stage}
            stage={stage}
            label={`${stage} approval`}
            value={value}
            options={choices[stage].map((mode) => ({
              value: mode,
              label: modeLabel[mode] ?? mode,
            }))}
            changed={value !== defaultDraft[stage]}
            disabled={props.disabled}
            onChange={(next) => props.onEdit(stage, next)}
          />
        )
      })}
    </>
  )
}
