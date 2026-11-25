"use client"

import type { AgentState } from "@/lib/types"
import { agentStages, type AgentStage } from "@/lib/types"

import { valueOf } from "./draft"
import type { EditProps } from "./fields"
import { StageRow } from "./StageRow"

const hints: Record<AgentStage, string> = {
  frame: "reads the story, plans",
  build: "writes the code",
  inspect: "reviews the change",
}

/**
 * The CLI of each stage that calls a model. A stage left on the project agent follows it;
 * the other stages of a run run commands and have no agent to pick.
 */
export function AgentsFields({
  props,
  agents,
}: {
  props: EditProps
  agents: AgentState[]
}) {
  const projectAgent =
    props.config.agent === "auto" || props.config.agent === "any"
      ? "the first ready CLI"
      : props.config.agent
  return (
    <>
      {agentStages.map((stage) => {
        const value = valueOf(props.config, props.draft, `agents.${stage}`)
        return (
          <StageRow
            key={stage}
            stage={stage}
            hint={hints[stage]}
            label={`${stage} agent`}
            value={value}
            options={[
              { value: "", label: `Project agent (${projectAgent})` },
              ...agents
                .filter((row) => row.ready || row.id === value)
                .map((row) => ({
                  value: row.id,
                  label: row.ready ? row.label : `${row.label} (not ready)`,
                })),
            ]}
            changed={value !== ""}
            disabled={props.disabled}
            onChange={(next) => props.onEdit(`agents.${stage}`, next)}
          />
        )
      })}
      <p className="pt-2 text-xs text-muted-foreground">
        A stage a run was set to by hand keeps that choice. The other stages run
        commands, so they have no agent.
      </p>
    </>
  )
}
