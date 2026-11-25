"use client"

import { AlertTriangleIcon, ShieldCheckIcon } from "lucide-react"

import { cn } from "cn"
import type { Audit } from "@/lib/implementation-api"
import { clockOf } from "@/lib/timeline"
import { ActionButton } from "@/components/implementation/ActionButton"

function short(hash: string): string {
  return hash.slice(0, 4)
}

/** The journal of the run with the hash each entry links to, and whether the chain holds. */
export function AuditTab({
  audit,
  note,
  onExport,
}: {
  audit: Audit | null
  note: string
  onExport: () => void
}) {
  if (audit === null) {
    return (
      <p className="rounded-[14px] border border-border bg-card p-5 text-sm text-muted-foreground">
        {note === "" ? "Reading the audit log…" : note}
      </p>
    )
  }
  const broken = !audit.intact
  const total = audit.entries.length
  const before = audit.entries[audit.brokenAt - 2]
  const at = audit.entries[audit.brokenAt - 1]
  return (
    <div className="flex flex-col gap-4">
      {broken ? (
        <section
          role="alert"
          aria-label="Audit chain broken"
          className="flex flex-col gap-2 rounded-[14px] border border-border bg-fail-soft p-4 text-fail sm:p-5"
        >
          <h2 className="flex items-center gap-2 text-lg leading-[26px] font-semibold">
            <AlertTriangleIcon aria-hidden="true" className="size-5" />
            Audit chain broken · entry {audit.brokenAt} of {total}
          </h2>
          <p className="text-[13px] leading-5 text-pretty">
            Entry {audit.brokenAt} does not link to entry {audit.brokenAt - 1}.
            Entries {audit.brokenAt} to {total} cannot be trusted.
          </p>
          {at !== undefined && before !== undefined ? (
            <p className="text-[13px] leading-5 text-pretty">
              Entry {audit.brokenAt} says it follows{" "}
              <span className="font-mono">{short(at.prev)}</span>, but entry{" "}
              {audit.brokenAt - 1} hashes to{" "}
              <span className="font-mono">{short(at.want)}</span>. Something was
              edited or removed after it was written.
            </p>
          ) : null}
          <p className="text-[13px] leading-5 text-pretty">
            <span className="font-medium">Kept </span>All {total} entries,
            exactly as stored. Nothing was deleted or repaired.
          </p>
        </section>
      ) : (
        <section
          aria-label="Audit chain intact"
          className="flex flex-col gap-1 rounded-[14px] border border-border bg-pass-soft p-4 text-pass sm:p-5"
        >
          <h2 className="flex items-center gap-2 text-lg leading-[26px] font-semibold">
            <ShieldCheckIcon aria-hidden="true" className="size-5" />
            Chain intact
          </h2>
          <p className="text-[13px] leading-5 text-pretty">
            All {total} {total === 1 ? "entry links" : "entries link"} to the
            one before.
            {audit.head !== "" ? (
              <>
                {" "}
                Head <span className="font-mono">{short(audit.head)}</span>.
              </>
            ) : null}{" "}
            Nothing was changed or removed after it was written.
          </p>
        </section>
      )}
      <section
        aria-label="Audit log"
        className="overflow-hidden rounded-[14px] border border-border bg-card shadow-sm"
      >
        <div className="flex flex-wrap items-center gap-2 px-4 py-4 sm:px-5">
          <h2 className="text-lg leading-[26px] font-semibold">Audit log</h2>
          <span className="text-xs text-muted-foreground">
            {total} {total === 1 ? "entry" : "entries"} · oldest first · each
            one carries the hash of the one before
          </span>
          <ActionButton className="ml-auto" onClick={onExport}>
            Export audit log
          </ActionButton>
        </div>
        {note !== "" ? (
          <p role="alert" className="px-5 pb-3 text-sm text-destructive">
            {note}
          </p>
        ) : null}
        <div className="grid grid-cols-[48px_minmax(0,1fr)] gap-x-3 border-t border-border bg-muted px-4 py-1.5 text-xs font-medium text-muted-foreground max-sm:hidden sm:grid-cols-[48px_130px_minmax(0,1fr)_150px] sm:px-5">
          <span>Time</span>
          <span>Who</span>
          <span>What</span>
          <span>Links to · hash</span>
        </div>
        {audit.entries.map((entry) => {
          const bad = broken && entry.seq === audit.brokenAt
          const after = broken && entry.seq > audit.brokenAt
          return (
            <div
              key={entry.seq}
              className={cn(
                "grid grid-cols-[48px_minmax(0,1fr)] items-start gap-x-3 gap-y-0.5 border-t border-border px-4 py-2.5 text-[13px] leading-5 sm:grid-cols-[48px_130px_minmax(0,1fr)_150px] sm:px-5",
                bad && "bg-fail-soft",
                after && "text-muted-foreground"
              )}
            >
              <span className="text-muted-foreground tabular-nums">
                {clockOf(entry.time)}
              </span>
              <span className="max-sm:col-start-2">
                {entry.actor === "person" ? "you" : entry.actor}
              </span>
              <span className="min-w-0 text-pretty max-sm:col-start-2">
                #{entry.seq} {entry.what}
                {bad ? (
                  <span className="ml-2 font-medium text-fail">
                    Broken here
                  </span>
                ) : null}
                {after ? (
                  <span className="ml-2 text-xs">Not trusted</span>
                ) : null}
              </span>
              <span className="font-mono text-xs max-sm:col-start-2">
                {short(entry.hash)} ←{" "}
                {entry.prev === "" ? "—" : short(entry.prev)}
                {bad ? (
                  <span className="block text-fail">
                    × {short(entry.want)} does not match
                  </span>
                ) : null}
              </span>
            </div>
          )
        })}
      </section>
    </div>
  )
}
