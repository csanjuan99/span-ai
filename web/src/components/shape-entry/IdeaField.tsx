"use client"

import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import { captionText, labelText } from "@/components/shaping/styles"

/** The idea box: a labelled textarea in the design's rounded field. */
export function IdeaField({
  id,
  rows,
  value,
  hint,
  onChange,
}: {
  id: string
  rows: number
  value: string
  /** Small text under the box. */
  hint?: string
  onChange: (value: string) => void
}) {
  return (
    <div className="flex w-full flex-col gap-2">
      <Label htmlFor={id} className={labelText}>
        Your idea
      </Label>
      <Textarea
        id={id}
        rows={rows}
        value={value}
        placeholder="Describe it the way you would to a colleague."
        className="field-sizing-fixed resize-none rounded-[10px] border-input bg-card px-3 py-2.5 text-sm leading-[22px] shadow-sm dark:bg-card"
        onChange={(event) => onChange(event.target.value)}
      />
      {hint !== undefined ? <p className={captionText}>{hint}</p> : null}
    </div>
  )
}
