"use client"

import { useEffect, useState } from "react"

import { cn } from "@/lib/utils"
import { stages } from "@/lib/types"

const seenKey = "span:started"
const stepMs = 90
const holdMs = 250

/** Whether this tab already showed the start-up, or the person asked for less motion. */
function skipped(): boolean {
  try {
    if (window.matchMedia("(prefers-reduced-motion: reduce)").matches) {
      return true
    }
    return window.sessionStorage.getItem(seenKey) === "1"
  } catch {
    // Storage can be blocked; the animation then shows on every load, which is harmless.
    return false
  }
}

/** Marks this tab as started. A tab that cannot store it just shows the animation again next load. */
function remember(): boolean {
  try {
    window.sessionStorage.setItem(seenKey, "1")
    return true
  } catch {
    return false
  }
}

/**
 * The first thing a tab shows: the ten stages fill in one by one in about a second, then it
 * leaves. It shows once per tab and not at all with reduced motion, and never covers a page for
 * longer than that, whatever is loading underneath.
 */
export function StartUp() {
  const [open, setOpen] = useState(() => !skipped())
  const [filled, setFilled] = useState(0)

  useEffect(() => {
    if (!open) {
      return undefined
    }
    remember()
    const timer = window.setInterval(
      () => setFilled((count) => Math.min(count + 1, stages.length)),
      stepMs
    )
    return () => window.clearInterval(timer)
  }, [open])

  useEffect(() => {
    if (open && filled >= stages.length) {
      const done = window.setTimeout(() => setOpen(false), holdMs)
      return () => window.clearTimeout(done)
    }
    return undefined
  }, [open, filled])

  if (!open) {
    return null
  }
  return (
    <div
      role="status"
      aria-label="Starting Span"
      className="fixed inset-0 z-50 flex flex-col items-center justify-center gap-4 bg-background"
    >
      <ol
        className="flex max-w-xl flex-wrap justify-center gap-2 px-4 text-xs"
        aria-hidden="true"
      >
        {stages.map((name, index) => (
          <li
            key={name}
            className={cn(
              "rounded-md px-2 py-1 transition-colors",
              index < filled
                ? "bg-pass-soft text-pass"
                : "text-muted-foreground"
            )}
          >
            <b className="mr-1 font-mono">{index < filled ? "+" : "·"}</b>
            {name}
          </li>
        ))}
      </ol>
      <p className="text-sm text-muted-foreground">Starting Span</p>
    </div>
  )
}
