"use client"

import { SelectField } from "@/components/settings-edit/SelectField"
import type { InterviewAgents } from "@/components/shape-entry/useInterviewAgents"

/** The agent select: Auto, then every CLI that is ready (and the one already chosen). */
export function AgentSelect({
  label = "Interview agent",
  agents,
  value,
  className,
  wrapperClassName,
  onChange,
}: {
  label?: string
  agents: InterviewAgents
  value: string
  className?: string
  wrapperClassName?: string
  onChange: (agent: string) => void
}) {
  return (
    <SelectField
      mono
      label={label}
      value={value}
      className={className}
      wrapperClassName={wrapperClassName}
      options={[
        { value: "", label: "Auto" },
        ...agents.choice.agents
          .filter((row) => row.ready || row.id === value)
          .map((row) => ({ value: row.id, label: row.id })),
      ]}
      onChange={onChange}
    />
  )
}

/** The model select: Auto, then the models the CLI lists (and the one already chosen). */
export function ModelSelect({
  agents,
  value,
  className,
  wrapperClassName,
  onChange,
}: {
  agents: InterviewAgents
  value: string
  className?: string
  wrapperClassName?: string
  onChange: (model: string) => void
}) {
  const listed = agents.models.some((model) => model.id === value)
  return (
    <SelectField
      mono
      label="Model"
      value={value}
      className={className}
      wrapperClassName={wrapperClassName}
      options={[
        { value: "", label: "Auto" },
        ...(value !== "" && !listed ? [{ value, label: value }] : []),
        ...agents.models.map((model) => ({ value: model.id, label: model.id })),
      ]}
      onChange={onChange}
    />
  )
}
