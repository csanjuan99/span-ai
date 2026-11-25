"use client"

import { ChangedBadge } from "@/components/settings-edit/ChangedBadge"

/** One option of a stage's select. */
export interface StageOption {
  value: string
  label: string
}

/**
 * One stage and the choice made for it: its name, a select, and whether the choice moved
 * from the default. The agents and approvals sections both list stages this way.
 */
export function StageRow({
  stage,
  hint,
  label,
  value,
  options,
  changed,
  disabled,
  onChange,
}: {
  stage: string
  /** What the stage does, in a few words. */
  hint?: string
  /** The accessible name of the select. */
  label: string
  value: string
  options: StageOption[]
  /** The value differs from the default. */
  changed: boolean
  disabled: boolean
  onChange: (value: string) => void
}) {
  return (
    <div className="grid grid-cols-[96px_minmax(0,1fr)] items-center gap-3 border-t border-border py-2 first:border-t-0 sm:grid-cols-[120px_minmax(0,1fr)_auto]">
      <span className="flex flex-col">
        <span className="text-[13px] font-medium">{stage}</span>
        {hint !== undefined ? (
          <span className="text-xs text-muted-foreground">{hint}</span>
        ) : null}
      </span>
      <select
        aria-label={label}
        className="h-8 w-full rounded-[10px] border border-input bg-card px-2.5 text-[13px] outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 disabled:opacity-50 sm:max-w-72"
        disabled={disabled}
        value={value}
        onChange={(event) => onChange(event.target.value)}
      >
        {options.map((option) => (
          <option key={option.value} value={option.value}>
            {option.label}
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
}
