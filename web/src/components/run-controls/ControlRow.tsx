"use client"

import { ChevronRightIcon } from "lucide-react"
import type { ComponentType, ReactNode } from "react"

/** One compact row of the run controls: an icon, what it does in one line, a caption and a chevron. */
export function ControlRow({
  icon: Icon,
  title,
  caption,
  onClick,
}: {
  icon: ComponentType<{ className?: string; "aria-hidden"?: boolean }>
  title: string
  caption: string
  onClick: () => void
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="flex min-h-11 w-full items-center gap-3 border-t border-border px-4 py-2.5 text-left hover:bg-muted/50"
    >
      <Icon
        className="size-4 shrink-0 text-muted-foreground"
        aria-hidden={true}
      />
      <span className="flex min-w-0 flex-1 flex-col">
        <span className="text-sm font-medium">{title}</span>
        <span className="text-xs text-muted-foreground">{caption}</span>
      </span>
      <ChevronRightIcon
        className="size-4 shrink-0 text-muted-foreground"
        aria-hidden={true}
      />
    </button>
  )
}

/** The expanded part of a control: the form a row opens, under the same divider. */
export function ControlBlock({ children }: { children: ReactNode }) {
  return (
    <div className="flex flex-col gap-2 border-t border-border p-4">
      {children}
    </div>
  )
}
