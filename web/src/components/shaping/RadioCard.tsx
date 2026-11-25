"use client"

import type { ReactNode } from "react"
import { cn } from "cn"
import { hintText } from "@/components/shaping/styles"

/** One choice of a radio group, drawn as a card. `roomy` is the larger card of a decision. */
export function RadioCard({
  name,
  checked,
  onSelect,
  title,
  roomy = false,
  children,
}: {
  name: string
  checked: boolean
  onSelect: () => void
  title: string
  roomy?: boolean
  children: ReactNode
}) {
  return (
    <label
      className={cn(
        "cursor-pointer border p-3",
        roomy
          ? "grid grid-cols-[20px_minmax(0,1fr)] gap-3 rounded-[14px] border-input bg-card p-4"
          : "flex gap-3 rounded-[10px] border-border",
        checked &&
          (roomy
            ? "border-primary bg-accent ring-1 ring-primary"
            : "border-primary bg-accent")
      )}
    >
      <input
        type="radio"
        name={name}
        checked={checked}
        onChange={onSelect}
        className={cn("mt-1", roomy && "size-4 accent-primary")}
      />
      <span className={cn("flex flex-col", roomy && "gap-0.5")}>
        <span
          className={cn("font-semibold", roomy ? "text-sm" : "text-[13px]")}
        >
          {title}
        </span>
        <span className={cn(hintText, roomy && "leading-5")}>{children}</span>
      </span>
    </label>
  )
}

/** A group of radio cards under a legend that only a screen reader reads. */
export function RadioCards<T extends string>({
  legend,
  name,
  value,
  options,
  onChange,
}: {
  legend: string
  name: string
  value: T
  options: readonly (readonly [T, string, string])[]
  onChange: (value: T) => void
}) {
  return (
    <fieldset className="flex flex-col gap-3">
      <legend className="sr-only">{legend}</legend>
      {options.map(([option, title, hint]) => (
        <RadioCard
          key={option}
          name={name}
          checked={value === option}
          onSelect={() => onChange(option)}
          title={title}
        >
          {hint}
        </RadioCard>
      ))}
    </fieldset>
  )
}
