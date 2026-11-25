"use client"

import { CheckIcon, RotateCwIcon } from "lucide-react"

import { checkedCard, roomyRadio } from "@/components/list-controls/RadioCard"
import { CheckLine } from "@/components/onboarding/CheckLine"
import { FixBox } from "@/components/onboarding/FixBox"
import { Panel } from "@/components/onboarding/Frame"
import { Pill } from "@/components/onboarding/Pill"
import { Button } from "@/components/ui/button"
import type { AgentState } from "@/lib/types"
import { cn } from "@/lib/utils"

function badgeOf(agent: AgentState) {
  if (agent.ready) {
    return (
      <Pill tone="pass">
        <CheckIcon aria-hidden="true" className="size-3" />
        Installed · ready
      </Pill>
    )
  }
  if (!agent.installed) {
    return <Pill tone="fail">Not installed</Pill>
  }
  return <Pill tone="hold">Installed · needs Gentle AI</Pill>
}

function Problem({ agent }: { agent: AgentState }) {
  if (agent.ready) {
    return (
      <span className="text-xs text-muted-foreground">
        Found on your PATH as <span className="font-mono">{agent.binary}</span>.
      </span>
    )
  }
  if (!agent.installed) {
    return (
      <ul>
        <CheckLine
          tone="fail"
          fix={
            agent.install !== undefined && agent.install !== "" ? (
              <FixBox command={agent.install}>
                Install it, then check again.
              </FixBox>
            ) : null
          }
        >
          <b className="text-fail">Not found.</b> No{" "}
          <span className="font-mono text-xs">{agent.binary}</span> command on
          your PATH.
        </CheckLine>
      </ul>
    )
  }
  return (
    <ul>
      <CheckLine
        tone="hold"
        fix={
          <FixBox>
            Install Gentle AI for this CLI, then check again. Span needs it to
            run the stages.
          </FixBox>
        }
      >
        <b className="text-hold">{agent.reason}.</b>
      </CheckLine>
    </ul>
  )
}

/** The agent CLIs Span can drive: which are ready, what is wrong with the rest, and the pick. */
export function AgentPanel({
  agents,
  picked,
  onPick,
  onCheck,
}: {
  agents: AgentState[] | null
  picked: string
  onPick: (id: string) => void
  onCheck: () => void
}) {
  const ready = (agents ?? []).filter((agent) => agent.ready).length
  return (
    <Panel
      title="Agent CLI"
      badge={
        agents === null ? null : (
          <Pill tone={ready > 0 ? "pass" : "fail"}>
            {ready > 0 ? (
              <CheckIcon aria-hidden="true" className="size-3" />
            ) : null}
            {ready} ready
          </Pill>
        )
      }
      lead="Pick the agent that does the work by default. You can change it later in Settings."
    >
      <fieldset className="flex flex-col gap-2">
        <legend className="sr-only">Agent CLI</legend>
        {(agents ?? []).map((agent) => (
          <label
            key={agent.id}
            className={cn(
              "grid cursor-pointer grid-cols-[20px_minmax(0,1fr)] gap-3 rounded-[14px] border border-input bg-card p-3.5 sm:grid-cols-[20px_minmax(0,1fr)_auto]",
              picked === agent.id && checkedCard,
              !agent.ready && "opacity-95"
            )}
          >
            <input
              type="radio"
              name="agent"
              className={roomyRadio}
              checked={picked === agent.id}
              disabled={!agent.ready}
              onChange={() => onPick(agent.id)}
            />
            <span className="flex min-w-0 flex-col gap-1">
              <span>
                <span className="text-sm font-semibold">{agent.label}</span>{" "}
                <span className="font-mono text-xs">{agent.binary}</span>
              </span>
              <Problem agent={agent} />
            </span>
            <span className="col-start-2 sm:col-start-3">{badgeOf(agent)}</span>
          </label>
        ))}
      </fieldset>
      <div>
        <Button type="button" variant="outline" onClick={onCheck}>
          <RotateCwIcon aria-hidden="true" />
          Check again
        </Button>
      </div>
    </Panel>
  )
}
