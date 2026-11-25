"use client"

import { ChangedBadge } from "@/components/settings-edit/ChangedBadge"
import {
  SelectField,
  type SelectOption,
} from "@/components/settings-edit/SelectField"

/** One option of a stage's select. */
export type StageOption = SelectOption

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
      <SelectField
        mono
        label={label}
        value={value}
        options={options}
        disabled={disabled}
        onChange={onChange}
      />
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
