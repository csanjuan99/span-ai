"use client"

import { VerdictPill } from "@/components/qa/Pill"
import { exitLabel, seconds, type QaView } from "@/lib/qa"

/** What the proof says, in the words the data allows: no pass is claimed that no gate showed. */
export function Overview({
  view,
  onOpenLog,
}: {
  view: QaView
  onOpenLog: (gate: string) => void
}) {
  return (
    <section aria-label="Proof" className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center gap-3">
        <VerdictPill verdict={view.verdict} />
        <h2 className="text-lg font-semibold text-pretty">{view.headline}</h2>
      </div>
      {view.counts !== null ? (
        <p className="text-sm text-muted-foreground tabular-nums">
          {view.counts.passed} passed · {view.counts.failed} failed ·{" "}
          {view.counts.skipped} skipped
          {view.counts.seconds > 0
            ? ` · ${view.counts.seconds.toFixed(1)} s`
            : ""}
        </p>
      ) : null}
      {view.evidenceNote !== "" ? (
        <p className="text-sm text-pretty text-muted-foreground">
          {view.evidenceNote}
        </p>
      ) : null}
      {view.reason !== "" ? (
        <p className="text-sm text-pretty">{view.reason}</p>
      ) : null}
      {view.next !== "" ? (
        <p className="text-sm text-pretty text-hold">{view.next}</p>
      ) : null}
      {view.accepted !== null ? (
        <p className="rounded-[10px] bg-accent-soft p-3 text-sm text-pretty">
          Accepted without proof: {view.accepted.reason}
        </p>
      ) : null}
      {view.gates.length > 0 ? (
        <div
          className="overflow-hidden rounded-[14px] border border-border bg-card"
          role="table"
          aria-label="Gates"
        >
          <div
            role="row"
            className="grid grid-cols-[88px_88px_56px_64px_minmax(0,1fr)] gap-3 bg-muted px-4 py-2 text-xs text-muted-foreground max-sm:hidden"
          >
            <span role="columnheader">Gate</span>
            <span role="columnheader">Verdict</span>
            <span role="columnheader">Exit</span>
            <span role="columnheader">Time</span>
            <span role="columnheader">Command</span>
          </div>
          {view.gates.map((gate) => (
            <div
              key={gate.name}
              role="row"
              className="grid grid-cols-[88px_88px_56px_64px_minmax(0,1fr)] items-center gap-3 border-t border-border px-4 py-2.5 text-sm max-sm:grid-cols-2"
            >
              <span role="cell" className="font-medium">
                {gate.name}
              </span>
              <span role="cell">
                <VerdictPill verdict={gate.verdict} />
              </span>
              <span role="cell" className="font-mono tabular-nums">
                {gate.ran ? gate.exitCode : "–"}
              </span>
              <span role="cell" className="tabular-nums">
                {seconds(gate.durationMs)}
              </span>
              <span
                role="cell"
                className="flex min-w-0 items-center gap-2 max-sm:col-span-2"
              >
                <span className="min-w-0 truncate font-mono text-xs text-muted-foreground">
                  {gate.ran ? gate.command : gate.detail}
                </span>
                {gate.ran && gate.lines > 0 ? (
                  <button
                    type="button"
                    onClick={() => onOpenLog(gate.name)}
                    className="shrink-0 text-xs text-primary underline-offset-4 hover:underline"
                  >
                    Open log
                  </button>
                ) : null}
              </span>
            </div>
          ))}
        </div>
      ) : null}
      <p className="text-xs text-muted-foreground">
        Terminal: <span className="font-mono">span qa show {view.run}</span>{" "}
        exits {view.exit} ({exitLabel(view.exit)}).
      </p>
    </section>
  )
}
