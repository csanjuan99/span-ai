"use client"

import { Check } from "lucide-react"
import { useId, useState } from "react"

import { kindLabel } from "@/lib/conflict"
import type { ConflictGroup } from "@/lib/types"
import { CompareOptions } from "@/components/conflict/CompareOptions"
import { ConflictDiff } from "@/components/conflict/ConflictDiff"
import { ConflictOptionCard } from "@/components/conflict/ConflictOptionCard"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"

/** Files listed before "+N more": a group of fifty lockfiles should not push the cards away. */
const fileCap = 5

function FileList({ files }: { files: string[] }) {
  const [all, setAll] = useState(false)
  const shown = all ? files : files.slice(0, fileCap)
  const hidden = files.length - shown.length
  return (
    <>
      {shown.map((file) => (
        <span
          key={file}
          className="rounded-md bg-muted px-1.5 py-0.5 font-mono text-xs break-all"
        >
          {file}
        </span>
      ))}
      {hidden > 0 ? (
        <Button
          type="button"
          variant="ghost"
          size="xs"
          onClick={() => setAll(true)}
        >
          +{hidden} more
        </Button>
      ) : null}
    </>
  )
}

/**
 * One group of conflicting files and its options. The fieldset makes the options one radio
 * group, named by the group summary, so exactly one option is chosen per group. What the
 * chosen option writes shows under it.
 */
export function ConflictGroupSection({
  group,
  index,
  total,
  chosen,
  onChoose,
}: {
  group: ConflictGroup
  index: number
  total: number
  chosen: string
  onChoose: (option: string) => void
}) {
  const id = useId()
  const [compare, setCompare] = useState(false)
  const picked = group.options.find((option) => option.id === chosen)
  return (
    <section className="flex flex-col gap-3 rounded-[14px] border border-border bg-card p-4">
      <div className="flex flex-wrap items-center gap-2">
        <span className="text-sm font-semibold">
          Group {index + 1} of {total}
        </span>
        <FileList files={group.files} />
        <Badge variant="outline">{kindLabel(group.kind)}</Badge>
        <Button
          type="button"
          variant="ghost"
          size="xs"
          aria-pressed={compare}
          onClick={() => setCompare(!compare)}
        >
          Compare options
        </Button>
        {picked === undefined ? null : (
          <Badge variant="secondary" className="ml-auto bg-pass-soft text-pass">
            <Check aria-hidden="true" />
            Chosen
          </Badge>
        )}
      </div>
      <p
        id={`${id}-summary`}
        className="text-sm text-pretty text-muted-foreground"
      >
        {group.summary}
      </p>
      <fieldset aria-labelledby={`${id}-summary`}>
        <ul className="grid gap-2 sm:grid-cols-2 xl:grid-cols-3">
          {group.options.map((option) => (
            <ConflictOptionCard
              key={option.id}
              name={`${id}-choice`}
              option={option}
              checked={chosen === option.id}
              onChoose={onChoose}
            />
          ))}
        </ul>
      </fieldset>
      {compare ? (
        <CompareOptions group={group} chosen={chosen} onChoose={onChoose} />
      ) : null}
      {picked === undefined || compare ? null : (
        <>
          <p className="text-xs text-muted-foreground">
            Result if you {picked.title.toLowerCase()}
          </p>
          <ConflictDiff diff={picked.diff} />
        </>
      )}
    </section>
  )
}
