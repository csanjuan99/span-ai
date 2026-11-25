"use client"

import { RadioCard } from "@/components/list-controls/RadioCard"
import { Panel, Pill } from "@/components/shaping/session-ui"
import type { Conflict } from "@/lib/shape-compare"
import { cardBody, wrapRow } from "@/components/shaping/styles"

/** One disagreement between sessions: each option names who says it, and you pick the one that wins. */
export function ConflictCard({
  conflict,
  picked,
  onPick,
}: {
  conflict: Conflict
  picked: string | undefined
  onPick: (session: string) => void
}) {
  return (
    <Panel aria-label={`Conflict ${conflict.id}`} className={cardBody}>
      <div className={wrapRow}>
        <span aria-hidden="true" className="font-mono font-semibold text-hold">
          ~
        </span>
        <span className="text-sm font-semibold">
          {conflict.id} · {conflict.title}
        </span>
        <Pill tone={picked === undefined ? "hold" : "pass"} small>
          {picked === undefined ? "Needs a decision" : "Decided"}
        </Pill>
      </div>
      <fieldset className="flex flex-col gap-2">
        <legend className="sr-only">Which one wins for {conflict.id}</legend>
        {conflict.options.map((option) => {
          const owner = option.members[0]?.session ?? ""
          const chosen = picked === owner
          return (
            <RadioCard
              key={`${owner}${option.text}`}
              roomy
              name={conflict.id}
              checked={chosen}
              onSelect={() => onPick(owner)}
              title={option.text}
            >
              {option.members
                .map((m) => `${m.ref} · ${m.item} · ${m.origin}`)
                .join("  ·  ")}
            </RadioCard>
          )
        })}
      </fieldset>
    </Panel>
  )
}
