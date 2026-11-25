"use client"

import { Eyebrow } from "@/components/shaping/session-ui"
import { TabStrip } from "@/components/shaping/TabStrip"
import { wrapRowWide } from "@/components/shaping/styles"

/** A two-way switch drawn as the design's tab strip, with its "View" caption. */
export function ViewTabs<T extends string>({
  label,
  value,
  options,
  onChange,
}: {
  label: string
  value: T
  options: { value: T; label: string }[]
  onChange: (next: T) => void
}) {
  return (
    <div className={wrapRowWide}>
      <Eyebrow>View</Eyebrow>
      <TabStrip
        label={label}
        value={value}
        options={options}
        onChange={onChange}
        className="w-full max-w-[520px] sm:w-auto sm:min-w-64"
      />
    </div>
  )
}
