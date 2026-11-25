"use client"

import { SettingsIcon } from "lucide-react"

import { ChangedBadge } from "@/components/settings-edit/ChangedBadge"
import { Button } from "@/components/ui/button"
import type { ConfigLine } from "@/lib/types"

function DiffRow({
  number,
  sign,
  text,
}: {
  number: number
  sign: "-" | "+"
  text: string
}) {
  return (
    <div
      className={
        sign === "-"
          ? "grid grid-cols-[36px_16px_minmax(0,1fr)] bg-diff-del pr-3.5 whitespace-pre"
          : "grid grid-cols-[36px_16px_minmax(0,1fr)] bg-diff-add pr-3.5 whitespace-pre"
      }
    >
      <span className="pr-2 text-right text-muted-foreground tabular-nums">
        {number}
      </span>
      <span>{sign}</span>
      <span>{text}</span>
    </div>
  )
}

function shown(value: string): string {
  return value === "" ? "(not set)" : value
}

/** What a save would write to span.yaml, with the buttons that save or drop it. */
export function WhatChanges({
  folder,
  count,
  lines,
  error,
  saving,
  onSave,
  onDiscard,
}: {
  folder: string
  count: number
  lines: ConfigLine[]
  error: string
  saving: boolean
  onSave: () => void
  onDiscard: () => void
}) {
  return (
    <section
      aria-label="What changes"
      className="flex flex-col gap-3 rounded-[14px] border border-border bg-card p-4 shadow-sm sm:px-5"
    >
      <div className="flex items-center gap-2">
        <h2 className="text-lg font-semibold">What changes</h2>
        {count > 0 ? <ChangedBadge>{count} unsaved</ChangedBadge> : null}
      </div>
      {error !== "" ? (
        <p role="alert" className="text-sm text-pretty text-destructive">
          {error}
        </p>
      ) : lines.length > 0 ? (
        <div
          role="region"
          aria-label="Code"
          tabIndex={0}
          className="overflow-x-auto rounded-[10px] border border-border bg-surface-code py-1.5 font-mono text-[13px] leading-5"
        >
          {lines.flatMap((line, index) => [
            <DiffRow
              key={`${line.key}-before`}
              number={index + 1}
              sign="-"
              text={`${line.key} ${shown(line.before)}`}
            />,
            <DiffRow
              key={`${line.key}-after`}
              number={index + 1}
              sign="+"
              text={`${line.key} ${shown(line.after)}`}
            />,
          ])}
        </div>
      ) : (
        <p className="text-sm text-pretty text-muted-foreground">
          Nothing to save yet. Change a value and it shows here before anything
          is written.
        </p>
      )}
      <div className="rounded-[10px] bg-muted px-3.5 py-3">
        <div className="mb-1 flex items-center gap-2 text-sm font-semibold">
          <SettingsIcon className="size-4" aria-hidden="true" />
          If you save
        </div>
        <div className="grid grid-cols-[64px_1fr] gap-2.5 border-t border-border py-2">
          <div className="pt-0.5 text-xs font-medium text-muted-foreground">
            Writes to
          </div>
          <div className="font-mono text-xs break-all">{folder}/span.yaml</div>
        </div>
        <div className="grid grid-cols-[64px_1fr] gap-2.5 border-t border-border py-2">
          <div className="pt-0.5 text-xs font-medium text-muted-foreground">
            Not touched
          </div>
          <div className="text-[13px]">
            Your comments, the order of keys, and every other setting.
          </div>
        </div>
      </div>
      <Button
        className="h-10 w-full"
        disabled={saving || error !== "" || lines.length === 0}
        onClick={onSave}
      >
        {count > 0
          ? `Save ${count} ${count === 1 ? "change" : "changes"}`
          : "Save changes"}
      </Button>
      <Button
        className="h-10 w-full"
        variant="outline"
        disabled={saving || count === 0}
        onClick={onDiscard}
      >
        Discard changes
      </Button>
    </section>
  )
}
