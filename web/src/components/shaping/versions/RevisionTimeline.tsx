"use client"

import { cn } from "@/lib/utils"

import { PillToggle } from "@/components/qa/PillToggle"
import { Panel, Pill } from "@/components/shaping/session-ui"
import { clockTime } from "@/lib/clock"
import { type Revision } from "@/lib/shape-versions"
import { cardBody, countText, headingText } from "@/components/shaping/styles"

/** The revisions of a session, each with a Before and an After pick. Every answer, split and accept is one. */
export function RevisionTimeline({
  revisions,
  before,
  after,
  onPick,
}: {
  revisions: Revision[]
  before: string
  after: string
  onPick: (role: "before" | "after", id: string) => void
}) {
  return (
    <Panel aria-label="Revisions" className={cardBody}>
      <div className="flex flex-wrap items-baseline gap-x-3 gap-y-1">
        <h2 className={headingText}>Revisions</h2>
        <span className="text-xs text-muted-foreground">
          Pick one as Before and one as After. Every answer, split and accept is
          a revision.
        </span>
      </div>
      <ol className="grid grid-cols-1 gap-2 sm:grid-cols-[repeat(auto-fit,minmax(11rem,1fr))] sm:gap-3">
        {revisions.map((rev) => {
          const picked = rev.id === before || rev.id === after
          return (
            <li
              key={rev.id}
              className={cn(
                "flex min-w-0 flex-col gap-1.5 rounded-[10px] border border-border bg-card p-3",
                picked && "border-primary ring-1 ring-primary"
              )}
            >
              <div className="flex flex-wrap items-center gap-1.5">
                <span className="font-mono text-[13px] leading-[22px] font-semibold text-primary">
                  {rev.id}
                </span>
                <span className={countText}>{clockTime(rev.at)}</span>
                {rev.id === before ? (
                  <Pill tone="acc" small>
                    Before
                  </Pill>
                ) : null}
                {rev.id === after ? (
                  <Pill tone="acc" small>
                    After
                  </Pill>
                ) : null}
                {rev.current ? (
                  <Pill tone="mut" small>
                    Current
                  </Pill>
                ) : null}
              </div>
              <div className="text-[13px] leading-5 font-medium text-pretty">
                {rev.label}
              </div>
              <div className="text-xs text-muted-foreground">{rev.actor}</div>
              <div className="mt-auto flex flex-wrap gap-1.5 pt-1">
                {(["before", "after"] as const).map((role) => (
                  <PillToggle
                    key={role}
                    on={(role === "before" ? before : after) === rev.id}
                    ariaLabel={`${role === "before" ? "Before" : "After"}: ${rev.id}`}
                    onClick={() => onPick(role, rev.id)}
                    className="h-11 sm:h-8"
                  >
                    {role === "before" ? "Before" : "After"}
                  </PillToggle>
                ))}
              </div>
            </li>
          )
        })}
      </ol>
    </Panel>
  )
}
