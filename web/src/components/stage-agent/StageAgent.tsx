import { useEffect, useState } from "react"

import { setStageAgent } from "@/lib/api"
import { agentStages, agentForStage, defaultAgentStage, usesAgent } from "@/lib/stage-agents"
import { Button } from "@/components/ui/button"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"

const agents = [
  { id: "cursor", label: "Cursor" },
  { id: "claude", label: "Claude" },
  { id: "codex", label: "Codex" },
  { id: "pi", label: "Pi" },
]

/**
 * Which CLI one stage of this run uses. The choice belongs to the run, so the same
 * project can try one model on build and another on inspect without editing span.yaml.
 */
export function StageAgent({
  id,
  current,
  choices,
}: {
  id: string
  current: string
  choices: Record<string, string> | undefined
}) {
  const [stage, setStage] = useState(() => defaultAgentStage(current))
  const [agent, setAgent] = useState(() => agentForStage(defaultAgentStage(current), choices))
  const [note, setNote] = useState("")
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    const next = defaultAgentStage(current)
    setStage(next)
    setAgent(agentForStage(next, choices))
    // The stage the run stopped at is the one a person came here to change.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [current])

  function pickStage(value: string) {
    setStage(value)
    setAgent(agentForStage(value, choices))
    setNote("")
  }

  async function save() {
    setBusy(true)
    setNote("")
    try {
      await setStageAgent(id, stage, agent)
      setNote(`${stage} will use ${agent === "" ? "the policy default" : agent} the next time it starts.`)
    } catch (err: unknown) {
      setNote(err instanceof Error ? err.message : "the choice was not saved")
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="flex flex-col gap-2 border border-border p-3">
      <Label htmlFor="stage-agent-stage">Agent for a stage of this run</Label>
      <p className="text-pretty text-sm text-muted-foreground">
        Only frame, build, and inspect call a model. The rest run commands. A change applies the next time that stage
        starts, so pause the run first when you want it before the current stage.
      </p>
      <div className="flex flex-wrap items-end gap-2">
        <Select value={stage} onValueChange={pickStage}>
          <SelectTrigger id="stage-agent-stage" className="w-36">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {agentStages.map((name) => (
              <SelectItem key={name} value={name}>
                {name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Select value={agent === "" ? "any" : agent} onValueChange={(value) => setAgent(value === "any" ? "" : value)}>
          <SelectTrigger aria-label="Agent for this stage" className="w-44">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="any">Policy default</SelectItem>
            {agents.map((row) => (
              <SelectItem key={row.id} value={row.id}>
                {row.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Button type="button" variant="secondary" disabled={busy} onClick={() => void save()}>
          Save for this run
        </Button>
      </div>
      {usesAgent(current) && agentForStage(current, choices) !== "" ? (
        <p className="text-pretty text-sm text-muted-foreground">
          This run already sets {current} to {agentForStage(current, choices)}.
        </p>
      ) : null}
      {note !== "" ? <p className="text-pretty text-sm">{note}</p> : null}
    </div>
  )
}
