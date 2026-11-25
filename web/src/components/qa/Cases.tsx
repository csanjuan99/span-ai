"use client"

import { useEffect, useState } from "react"

import { VerdictPill } from "@/components/qa/Pill"
import { PillToggle } from "@/components/qa/PillToggle"
import { Button } from "@/components/ui/button"
import { reasonText } from "@/lib/guards"
import {
  caseTime,
  groupByFile,
  readQaCases,
  rerunQa,
  type QaCase,
  type QaCasesView,
} from "@/lib/qa"

const filters = [
  { value: "", label: "All" },
  { value: "failed", label: "Failed" },
  { value: "passed", label: "Passed" },
  { value: "skipped", label: "Skipped" },
  { value: "new", label: "New" },
]

function caseVerdict(result: QaCase["result"]) {
  return result === "pass" ? "pass" : result === "fail" ? "fail" : "held"
}

/** The cases of the proof grouped by file, failures first, each one opening to its own output. */
export function Cases({ id, version }: { id: string; version: number }) {
  const [status, setStatus] = useState("")
  const [view, setView] = useState<QaCasesView | null>(null)
  const [problem, setProblem] = useState("")
  const [open, setOpen] = useState("")
  const [busy, setBusy] = useState(false)
  const [tick, setTick] = useState(0)

  useEffect(() => {
    let live = true
    readQaCases(id, status)
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
  }, [id, status, tick, version])

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
          </PillToggle>
        ))}
        {failing.length > 0 ? (
          <Button
            type="button"
            size="sm"
            className="ml-auto"
            disabled={busy}
            onClick={() => void rerun(failing)}
          >
            {busy ? "Running…" : "Rerun failed cases"}
          </Button>
        ) : null}
      </div>
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
      {groupByFile(view?.cases ?? []).map((group) => (
        <div
          key={group.path}
          className="overflow-hidden rounded-[14px] border border-border bg-card"
        >
          <div className="flex flex-wrap items-center gap-2 bg-muted px-4 py-2 text-sm">
            <span className="font-mono">
              {group.path === "" ? "Not placed in a file" : group.path}
            </span>
            <span className="ml-auto text-xs text-muted-foreground tabular-nums">
              {group.passed} passed · {group.failed} failed
            </span>
          </div>
          {group.cases.map((item) => (
            <div key={item.name} className="border-t border-border">
              <button
                type="button"
                aria-expanded={open === item.name}
                onClick={() => setOpen(open === item.name ? "" : item.name)}
                className="grid min-h-11 w-full grid-cols-[72px_minmax(0,1fr)_56px] items-center gap-3 px-4 py-2 text-left text-sm"
              >
                <VerdictPill verdict={caseVerdict(item.result)}>
                  {item.result}
                </VerdictPill>
                <span className="min-w-0 truncate font-mono text-[13px]">
                  {item.name}
                  {item.isNew ? (
                    <span className="ml-2 rounded-md border border-dashed border-border px-1.5 text-[11px] text-muted-foreground">
                      new
                    </span>
                  ) : null}
                </span>
                <span className="text-right text-muted-foreground tabular-nums">
                  {caseTime(item.elapsedMs)}
                </span>
              </button>
              {open === item.name ? (
                <div className="flex flex-col gap-2 bg-muted px-4 py-3 text-sm">
                  {item.message !== "" ? (
                    <p className="text-pretty">{item.message}</p>
                  ) : null}
                  {item.line > 0 ? (
                    <p className="font-mono text-xs text-muted-foreground">
                      {item.path}:{item.line}
                    </p>
                  ) : null}
                  {item.output.length > 0 ? (
                    <pre className="overflow-x-auto rounded-[10px] border border-border bg-surface-code py-2 font-mono text-xs leading-5">
                      {item.output.map((line, index) => (
                        <div key={index} className="px-3.5 whitespace-pre">
                          {line}
                        </div>
                      ))}
                    </pre>
                  ) : null}
                  <Button
                    type="button"
                    size="sm"
                    variant="outline"
                    className="self-start"
                    disabled={busy}
                    onClick={() => void rerun([item.name])}
                  >
                    Rerun this case
                  </Button>
                </div>
              ) : null}
            </div>
          ))}
        </div>
      ))}
      <p className="text-xs text-muted-foreground">
        Terminal: <span className="font-mono">span qa cases {id} --failed</span>
      </p>
    </section>
  )
}
