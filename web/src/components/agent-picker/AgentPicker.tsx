import { useEffect, useState } from "react"

import { agentChoice, getJSON, saveAgent } from "@/lib/api"
import type { AgentChoice } from "@/lib/types"
import { Button } from "@/components/ui/button"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"

const empty: AgentChoice = { agents: [], selected: "any", chosen: "", exists: false, reason: "" }

/**
 * Which CLI writes this project. The page reads the same registry the engine uses,
 * so a row that cannot run says why instead of failing on the next stage.
 */
export function AgentPicker({ project }: { project: string }) {
  const [loaded, setLoaded] = useState<AgentChoice>(empty)
  const [picked, setPicked] = useState("")
  const [note, setNote] = useState("")
  const [busy, setBusy] = useState(false)

  // A different project is a different registry, so a pick made for the old one is dropped
  // while rendering, not in an effect that would render twice.
  const [seen, setSeen] = useState(project)
  if (seen !== project) {
    setSeen(project)
    setPicked("")
  }
  const blank = project.trim() === ""

  useEffect(() => {
    if (blank) {
      return
    }
    let live = true
    void getJSON(`/agent?path=${encodeURIComponent(project)}`)
      .then((body) => {
        if (live) {
          setLoaded(agentChoice(body))
        }
      })
      .catch((err: unknown) => {
        if (live) {
          setNote(err instanceof Error ? err.message : "the agents could not be read")
        }
      })
    return () => {
      live = false
    }
  }, [project, blank])

  // Nothing is shown for a blank project, even if an earlier one was loaded.
  const choice = blank ? empty : loaded
  const value = picked === "" ? choice.selected : picked
  const current = choice.agents.find((row) => row.id === value)
  const chosen = choice.agents.find((row) => row.id === choice.chosen)

  async function save() {
    setBusy(true)
    setNote("")
    try {
      const body = await saveAgent(project, value)
      setLoaded(agentChoice(body))
      setPicked("")
      setNote("Saved. The next run uses this CLI.")
    } catch (err: unknown) {
      setNote(err instanceof Error ? err.message : "the agent was not saved")
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="flex flex-col gap-2 border border-border p-3">
      <Label htmlFor="agent">Agent</Label>
      <p className="text-pretty text-sm text-muted-foreground">
        Cursor, Claude, Codex, or Pi. Span calls the CLI headless and needs that harness to already have Gentle AI.
        {chosen !== undefined ? ` Running now: ${chosen.label}.` : ""}
        {choice.chosen === "" ? " No CLI is ready on this machine." : ""}
      </p>
      <div className="flex flex-wrap items-end gap-2">
        <Select value={value} onValueChange={setPicked}>
          <SelectTrigger id="agent" className="w-56">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="any">Any ready CLI</SelectItem>
            {choice.agents.map((row) => (
              <SelectItem key={row.id} value={row.id} disabled={!row.ready}>
                {row.label}
                {row.ready ? "" : ` — ${row.reason ?? "not ready"}`}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Button type="button" variant="secondary" disabled={busy || !choice.exists} onClick={() => void save()}>
          Save agent
        </Button>
      </div>
      {value === "any" ? null : current !== undefined && !current.ready ? (
        <p className="text-pretty text-sm text-hold">{current.reason}</p>
      ) : null}
      {choice.reason !== "" ? <p className="text-pretty text-sm text-muted-foreground">{choice.reason}</p> : null}
      {note !== "" ? <p className="text-pretty text-sm">{note}</p> : null}
    </div>
  )
}
