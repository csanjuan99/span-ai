"use client"

import type { ComponentProps } from "react"

import { cn } from "@/lib/utils"

/** The quiet button of the tabs: open the whole diff, show it anyway, export the log. */
export function ActionButton({
  className,
  ...props
}: ComponentProps<"button">) {
  return (
    <button
      type="button"
      className={cn(
        "inline-flex h-9 items-center rounded-lg border border-input bg-card px-3 text-[13px] font-medium disabled:opacity-60 max-sm:h-11",
        className
      )}
      {...props}
    />
  )
}
