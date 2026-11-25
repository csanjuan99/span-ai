"use client"

import { Button } from "@/components/ui/button"
import type { ConfigLine } from "@/lib/types"

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
      className="sticky bottom-4 flex flex-col gap-3 rounded-xl border border-primary bg-card p-5 shadow-md"
    >
      <div className="flex items-center justify-between gap-2">
        <h2 className="text-lg font-semibold">What changes</h2>
        <span className="rounded-md bg-accent px-2 py-0.5 text-xs text-primary">
          {count} unsaved
        </span>
      </div>
      {error !== "" ? (
        <p role="alert" className="text-sm text-pretty text-destructive">
          {error}
        </p>
      ) : (
        <ul className="flex flex-col gap-1.5 font-mono text-sm">
          {lines.map((line) => (
            <li key={line.key} className="flex flex-col">
              <span className="text-muted-foreground">{line.key}</span>
              <span className="break-words text-destructive">
                - {line.before === "" ? "(not set)" : line.before}
              </span>
              <span className="break-words text-pass">
                + {line.after === "" ? "(not set)" : line.after}
              </span>
            </li>
          ))}
        </ul>
      )}
      <dl className="grid gap-1 text-xs text-muted-foreground sm:grid-cols-[6rem_minmax(0,1fr)]">
        <dt>If you save</dt>
        <dd>Only these values change in {folder}/span.yaml.</dd>
        <dt>Not touched</dt>
        <dd>Your comments, the order of keys, and every other setting.</dd>
      </dl>
      <div className="flex flex-wrap gap-2">
        <Button
          disabled={saving || error !== "" || lines.length === 0}
          onClick={onSave}
        >
          Save {count} {count === 1 ? "change" : "changes"}
        </Button>
        <Button variant="outline" disabled={saving} onClick={onDiscard}>
          Discard changes
        </Button>
      </div>
    </section>
  )
}
