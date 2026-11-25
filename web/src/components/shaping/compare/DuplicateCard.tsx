"use client"

import { Panel, Pill } from "@/components/shaping/session-ui"
import { Button } from "@/components/ui/button"
import type { Duplicate } from "@/lib/shape-compare"
import { wrapRow } from "@/components/shaping/styles"

/** Two sessions that tell mostly the same story. Matching is by shared words, so the figure is an estimate. */
export function DuplicateCard({
  duplicate,
  busy,
  onKeepSeparate,
}: {
  duplicate: Duplicate
  busy: boolean
  onKeepSeparate: () => void
}) {
  return (
    <Panel
      aria-label={`${duplicate.aRef} and ${duplicate.bRef}`}
      className="flex flex-col gap-2 px-4 py-4 sm:px-5"
    >
      <div className={wrapRow}>
        <span className="text-sm font-semibold">
          {duplicate.aRef} and {duplicate.bRef}
        </span>
        <Pill tone={duplicate.dismissed ? "mut" : "hold"} small>
          Mostly the same story · estimate
        </Pill>
      </div>
      <p className="text-xs text-muted-foreground">
        {duplicate.aItems.length} items of {duplicate.aRef} are also in{" "}
        {duplicate.bRef}: {duplicate.aItems.join(", ")}.
      </p>
      {duplicate.dismissed ? (
        <p className="text-xs text-muted-foreground">
          You kept these separate.
        </p>
      ) : (
        <Button
          variant="outline"
          size="lg"
          className="w-full sm:w-fit"
          disabled={busy}
          onClick={onKeepSeparate}
        >
          Keep separate
        </Button>
      )}
    </Panel>
  )
}
