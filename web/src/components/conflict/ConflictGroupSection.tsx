import { useId, useState } from "react"

import { kindLabel } from "@/lib/conflict"
import type { ConflictGroup } from "@/lib/types"
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
    <div className="flex flex-col gap-1">
      <ul className="flex flex-col gap-0.5 font-mono text-sm tabular-nums">
        {shown.map((file) => (
          <li key={file} className="break-all">
            {file}
          </li>
        ))}
      </ul>
      {hidden > 0 ? (
        <Button type="button" variant="ghost" size="xs" className="self-start" onClick={() => setAll(true)}>
          +{hidden} more
        </Button>
      ) : null}
    </div>
  )
}

/**
 * One group of conflicting files and its options. The fieldset makes the options one
 * radio group, named by the group summary, so exactly one option is chosen per group.
 */
export function ConflictGroupSection({
  group,
  chosen,
  onChoose,
}: {
  group: ConflictGroup
  chosen: string
  onChoose: (option: string) => void
}) {
  const id = useId()
  return (
    <fieldset aria-labelledby={`${id}-summary`} className="flex flex-col gap-3 border-t border-border pt-4">
      <Badge variant="outline" className="self-start">
        {kindLabel(group.kind)}
      </Badge>
      <FileList files={group.files} />
      <p id={`${id}-summary`} className="text-pretty">
        {group.summary}
      </p>
      <ul className="grid gap-3 sm:grid-cols-2">
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
  )
}
