/** The four things a first run needs, in the order a person meets them. */
export type StepId = "folder" | "agent" | "jira" | "story"

export interface SetupStep {
  id: StepId
  /** What the person gets, phrased as an outcome. */
  title: string
  done: boolean
  /** One line that says what is true now, or what is missing. */
  detail: string
}

/** What the desk already knows. Nothing here is a flag the person ticks by hand. */
export interface SetupFacts {
  /** The folder is a repository or holds some. */
  folderReady: boolean
  folderDetail: string
  /** A CLI with Gentle AI can run on this machine. */
  agentReady: boolean
  /** Label and model of the CLI that would run, "" when none is ready. */
  agentDetail: string
  jiraConfigured: boolean
  jiraDetail: string
  /** Why the session could not be read or saved, "" when nothing went wrong. */
  jiraProblem?: string
  /** Runs this server already knows. */
  runs: number
}

/** Builds the steps from what the desk read. The order is fixed. */
export function setupSteps(facts: SetupFacts): SetupStep[] {
  return [
    {
      id: "folder",
      title: "Pick the folder to work in",
      done: facts.folderReady,
      detail: facts.folderReady ? facts.folderDetail : "Type a folder with repositories, or a repository, and find its repos.",
    },
    {
      id: "agent",
      title: "Choose the agent and its model",
      done: facts.agentReady,
      detail: facts.agentReady ? facts.agentDetail : "No CLI with Gentle AI is ready on this machine yet.",
    },
    {
      id: "jira",
      title: "Connect Jira",
      done: facts.jiraConfigured,
      detail: facts.jiraConfigured
        ? facts.jiraDetail
        : (facts.jiraProblem ?? "") !== ""
          ? (facts.jiraProblem ?? "")
          : "Needed to load a story by its key. Without it you can still paste a story.",
    },
    {
      id: "story",
      title: "Run your first story",
      done: facts.runs > 0,
      detail: facts.runs > 0 ? `${facts.runs} ${facts.runs === 1 ? "run" : "runs"} so far.` : "Load or paste a story, then run it.",
    },
  ]
}

/** The first step still open, which is the one worth the person's attention. */
export function activeStep(steps: SetupStep[]): SetupStep | undefined {
  return steps.find((step) => !step.done)
}

/** How many steps are done. */
export function doneCount(steps: SetupStep[]): number {
  return steps.filter((step) => step.done).length
}
