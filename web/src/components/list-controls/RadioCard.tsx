"use client"

import type { ReactNode } from "react"
import { cn } from "@/lib/utils"
import { hintText } from "@/components/shaping/styles"

/** What a chosen card looks like, shared by every radio choice drawn as a card. */
export const checkedCard = "border-primary bg-accent ring-1 ring-primary"

/** The radio input of a roomy card: large enough to hit, in the accent colour. */
export const roomyRadio = "mt-1 size-4 accent-primary"

/** One choice of a radio group, drawn as a card. `roomy` is the larger card of a decision. */
export function RadioCard({
  name,
  checked,
  onSelect,
  title,
  value,
  roomy = false,
  children,
}: {
  name: string
  checked: boolean
  onSelect: () => void
  title: ReactNode
  value?: string
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
        checked && (roomy ? checkedCard : "border-primary bg-accent")
      )}
    >
      <input
        type="radio"
        name={name}
        value={value}
        checked={checked}
        onChange={onSelect}
        className={cn("mt-1", roomy && roomyRadio)}
      />
      <span className={cn("flex flex-col", roomy && "min-w-0 gap-1.5")}>
        <span
          className={cn(
            "font-semibold",
            roomy ? "flex flex-wrap items-center gap-2 text-sm" : "text-[13px]"
          )}
        >
          {title}
        </span>
        <span
          className={cn(hintText, roomy && "flex flex-col gap-1.5 leading-5")}
        >
          {children}
        </span>
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
