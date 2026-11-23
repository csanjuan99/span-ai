import type { JournalEvent } from "@/hooks/use-events"

/** The CLI and model one stage last ran with. */
export interface RanWith {
  stage: string
  agent: string
  /** "" when the CLI was left on the model Gentle AI configured. */
  model: string
}

/** The stages in the order a run meets them, so the list reads like the run. */
const order = ["frame", "build", "inspect"]

/** The last agent.run of each stage, from the journal. What ran, not what was configured. */
export function ranWith(events: JournalEvent[]): RanWith[] {
  const last = new Map<string, RanWith>()
  for (const event of events) {
    if (event.type !== "agent.run" || event.detail.stage === undefined || event.detail.stage === "") {
      continue
    }
    last.set(event.detail.stage, { stage: event.detail.stage, agent: event.detail.agent ?? "", model: event.detail.model ?? "" })
  }
  return [...last.values()].sort((a, b) => rank(a.stage) - rank(b.stage))
}

function rank(stage: string): number {
  const index = order.indexOf(stage)
  return index === -1 ? order.length : index
}
