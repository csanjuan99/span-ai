"use client"

import { CircleAlertIcon, ShieldCheckIcon } from "lucide-react"

import { Panel } from "@/components/shaping/session-ui"
import { Consequences } from "@/components/status-block/Consequences"
import { Button } from "@/components/ui/button"
import { touchButton } from "@/lib/button-class"
import { canAcceptRisk, type QaState, type QaView } from "@/lib/qa"
import { cn } from "@/lib/utils"

type HonestState = Exclude<QaState, "ok">

/** What each honest state says in its band, and what it does not show. */
const states: Record<
  HonestState,
  { band: string; notShown: string; tone: string }
> = {
  no_cases: {
    band: "No cases found",
    notShown: "A pass. Nothing was proven.",
    tone: "border-hold/40 bg-hold-soft text-hold",
  },
  never_ran: {
    band: "Gate never ran",
    notShown: "Any pass or fail. Span shows “not run”, not “passed”.",
    tone: "border-hold/40 bg-hold-soft text-hold",
  },
  unsealed: {
    band: "Sealed build missing",
    notShown: "A pass for this code.",
    tone: "border-fail/40 bg-fail-soft text-fail",
  },
}

/** What sits at the right of the band: where the reading came from. */
function bandNote(view: QaView, state: HonestState): string {
  if (state === "no_cases") {
    return `${view.gates[0]?.name ?? "tests"} · exit ${view.exitCode}`
  }
  return state === "never_ran" ? "no command recorded" : "not sealed"
}

/**
 * A reading Span refuses to call a pass: no cases, a gate that never ran, or a build that
 * was never sealed. It says what happened, what was kept and what is not shown.
 */
export function Honest({
  view,
  state,
  onAccept,
}: {
  view: QaView
  state: HonestState
  /** Moves to the form that records the risk. */
  onAccept: () => void
}) {
  const copy = states[state]
  return (
    <Panel aria-label="Proof" className="overflow-hidden">
      <div
        className={cn(
          "flex flex-wrap items-center justify-between gap-2 border-b px-4 py-2 text-xs font-semibold",
          copy.tone
        )}
      >
        <span className="flex items-center gap-2">
          <CircleAlertIcon className="size-4" aria-hidden="true" />
          {copy.band}
        </span>
        <span className="font-mono font-normal">{bandNote(view, state)}</span>
      </div>
      <div className="flex flex-col gap-3 p-4">
        <h2 className="text-lg font-semibold text-pretty">{view.headline}</h2>
        {view.reason !== "" ? (
          <p className="text-sm text-pretty">
            <strong className="font-semibold">What happened.</strong>{" "}
            {view.reason}
          </p>
        ) : null}
        <Consequences
          title="What is kept, and what to do"
          icon={ShieldCheckIcon}
          iconClassName="text-primary"
          rows={[
            { label: "Kept", body: view.kept },
            { label: "Not shown", body: copy.notShown },
            { label: "Next", body: view.next },
          ].filter((row) => row.body !== "")}
        />
        {canAcceptRisk(view) ? (
          <>
            <Button
              type="button"
              variant="outline"
              className={cn("w-full", touchButton)}
              onClick={onAccept}
            >
              Accept with no tests
            </Button>
            <p className="text-xs text-muted-foreground">
              Accepting records the risk with your name and the reason.
            </p>
          </>
        ) : null}
      </div>
    </Panel>
  )
}
