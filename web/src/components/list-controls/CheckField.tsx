"use client"

import type { ReactNode } from "react"

import { cn } from "@/lib/utils"

/** A checkbox with its label, as one row. The caller places and sizes it with the classes. */
export function CheckField({
  checked,
  onChange,
  label,
  ariaLabel,
  disabled = false,
  className,
  inputClassName = "size-4 accent-primary",
}: {
  checked: boolean
  onChange: (checked: boolean) => void
  label?: ReactNode
  ariaLabel?: string
  disabled?: boolean
  /** The classes of the row. */
  className?: string
  /** The classes of the box itself. */
  inputClassName?: string
}) {
  return (
    <label className={className}>
      <input
        type="checkbox"
        className={cn(inputClassName)}
        checked={checked}
        disabled={disabled}
        aria-label={ariaLabel}
        onChange={(event) => onChange(event.target.checked)}
      />
      {label}
    </label>
  )
}
