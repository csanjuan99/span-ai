"use client"

import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import type { ProjectConfig } from "@/lib/types"

import {
  defaultDraft,
  savedDraft,
  valueOf,
  type Draft,
  type DraftKey,
} from "./draft"
import { FieldRow } from "./FieldRow"

/** What every editable section is given: the saved settings, what was typed, and how to type. */
export interface EditProps {
  config: ProjectConfig
  draft: Draft
  disabled: boolean
  onEdit: (key: DraftKey, value: string) => void
}

function stateOf(props: EditProps, key: DraftKey) {
  const value = valueOf(props.config, props.draft, key)
  return {
    value,
    changed: value.trim() !== defaultDraft[key],
    unsaved:
      props.draft[key] !== undefined &&
      props.draft[key].trim() !== savedDraft(props.config)[key],
  }
}

/** A setting typed on one line. */
export function TextField({
  props,
  name,
  label,
  fallback,
  placeholder,
}: {
  props: EditProps
  name: DraftKey
  label: string
  fallback: string
  placeholder?: string
}) {
  const state = stateOf(props, name)
  return (
    <FieldRow
      label={label}
      fallback={fallback}
      changed={state.changed}
      unsaved={state.unsaved}
    >
      <Input
        aria-label={label}
        className="h-auto rounded-[10px] border-input bg-card px-3 py-2.5 font-mono text-sm shadow-sm dark:bg-card"
        disabled={props.disabled}
        placeholder={placeholder}
        value={state.value}
        onChange={(event) => props.onEdit(name, event.target.value)}
      />
    </FieldRow>
  )
}

/** A list of checks, one command per line. */
export function CommandsField({
  props,
  name,
  label,
}: {
  props: EditProps
  name: DraftKey
  label: string
}) {
  const state = stateOf(props, name)
  return (
    <FieldRow
      label={label}
      fallback="none"
      changed={state.changed}
      unsaved={state.unsaved}
    >
      <Textarea
        aria-label={label}
        className="rounded-[10px] border-input bg-surface-code px-2.5 py-2 font-mono text-xs"
        disabled={props.disabled}
        placeholder="One command per line"
        rows={2}
        spellCheck={false}
        value={state.value}
        onChange={(event) => props.onEdit(name, event.target.value)}
      />
    </FieldRow>
  )
}
