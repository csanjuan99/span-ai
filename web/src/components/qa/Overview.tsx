"use client"

import { FlaskConicalIcon, FileTextIcon } from "lucide-react"

import { VerdictPill } from "@/components/qa/Pill"
import { Pill } from "@/components/onboarding/Pill"
import { Panel } from "@/components/shaping/session-ui"
import { Consequences } from "@/components/status-block/Consequences"
import { Button } from "@/components/ui/button"
import { touchButton } from "@/lib/button-class"
import {
  exitLabel,
  firstLoggedGate,
  hasLog,
  seconds,
  type QaView,
} from "@/lib/qa"
import { cn } from "@/lib/utils"

const gateColumns = "grid-cols-[88px_112px_72px_64px_minmax(0,1fr)]"

function exitText(ran: boolean, code: number): string {
  return ran ? `exit ${code}` : "not run"
}

function Count({
  label,
  mark,
  value,
  tone,
  note,
}: {
  label: string
  mark: string
  value: number
  tone: string
  note: string
}) {
  return (
    <div className="flex flex-col gap-1 p-4">
      <span className="text-xs text-muted-foreground">{label}</span>
      <span className={cn("text-xl font-semibold tabular-nums", tone)}>
        <span aria-hidden="true">{mark}</span> {value}
      </span>
      <span className="text-xs text-muted-foreground">{note}</span>
    </div>
  )
}

/** What the page offers beside the verdict: what each way on shows, changes and keeps. */
function nextRows(view: QaView) {
  const counts = view.counts
  if (view.verdict === "fail") {
    return {
      title: "Next",
      rows: [
        {
          label: "Shows",
          body: `The ${counts?.failed ?? "failing"} failing cases with their output.`,
        },
        { label: "Changes", body: "Nothing." },
        { label: "Kept", body: view.kept },
        { label: "Fix", body: view.next },
      ],
    }
  }
  return {
    title: "What you can do",
    rows: [
      {
        label: "Look",
        body: `Which ${counts?.skipped ?? ""} were skipped and why.`,
      },
      {
        label: "Accept",
        body: "Record the risk and continue, with your name on it.",
      },
      { label: "Kept", body: view.kept },
    ],
  }
}

