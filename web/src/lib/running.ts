import { usesAgent } from "@/lib/stage-agents"
import { stages } from "@/lib/types"

const names: Record<string, string> = {
  "claude-code": "Claude Code",
  claude: "Claude Code",
  codex: "Codex",
  cursor: "Cursor",
  "cursor-agent": "Cursor",
  pi: "Pi",
}

/** What the page says while a run goes on by itself: who works, what comes next, where. */
export interface Running {
  /** The agent CLI that works, as the engine names it. */
  agent: string
  title: string
  /** The last line the agent wrote, or empty. */
  now: string
  next: string
  where: string
  /** What pausing would do, or empty after the last stage. */
  pause: string
}

function cap(text: string): string {
  return text.charAt(0).toUpperCase() + text.slice(1)
}

/** A pause stops an agent in flight; any other stage finishes before the run holds. */
function pauseOf(stage: string, following: string | undefined): string {
  if (usesAgent(stage)) {
    return `Pausing stops the agent. The run holds at ${stage}, which runs again on resume.`
  }
  return following === undefined
    ? ""
    : `The run holds after ${stage} seals and before ${following} starts. Nothing is lost.`
}

/**
 * Reads the running card from what the page already knows. It states only what the engine
 * guarantees: the order of the stages. Which stage asks for an approval depends on the run's
 * modes, which the page is not given, so the card does not name one.
 */
export function runningOf(input: {
  stage: string
  agent: string
  branch: string
  workspace: string
  transcript: string[]
}): Running {
  const index = stages.findIndex((stage) => stage === input.stage)
  const following = index >= 0 ? stages[index + 1] : undefined
  const folder = input.workspace.split("/").filter(Boolean).pop() ?? ""
  const now =
    [...input.transcript].reverse().find((line) => line.trim() !== "") ?? ""
  return {
    agent: input.agent,
    title: `${names[input.agent] ?? "The agent"} is working in ${input.stage}.`,
    now: now.trim(),
    next:
      following === undefined
        ? `${cap(input.stage)} seals and the run is done.`
        : `${cap(input.stage)} seals, then ${following} runs on its own.`,
    where: [folder, input.branch].filter((part) => part !== "").join(" · "),
    pause: pauseOf(input.stage, following),
  }
}
