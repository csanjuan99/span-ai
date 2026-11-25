import type { ReactNode } from "react"

import { mutedLabelText } from "@/components/shaping/styles"
import { cn } from "@/lib/utils"

/** One figure of a strip: what it counts, how much, and a caption under it. */
export function FigureCell({
  label,
  value,
  caption,
  className,
  valueClassName = "text-lg",
}: {
  label: string
  value: ReactNode
  caption: ReactNode
  /** The borders and padding that place the cell in its strip. */
  className?: string
  valueClassName?: string
}) {
  return (
    <div className={cn("flex min-w-0 flex-col gap-1", className)}>
      <div className={mutedLabelText}>{label}</div>
      <div className={cn("font-semibold tabular-nums", valueClassName)}>
        {value}
      </div>
      <div className="text-xs text-muted-foreground">{caption}</div>
    </div>
  )
}
