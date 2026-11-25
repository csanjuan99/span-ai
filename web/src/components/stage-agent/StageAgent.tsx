"use client"

import { useState } from "react"

import { setStageAgent } from "@/lib/api"
import {
  agentStages,
  agentForStage,
  defaultAgentStage,
} from "@/lib/stage-agents"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { cn } from "@/lib/utils"
import { touchArea } from "@/lib/button-class"

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
  const [agent, setAgent] = useState(() =>
    agentForStage(defaultAgentStage(current), choices)
  )
  const [note, setNote] = useState("")

  // The stage the run stopped at is the one a person came here to change. When the run moves
  // on, the form follows it; that is adjusted while rendering, not in an effect.
  const [seen, setSeen] = useState(current)
  if (seen !== current) {
    const next = defaultAgentStage(current)
    setSeen(current)
    setStage(next)
    setAgent(agentForStage(next, choices))
  }

  function pickStage(value: string) {
    setStage(value)
    setAgent(agentForStage(value, choices))
    setNote("")
  }

  // The choice is saved the moment it is made: one control, no separate save step.
  async function pick(value: string) {
    const chosen = value === "any" ? "" : value
    setAgent(chosen)
    setNote("")
    try {
      await setStageAgent(id, stage, chosen)
      setNote(
        `${stage} will use ${chosen === "" ? "the policy default" : chosen} the next time it starts.`
      )
    } catch (err: unknown) {
      setNote(err instanceof Error ? err.message : "the choice was not saved")
    }
  }

  return (
    <div className="flex items-center gap-3 border-t border-border px-4 py-3">
      <div className="flex min-w-0 flex-1 flex-col">
        <div className="flex items-center gap-1 text-sm font-medium whitespace-nowrap">
          <span>Agent for</span>
          <Select value={stage} onValueChange={pickStage}>
            <SelectTrigger
              aria-label="Stage to choose an agent for"
              size="sm"
              className={cn(
                touchArea,
                "h-6 gap-1 border-0 px-1 text-sm font-medium shadow-none"
              )}
            >
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
        </div>
        <p className="text-xs text-pretty text-muted-foreground">
          {note !== ""
            ? note
            : "Only frame, build and inspect call a model. Applies the next time that stage starts."}
        </p>
      </div>
      <Select
        value={agent === "" ? "any" : agent}
        onValueChange={(value) => void pick(value)}
      >
        <SelectTrigger
          aria-label={`Choose agent for ${stage}`}
          className="w-36 shrink-0 rounded-[10px] text-sm"
        >
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
    </div>
  )
}