/** The proof of the run in the words the data allows: no pass is claimed that no gate showed. */
export function Overview({
  view,
  onOpenLog,
  onOpenCases,
  onAccept,
}: {
  view: QaView
  onOpenLog: (gate: string) => void
  /** Goes to the case list. */
  onOpenCases: () => void
  /** Moves to the form that records the risk. */
  onAccept: () => void
}) {
  const counts = view.counts
  const logged = firstLoggedGate(view.gates)
  const asks = view.verdict === "fail" || view.verdict === "held"
  const next = nextRows(view)
  return (
    <section
      aria-label="Proof"
      className={cn(
        "grid gap-4",
        asks && "lg:grid-cols-[minmax(0,1fr)_320px] lg:items-start"
      )}
    >
      <div className="flex min-w-0 flex-col gap-4">
        <Panel className="flex flex-col gap-3 p-4">
          <div className="flex flex-wrap items-center gap-3">
            <VerdictPill verdict={view.verdict} />
            <h2 className="text-lg font-semibold text-pretty">
              {view.headline}
            </h2>
          </div>
          {view.command !== "" ? (
            <>
              <p className="text-[13px] text-muted-foreground">
                The command that produced it
              </p>
              <div className="flex items-center justify-between gap-4 rounded-[10px] border border-border bg-surface-code px-3.5 py-2 font-mono text-[13px] leading-5">
                <span className="min-w-0 truncate">
                  <span className="text-code-keyword" aria-hidden="true">
                    $
                  </span>{" "}
                  {view.command}
                </span>
                <span
                  className={cn(
                    "shrink-0",
                    view.exitCode === 0 ? "text-pass" : "text-fail"
                  )}
                >
                  exit {view.exitCode}
                </span>
              </div>
              <p className="flex flex-wrap items-center gap-2 text-[13px] text-muted-foreground">
                <span>Evidence</span>
                <Pill tone={view.verdict === "held" ? "hold" : "pass"}>
                  Run by Span
                </Pill>
                {view.evidenceNote}
              </p>
            </>
          ) : null}
          {view.reason !== "" ? (
            <p className="text-sm text-pretty">{view.reason}</p>
          ) : null}
          {view.accepted !== null ? (
            <p className="rounded-[10px] bg-accent-soft p-3 text-sm text-pretty">
              Accepted without proof: {view.accepted.reason}
            </p>
          ) : null}
        </Panel>
        {counts !== null ? (
          <Panel className="grid grid-cols-3 divide-x divide-border">
            <Count
              label="Passed"
              mark="+"
              value={counts.passed}
              tone="text-pass"
              note="cases"
            />
            <Count
              label="Failed"
              mark="×"
              value={counts.failed}
              tone="text-fail"
              note="cases"
            />
            <Count
              label="Skipped"
              mark="·"
              value={counts.skipped}
              tone="text-hold"
              note={
                counts.seconds > 0
                  ? `cases · took ${counts.seconds.toFixed(1)} s`
                  : "cases"
              }
            />
          </Panel>
        ) : null}
        {view.gates.length > 0 ? (
          <Panel className="overflow-hidden">
            <div className="flex flex-wrap items-baseline gap-2 px-4 py-3">
              <h2 className="text-lg leading-[26px] font-semibold">Gates</h2>
              <span className="text-xs leading-4 text-muted-foreground">
                In the order they ran
              </span>
            </div>
            <div role="table" aria-label="Gates">
              <div
                role="row"
                className={cn(
                  gateColumns,
                  "grid gap-3 bg-muted px-4 py-2 text-xs text-muted-foreground max-sm:hidden"
                )}
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
                  className={cn(
                    gateColumns,
                    "grid items-center gap-3 border-t border-border px-4 py-2.5 text-sm max-sm:grid-cols-2"
                  )}
                >
                  <span role="cell" className="font-medium">
                    {gate.name}
                  </span>
                  <span role="cell">
                    <VerdictPill verdict={gate.verdict} />
                  </span>
                  <span
                    role="cell"
                    className={cn(
                      "font-mono text-xs tabular-nums",
                      gate.ran && gate.exitCode !== 0 && "text-fail"
                    )}
                  >
                    {exitText(gate.ran, gate.exitCode)}
                  </span>
                  <span role="cell" className="text-xs tabular-nums">
                    {seconds(gate.durationMs)}
                  </span>
                  <span
                    role="cell"
                    className="flex min-w-0 items-center gap-2 max-sm:col-span-2"
                  >
                    <span className="min-w-0 truncate font-mono text-xs text-muted-foreground">
                      {gate.ran ? gate.command : gate.detail}
                    </span>
                    {hasLog(gate) ? (
                      <button
                        type="button"
                        onClick={() => onOpenLog(gate.name)}
                        className="shrink-0 text-xs text-primary underline-offset-4 hover:underline max-sm:min-h-11"
                      >
                        Open log
                      </button>
                    ) : null}
                  </span>
                </div>
              ))}
            </div>
          </Panel>
        ) : null}
        <p className="text-xs text-muted-foreground">
          Terminal: <span className="font-mono">span qa show {view.run}</span>{" "}
          exits {view.exit} ({exitLabel(view.exit)}).
        </p>
      </div>
      {asks ? (
        <Panel className="flex flex-col gap-3 p-4">
          <Consequences
            title={next.title}
            icon={FlaskConicalIcon}
            iconClassName="text-primary"
            rows={next.rows.filter((row) => row.body !== "")}
          />
          <Button
            type="button"
            className={cn("w-full", touchButton)}
            onClick={onOpenCases}
          >
            <FlaskConicalIcon aria-hidden="true" />
            {view.verdict === "fail"
              ? `See the ${counts?.failed ?? ""} failing cases`
              : `See the ${counts?.skipped ?? ""} skipped cases`}
          </Button>
          {view.verdict === "held" && view.accepted === null ? (
            <Button
              type="button"
              variant="outline"
              className={cn("w-full", touchButton)}
              onClick={onAccept}
            >
              Accept with {counts?.skipped ?? ""} skipped
            </Button>
          ) : null}
          {logged !== undefined ? (
            <Button
              type="button"
              variant="ghost"
              className={cn("w-full", touchButton)}
              onClick={() => onOpenLog(logged.name)}
            >
              <FileTextIcon aria-hidden="true" />
              Open full log
            </Button>
          ) : null}
        </Panel>
      ) : null}
    </section>
  )
}
