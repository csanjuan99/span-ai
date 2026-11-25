"use client"

import { SettingsIcon } from "lucide-react"

import { ChangedBadge } from "@/components/settings-edit/ChangedBadge"
import { Button } from "@/components/ui/button"
import { Consequences } from "@/components/status-block/Consequences"
import type { ConfigEffect, ConfigLine } from "@/lib/types"
import { cn } from "@/lib/utils"

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
      className={cn(
        "grid grid-cols-[36px_16px_minmax(0,1fr)] pr-3.5 whitespace-pre",
        sign === "-" ? "bg-diff-del" : "bg-diff-add"
      )}
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
  effects,
  error,
  saving,
  onSave,
  onDiscard,
}: {
  folder: string
  count: number
  lines: ConfigLine[]
  effects: ConfigEffect[]
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
      <Consequences
        title="If you save"
        icon={SettingsIcon}
        rows={[
          ...effects.map((effect) => ({
            label: effect.label,
            body: effect.text,
          })),
          { label: "Writes to", body: `${folder}/span.yaml` },
          {
            label: "Not touched",
            body: "Your comments, the order of keys, sealed stages and past proof.",
          },
        ]}
      />
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
      <p className="text-xs text-pretty text-muted-foreground">
        Save token, Test and Close session act at once. They are not in this
        list.
      </p>
    </section>
  )
}
