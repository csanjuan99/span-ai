"use client"

import { SearchIcon } from "lucide-react"

/** The search box of a list: a labelled input in the design's rounded field. */
export function SearchField({
  label,
  placeholder,
  value,
  onChange,
}: {
  /** Read by screen readers; the box shows only the placeholder. */
  label: string
  placeholder: string
  value: string
  onChange: (value: string) => void
}) {
  return (
    <label className="flex h-11 min-w-0 flex-[1_1_280px] items-center gap-2 rounded-[10px] border border-input bg-card px-3 shadow-sm md:h-10">
      <span className="sr-only">{label}</span>
      <SearchIcon
        aria-hidden="true"
        className="size-4 shrink-0 text-muted-foreground"
      />
      <input
        type="search"
        value={value}
        onChange={(event) => onChange(event.target.value)}
        placeholder={placeholder}
        className="h-full min-w-0 flex-1 bg-transparent text-sm outline-none placeholder:text-muted-foreground"
      />
    </label>
  )
}
