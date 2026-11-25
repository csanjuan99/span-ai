"use client"

import { useState } from "react"

import { useOptionalTheme } from "@/components/theme-provider"
import { readMotion, saveMotion, type MotionPref } from "@/lib/motion"
import { cn } from "@/lib/utils"

import { ChangedBadge } from "./ChangedBadge"
import { FieldRow } from "./FieldRow"
import { SelectField } from "./SelectField"

const themes = [
  { value: "system", label: "System" },
  { value: "light", label: "Light" },
  { value: "dark", label: "Dark" },
] as const

/** How Span looks and moves. Both choices are saved in this browser and apply at once. */
export function AppearanceFields() {
  const theme = useOptionalTheme()
  const [motion, setMotion] = useState<MotionPref>(readMotion)
  const [stored, setStored] = useState(true)

  return (
    <>
      {theme !== undefined ? (
        <FieldRow
          label="Theme"
          fallback="System"
          changed={theme.theme !== "system"}
          unsaved={false}
        >
          <div
            role="group"
            aria-label="Theme"
            className="flex w-fit gap-1 rounded-[14px] bg-muted p-1"
          >
            {themes.map((item) => (
              <button
                key={item.value}
                type="button"
                aria-pressed={theme.theme === item.value}
                onClick={() => theme.setTheme(item.value)}
                className={cn(
                  "h-9 rounded-[10px] px-4 text-[13px] font-medium text-muted-foreground",
                  "aria-pressed:bg-card aria-pressed:font-semibold aria-pressed:text-primary aria-pressed:shadow-sm"
                )}
              >
                {item.label}
              </button>
            ))}
          </div>
        </FieldRow>
      ) : null}
      <FieldRow
        label="Motion"
        fallback="Follow system"
        changed={motion !== "system"}
        unsaved={false}
      >
        <SelectField
          label="Motion"
          value={motion}
          options={[
            { value: "system", label: "Follow system" },
            { value: "reduced", label: "Reduce motion" },
          ]}
          onChange={(value) => {
            const next: MotionPref = value === "reduced" ? "reduced" : "system"
            setMotion(next)
            setStored(saveMotion(next))
          }}
        />
        <p className="text-xs text-muted-foreground">
          Reduced motion turns off the start-up animation, the skeleton shimmer
          and every other animation.
        </p>
        {!stored ? (
          <ChangedBadge>This browser would not keep it</ChangedBadge>
        ) : null}
      </FieldRow>
    </>
  )
}
