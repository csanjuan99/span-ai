"use client"

import { useEffect, useState } from "react"

import { applyMotion, motionReduced } from "@/lib/motion"
import { cn } from "@/lib/utils"
import { stages } from "@/lib/types"

const seenKey = "span:started"
const stepMs = 90
const holdMs = 250

/** Whether this tab already showed the start-up, or the person asked for less motion. */
function skipped(): boolean {
  try {
    if (motionReduced()) {
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

  // The saved motion choice reaches the whole page from here, the one component every load mounts.
  useEffect(() => applyMotion(), [])

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
      className="fixed inset-0 z-50 flex flex-col items-center justify-center gap-4 bg-background px-4 sm:px-10"
    >
      <ol
        className="flex w-full max-w-[1000px] flex-wrap justify-center gap-2 px-4 sm:grid sm:grid-cols-10 sm:px-0"
        aria-hidden="true"
      >
        {stages.map((name, index) => (
          <li
            key={name}
            className={cn(
              "flex h-8 min-w-0 items-center gap-1.5 rounded-full border px-2.5 text-xs font-medium whitespace-nowrap transition-colors",
              index < filled
                ? "border-transparent bg-accent text-primary"
                : "border-input bg-muted text-muted-foreground"
            )}
          >
            <b className="font-mono font-semibold">
              {index < filled ? "+" : "·"}
            </b>
            {name}
          </li>
        ))}
      </ol>
      <div className="text-center">
        <p className="text-sm font-semibold">Starting Span</p>
        <p className="text-xs text-muted-foreground">
          Opening your run folder.
        </p>
      </div>
    </div>
  )
}
