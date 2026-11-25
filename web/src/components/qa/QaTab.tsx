"use client"

import { useState } from "react"

import { Cases } from "@/components/qa/Cases"
import { LogViewer } from "@/components/qa/LogViewer"
import { Overview } from "@/components/qa/Overview"
import { PillToggle } from "@/components/qa/PillToggle"
import { Transcript } from "@/components/qa/Transcript"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { useRunResource } from "@/hooks/use-run-resource"
import { acceptQa, readQa } from "@/lib/qa"
import { reasonText } from "@/lib/guards"

type Pane = "proof" | "transcript"

/** The QA tab: the proof, its cases, a gate's whole log and what the agent printed. */
export function QaTab({ id }: { id: string }) {
  const proof = useRunResource(id, readQa)
  const [failure, setFailure] = useState("")
  const [pane, setPane] = useState<Pane>("proof")
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

  if (log !== "") {
    return <LogViewer id={id} gate={log} onClose={() => setLog("")} />
  }
  return (
    <div className="flex flex-col gap-6">
      <div className="flex gap-2" role="group" aria-label="QA view">
        {(["proof", "transcript"] as const).map((item) => (
          <PillToggle
            key={item}
            on={pane === item}
            onClick={() => setPane(item)}
            className="capitalize"
          >
            {item}
          </PillToggle>
        ))}
      </div>
      {problem !== "" ? (
        <p role="alert" className="text-sm text-destructive">
          {problem}
        </p>
      ) : null}
      {pane === "transcript" ? <Transcript id={id} /> : null}
      {pane === "proof" && view !== null ? (
        <>
          <Overview view={view} onOpenLog={setLog} />
          {view.state === "ok" &&
          view.verdict === "fail" &&
          view.accepted === null ? (
            <div className="flex flex-col gap-2 rounded-[14px] border border-border bg-card p-4">
              <label htmlFor="qa-accept" className="text-sm font-medium">
                Go on without proof
              </label>
              <Input
                id="qa-accept"
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
          {view.state === "ok" || view.cases > 0 ? (
            <Cases id={id} version={version} />
          ) : null}
        </>
      ) : null}
    </div>
  )
}
