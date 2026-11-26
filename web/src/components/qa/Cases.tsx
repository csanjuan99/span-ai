"use client"

import { RefreshCwIcon } from "lucide-react"
import { useEffect, useState } from "react"

import { Switch } from "@/components/list-controls/Switch"
import { CaseTable } from "@/components/qa/CaseTable"
import { PillToggle } from "@/components/qa/PillToggle"
import { Panel } from "@/components/shaping/session-ui"
import { Consequences } from "@/components/status-block/Consequences"
import { Button } from "@/components/ui/button"
import { touchButton } from "@/lib/button-class"
import { reasonText } from "@/lib/guards"
import {
  groupByFile,
  readQaCases,
  rerunQa,
  type QaCase,
  type QaCasesView,
} from "@/lib/qa"
import { cn } from "@/lib/utils"

const filters = [
  { value: "", label: "All" },
  { value: "failed", label: "Failed" },
  { value: "passed", label: "Passed" },
  { value: "skipped", label: "Skipped" },
  { value: "new", label: "New" },
]

/** The count a filter chip carries; null for the filters the server does not total. */
function counted(filter: string, view: QaCasesView): number | null {
  const counts: Record<string, number> = {
    "": view.total,
    failed: view.failed,
    new: view.added,
  }
  return counts[filter] ?? null
}

/** The cases of the proof, failures first, grouped by file unless the person turns that off. */
export function Cases({ id, version }: { id: string; version: number }) {
  const [status, setStatus] = useState("")
  const [grouped, setGrouped] = useState(true)
  const [view, setView] = useState<QaCasesView | null>(null)
  const [problem, setProblem] = useState("")
  const [open, setOpen] = useState("")
  const [busy, setBusy] = useState(false)
  const [tick, setTick] = useState(0)

  useEffect(() => {
    let live = true
    readQaCases(id, status, grouped)
      .then((body) => {
        if (live) {
          setView(body)
          setProblem("")
        }
      })
      .catch((err: unknown) => {
        if (live) {
          setProblem(reasonText(err, "the cases could not be read"))
        }
      })
    return () => {
      live = false
    }
  }, [id, status, grouped, tick, version])

  async function rerun(names: string[]) {
    setBusy(true)
    setProblem("")
    try {
      await rerunQa(id, names)
      setTick((n) => n + 1)
    } catch (err: unknown) {
      setProblem(reasonText(err, "the cases could not be run again"))
    } finally {
      setBusy(false)
    }
  }

  if (view !== null && !view.known) {
    return (
      <p className="text-sm text-pretty text-muted-foreground">{view.note}</p>
    )
  }
  const failing = (view?.cases ?? [])
    .filter((item) => item.result === "fail")
    .map((item) => item.name)
  const table = (cases: QaCase[]) => (
    <CaseTable
      cases={cases}
      open={open}
      onToggle={(name) => setOpen(open === name ? "" : name)}
      busy={busy}
      onRerun={(name) => void rerun([name])}
    />
  )
  return (
    <section aria-label="Cases" className="flex flex-col gap-3">
      <div className="flex flex-wrap items-center gap-2">
        {filters.map((item) => (
          <PillToggle
            key={item.value}
            on={status === item.value}
            onClick={() => setStatus(item.value)}
          >
            {item.label}
            {view !== null && counted(item.value, view) !== null ? (
              <span className="ml-1 font-mono tabular-nums">
                {counted(item.value, view)}
              </span>
            ) : null}
          </PillToggle>
        ))}
        <span className="ml-auto flex min-h-11 items-center gap-2 text-sm md:min-h-0">
          <Switch label="Group by file" on={grouped} onChange={setGrouped} />
          <span aria-hidden="true">Group by file</span>
        </span>
      </div>
      <div
        className={cn(
          "grid gap-4",
          failing.length > 0 &&
            "lg:grid-cols-[minmax(0,1fr)_320px] lg:items-start"
        )}
      >
        <div className="flex min-w-0 flex-col gap-3">
          {problem !== "" ? (
            <p role="alert" className="text-sm text-destructive">
              {problem}
            </p>
          ) : null}
          {view !== null && view.cases.length === 0 ? (
            <p className="text-sm text-muted-foreground">
              No case matches this filter.
            </p>
          ) : null}
          {view !== null ? (
            <div className="flex flex-wrap items-baseline gap-2">
              <h2 className="text-base font-semibold">Acceptance cases</h2>
              <p className="text-xs text-muted-foreground">
                {grouped
                  ? "Failures first within each file."
                  : "Failures first."}{" "}
                {view.total} cases, {view.passed} passed, {view.failed} failed.
              </p>
            </div>
          ) : null}
          {grouped
            ? groupByFile(view?.cases ?? []).map((group) => (
                <div key={group.path} className="flex min-w-0 flex-col gap-1.5">
                  <div className="flex flex-wrap items-baseline gap-2 text-sm">
                    <span className="font-mono font-semibold">
                      {group.path === "" ? "Not placed in a file" : group.path}
                    </span>
                    <span className="text-xs text-muted-foreground tabular-nums">
                      {group.passed} passed · {group.failed} failed
                    </span>
                  </div>
                  {table(group.cases)}
                </div>
              ))
            : view !== null && view.cases.length > 0
              ? table(view.cases)
              : null}
        </div>
        {failing.length > 0 ? (
          <Panel className="flex flex-col gap-3 p-4 lg:sticky lg:top-4">
            <Consequences
              title="Rerun"
              icon={RefreshCwIcon}
              iconClassName="text-primary"
              rows={[
                {
                  label: "Does",
                  body: `Runs the ${failing.length} failing cases again on the branch checkout.`,
                },
                {
                  label: "Changes",
                  body: "Nothing in the code. The result is added to this run.",
                },
                {
                  label: "Kept",
                  body: "The first result stays in the history.",
                },
                {
                  label: "Next",
                  body: "If they pass, resume verify. If not, the output shows here.",
                },
              ]}
            />
            <Button
              type="button"
              className={cn("w-full", touchButton)}
              disabled={busy}
              onClick={() => void rerun(failing)}
            >
              <RefreshCwIcon aria-hidden="true" />
              {busy ? "Running…" : `Rerun ${failing.length} failed cases`}
            </Button>
          </Panel>
        ) : null}
      </div>
      <p className="text-xs text-muted-foreground">
        Terminal: <span className="font-mono">span qa cases {id} --failed</span>
      </p>
    </section>
  )
}
