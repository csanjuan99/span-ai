"use client"

import { Button } from "@/components/ui/button"

/** The one action a held run offers, with what it does stated right under it. */
export function ActionWithHint({
  label,
  hint,
  onClick,
}: {
  label: string
  hint: string
  onClick: () => void
}) {
  return (
    <div className="flex flex-col gap-2">
      <Button type="button" className="self-start" onClick={onClick}>
        {label}
      </Button>
      <p className="text-sm text-pretty text-muted-foreground">{hint}</p>
    </div>
  )
}
