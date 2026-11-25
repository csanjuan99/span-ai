"use client"

import { useState } from "react"

import { Pill } from "@/components/onboarding/Pill"
import { dryLook } from "@/lib/onboarding"
import { cn } from "@/lib/utils"

/** What a stage touches. Anything past the run's own branch is worth a second look. */
const quiet = ["nothing", "run branch"]

/** What each stage would do with the current settings. It reads nothing and writes nothing. */
export function DryLook() {
  const [on, setOn] = useState(true)
  return (
    <section
      aria-label="Dry look"
      className="overflow-hidden rounded-[14px] border border-border bg-card shadow-sm"
    >
      <div className="flex flex-wrap items-center gap-2 px-4 py-4 sm:px-5">
        <h2 className="text-lg font-semibold">Dry look</h2>
        <Pill tone="pass">Nothing has run</Pill>
        <button
          type="button"
          aria-pressed={on}
          onClick={() => setOn((now) => !now)}
          className={cn(
            "ml-auto h-7 rounded-full border border-input px-3 text-xs font-medium text-muted-foreground",
            "aria-pressed:border-transparent aria-pressed:bg-accent aria-pressed:text-primary"
          )}
        >
          {on ? "Dry look on" : "Dry look off"}
        </button>
      </div>
      {on ? (
        <div className="px-4 pb-2 sm:px-5">
          <div className="hidden grid-cols-[28px_120px_minmax(0,1fr)_110px_100px] gap-3 pb-2 text-xs font-medium text-muted-foreground sm:grid">
            <span />
            <span>Stage</span>
            <span>What the agent does</span>
            <span>Touches</span>
            <span>Asks you</span>
          </div>
          {dryLook.map((row) => (
            <div
              key={row.stage}
              className="grid grid-cols-[28px_minmax(0,1fr)] gap-x-3 gap-y-1 border-t border-border py-2.5 sm:grid-cols-[28px_120px_minmax(0,1fr)_110px_100px] [&>*:nth-child(n+3)]:col-start-2 sm:[&>*:nth-child(n+3)]:col-start-auto"
            >
              <span
                aria-hidden="true"
                className="font-mono font-semibold text-primary"
              >
                ·
              </span>
              <span className="text-sm font-semibold">{row.stage}</span>
              <span className="text-[13px]">{row.does}</span>
              <span>
                <Pill tone={quiet.includes(row.touches) ? "mut" : "hold"}>
                  {row.touches}
                </Pill>
              </span>
              <span>
                {row.asks === "never" ? (
                  <span className="text-xs text-muted-foreground">never</span>
                ) : (
                  <Pill tone="hold">{row.asks}</Pill>
                )}
              </span>
            </div>
          ))}
        </div>
      ) : null}
      <p className="px-4 pb-4 text-xs text-muted-foreground sm:px-5">
        Dry look reads nothing and writes nothing. It only shows what each stage
        is set to do with your current settings.
      </p>
    </section>
  )
}
