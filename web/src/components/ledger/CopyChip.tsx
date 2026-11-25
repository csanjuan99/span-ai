"use client"

import { CheckIcon } from "lucide-react"
import { useEffect, useRef, useState, type ReactNode } from "react"

import { cn } from "@/lib/utils"

interface Props {
  /** What is copied, named for a screen reader: "commit", "parent digest". */
  label: string
  /** The full value that goes to the clipboard. */
  value: string
  /** What the chip shows, usually the short form of the value. */
  children: ReactNode
  className?: string
}

/** A small button that copies a full hash and says so for a moment. */
export function CopyChip({ label, value, children, className }: Props) {
  const [copied, setCopied] = useState(false)
  const timer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined)

  useEffect(() => () => clearTimeout(timer.current), [])

  async function copy() {
    try {
      await navigator.clipboard.writeText(value)
      setCopied(true)
      clearTimeout(timer.current)
      timer.current = setTimeout(() => setCopied(false), 1400)
    } catch {
      // The clipboard can be denied; the full value stays in the title.
      setCopied(false)
    }
  }

  return (
    <button
      type="button"
      title={value}
      aria-label={`Copy ${label} ${value}`}
      onClick={() => void copy()}
      className={cn(
        "relative inline-flex items-center gap-1.5 rounded-md after:absolute after:-inset-x-1 after:-inset-y-3 sm:after:hidden border border-border px-1.5 py-0.5 font-mono text-xs tabular-nums text-muted-foreground transition-colors hover:border-primary/50 hover:text-foreground focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none",
        className,
      )}
    >
      {children}
      <span aria-live="polite" className="inline-flex w-3 justify-center">
        {copied ? <CheckIcon aria-label="copied" className="size-3 text-pass" /> : null}
      </span>
    </button>
  )
}
