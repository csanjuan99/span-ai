"use client"

import { useState } from "react"

import { AgentSelect } from "@/components/shape-entry/AgentModelSelects"
import { useInterviewAgents } from "@/components/shape-entry/useInterviewAgents"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Panel } from "@/components/shaping/session-ui"
import { setShapeAgent } from "@/lib/api"
import { reasonText } from "@/lib/guards"
import type { ShapeSession } from "@/lib/types"

/** Which agent and model the next turns use. Changing it runs nothing. */
export function SessionAgent({
  session,
  onSession,
}: {
  session: ShapeSession
  onSession: (next: ShapeSession) => void
}) {
  const [agent, setAgent] = useState(session.agent ?? "")
  const [model, setModel] = useState(session.model ?? "")
  const [note, setNote] = useState("")
  const [busy, setBusy] = useState(false)
  const agents = useInterviewAgents(session.workspace, agent)

  async function save() {
    setBusy(true)
    setNote("")
    try {
      onSession(
        await setShapeAgent(session.id, session.workspace, agent, model.trim())
      )
      setNote("Saved. The next turn uses it.")
    } catch (err: unknown) {
      setNote(reasonText(err, "the agent was not saved"))
    } finally {
      setBusy(false)
    }
  }

  return (
    <Panel aria-label="Interview agent" className="flex flex-col gap-3 p-4">
      <div className="text-sm font-semibold">Interview agent</div>
      <div className="flex flex-col gap-1.5">
        <div className="text-[13px] font-medium">Agent</div>
        <AgentSelect
          agents={agents}
          value={agent}
          wrapperClassName="w-full"
          className="w-full"
          onChange={setAgent}
        />
      </div>
      <div className="flex flex-col gap-1.5">
        <label htmlFor="session-model" className="text-[13px] font-medium">
          Model
        </label>
        <Input
          id="session-model"
          aria-label="Model"
          placeholder="Auto"
          value={model}
          onChange={(event) => setModel(event.target.value)}
        />
      </div>
      <Button
        type="button"
        variant="outline"
        className="h-10"
        disabled={busy}
        onClick={() => void save()}
      >
        Use for the next turns
      </Button>
      {note !== "" ? (
        <p role="status" className="text-xs text-muted-foreground">
          {note}
        </p>
      ) : null}
    </Panel>
  )
}
