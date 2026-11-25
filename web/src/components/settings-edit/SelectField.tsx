"use client"

import { ChevronDownIcon } from "lucide-react"
import type { ChangeEvent } from "react"

import { cn } from "@/lib/utils"

/** One choice of a {@link SelectField}. */
export interface SelectOption {
  value: string
  label: string
}

/**
 * A native select drawn like the design's: monospace, fit to its text, a rounded border and
 * our own chevron. It stays a real select, so the keyboard and screen readers work as usual.
 */
export function SelectField({
  label,
  value,
  options,
  disabled = false,
  mono = false,
  onChange,
}: {
  label: string
  value: string
  options: SelectOption[]
  disabled?: boolean
  mono?: boolean
  onChange: (value: string) => void
}) {
  return (
    <span className="relative inline-flex w-fit max-w-full">
      <select
        aria-label={label}
        className={cn(
          "h-8 max-w-full appearance-none rounded-[10px] border border-input bg-card py-0 pr-8 pl-2.5 text-[13px] outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 disabled:opacity-50",
          mono && "font-mono text-xs"
        )}
        disabled={disabled}
        value={value}
        onChange={(event: ChangeEvent<HTMLSelectElement>) =>
          onChange(event.target.value)
        }
      >
        {options.map((option) => (
          <option key={option.value} value={option.value}>
            {option.label}
          </option>
        ))}
      </select>
      <ChevronDownIcon
        aria-hidden="true"
        className="pointer-events-none absolute top-1/2 right-2.5 size-3.5 -translate-y-1/2 text-muted-foreground"
      />
    </span>
  )
}
