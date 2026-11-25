"use client"

import { useState } from "react"

import { Cases } from "@/components/qa/Cases"
import { LogViewer } from "@/components/qa/LogViewer"
import { Overview } from "@/components/qa/Overview"
import { PillToggle } from "@/components/qa/PillToggle"
import { Transcript } from "@/components/qa/Transcript"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { designField } from "@/lib/field-class"
import { cn } from "@/lib/utils"
import { useRunResource } from "@/hooks/use-run-resource"
import { acceptQa, readQa } from "@/lib/qa"
import { reasonText } from "@/lib/guards"

type Pane = "overview" | "cases" | "log" | "transcript"

const panes: { value: Pane; label: string }[] = [
  { value: "overview", label: "Overview" },
  { value: "cases", label: "Cases" },
  { value: "log", label: "Log" },
  { value: "transcript", label: "Transcript" },
]

/** The QA tab: overview, cases, a gate's whole log, and what the agent printed. */
export function QaTab({ id }: { id: string }) {
  const proof = useRunResource(id, readQa)
  const [failure, setFailure] = useState("")
  const [pane, setPane] = useState<Pane>("overview")
  const [log, setLog] = useState("")
  const [reason, setReason] = useState("")
  const [version, setVersion] = useState(0)

  const view = proof.state.status === "ready" ? proof.state.value : null
  const unread = proof.state.status === "error" ? proof.state.message : ""
  const problem = failure || unread

  async function accept() {
    try {
      await acceptQa(id, reason)
      setReason("")
      setFailure("")
      setVersion((n) => n + 1)
      proof.reload()
    } catch (err: unknown) {
      setFailure(reasonText(err, "the risk could not be recorded"))
    }
  }

  const logged =
    log !== ""
      ? log
      : (view?.gates.find((gate) => gate.ran && gate.lines > 0)?.name ?? "")

  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-wrap gap-2" role="group" aria-label="QA view">
        {panes.map((item) => (
          <PillToggle
            key={item.value}
            on={pane === item.value}
            onClick={() => setPane(item.value)}
          >
            {item.label}
          </PillToggle>
        ))}
      </div>
      {problem !== "" ? (
        <p role="alert" className="text-sm text-destructive">
          {problem}
        </p>
      ) : null}
      {pane === "transcript" ? <Transcript id={id} /> : null}
      {pane === "cases" ? <Cases id={id} version={version} /> : null}
      {pane === "log" ? (
        logged === "" ? (
          <p className="text-sm text-pretty text-muted-foreground">
            This run has no log yet.
          </p>
        ) : (
          <LogViewer
            id={id}
            gate={logged}
            onClose={() => {
              setLog("")
              setPane("overview")
            }}
          />
        )
      ) : null}
      {pane === "overview" && view !== null ? (
        <>
          <Overview
            view={view}
            onOpenLog={(gate) => {
              setLog(gate)
              setPane("log")
            }}
          />
          {view.state === "ok" &&
          view.verdict === "fail" &&
          view.accepted === null ? (
            <div className="flex flex-col gap-2 rounded-[14px] border border-border bg-card p-4">
              <label htmlFor="qa-accept" className="text-sm font-medium">
                Go on without proof
              </label>
              <Input
                id="qa-accept"
                className={cn("h-auto", designField)}
                placeholder="Why is the risk acceptable?"
                value={reason}
                onChange={(event) => setReason(event.target.value)}
              />
              <Button
                type="button"
                variant="outline"
                size="sm"
                className="self-start"
                disabled={reason.trim() === ""}
                onClick={() => void accept()}
              >
                Record the risk
              </Button>
            </div>
          ) : null}
        </>
      ) : null}
    </div>
  )
}
