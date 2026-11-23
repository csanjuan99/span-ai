/** The stages that call a model. The rest run commands. */
export const agentStages = ["frame", "build", "inspect"] as const

export function usesAgent(stage: string): boolean {
  return (agentStages as readonly string[]).includes(stage)
}

/** The stage the picker opens on: the one the run stopped at, when it calls a model. */
export function defaultAgentStage(current: string): string {
  return usesAgent(current) ? current : "build"
}

/** The chosen CLI for a stage, or an empty string when the policy decides. */
export function agentForStage(stage: string, choices: Record<string, string> | undefined): string {
  return choices?.[stage] ?? ""
}

/** One sentence per stage so the choice is readable without opening the picker. */
export function stageAgentLine(choices: Record<string, string> | undefined): string {
  if (!choices) {
    return ""
  }
  const parts = agentStages
    .map((stage) => (choices[stage] ? `${stage} uses ${choices[stage]}` : ""))
    .filter((part) => part !== "")
  return parts.join(", ")
}
