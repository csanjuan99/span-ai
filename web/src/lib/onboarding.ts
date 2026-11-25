/** Where the first-run flow stands. It is kept in this browser so a person can leave and come back. */
export interface OnboardingState {
  /** 1 welcome, 2 connect, 3 first run. */
  step: number
  skipped: boolean
  done: boolean
  /** The project folder typed in step 2, so the next visit starts with it filled in. */
  folder: string
  /** The CLI picked in step 2, applied when the first run starts. Empty keeps Span's choice. */
  agent: string
  /** When the state was last saved, in milliseconds. 0 when it never was. */
  savedAt: number
}

const key = "span:onboarding"
const first: OnboardingState = {
  step: 1,
  skipped: false,
  done: false,
  folder: "",
  agent: "",
  savedAt: 0,
}
const lastStep = 3

function clamp(step: number): number {
  return Math.min(Math.max(Math.round(step), 1), lastStep)
}

function parse(raw: string | null): OnboardingState {
  if (raw === null) {
    return first
  }
  let value: unknown
  try {
    value = JSON.parse(raw)
  } catch {
    return first
  }
  if (typeof value !== "object" || value === null) {
    return first
  }
  const body = value as Record<string, unknown>
  return {
    step: typeof body.step === "number" ? clamp(body.step) : first.step,
    skipped: body.skipped === true,
    done: body.done === true,
    folder: typeof body.folder === "string" ? body.folder : "",
    agent: typeof body.agent === "string" ? body.agent : "",
    savedAt: typeof body.savedAt === "number" ? body.savedAt : 0,
  }
}

/** The saved state, or the start when nothing was saved or storage cannot be read. */
export function readOnboarding(): OnboardingState {
  try {
    return parse(window.localStorage.getItem(key))
  } catch {
    return first
  }
}

/** Saves the fields given over what was saved. False when this browser refuses to store it. */
export function saveOnboarding(patch: Partial<OnboardingState>): boolean {
  const next = { ...readOnboarding(), ...patch, savedAt: Date.now() }
  next.step = clamp(next.step)
  try {
    window.localStorage.setItem(key, JSON.stringify(next))
    return true
  } catch {
    return false
  }
}

/** Whether a first visit should land on the welcome page: not skipped and not finished. */
export function needsOnboarding(): boolean {
  const state = readOnboarding()
  return !state.skipped && !state.done
}

/** One stage as the dry look draws it. */
export interface DryRow {
  stage: string
  does: string
  touches: string
  asks: string
}

/** What each stage does, what it can change, and whether it asks a person. Nothing here runs. */
export const dryLook: DryRow[] = [
  {
    stage: "intake",
    does: "Saves the brief and reads the story.",
    touches: "nothing",
    asks: "never",
  },
  {
    stage: "frame",
    does: "Narrows the scope and writes a plan. Asks you if the story is unclear.",
    touches: "nothing",
    asks: "may ask you",
  },
  {
    stage: "build",
    does: "An agent edits files and commits on the run branch.",
    touches: "run branch",
    asks: "never",
  },
  {
    stage: "prove",
    does: "Runs your checks and reads the results.",
    touches: "run branch",
    asks: "never",
  },
  {
    stage: "qa",
    does: "Matches each acceptance line to a test.",
    touches: "nothing",
    asks: "never",
  },
  {
    stage: "inspect",
    does: "Reads the diff and the proof.",
    touches: "nothing",
    asks: "may send it back",
  },
  {
    stage: "open",
    does: "Opens the change for review.",
    touches: "the remote",
    asks: "you approve",
  },
  {
    stage: "merge",
    does: "Merges the run branch into the base.",
    touches: "base branch",
    asks: "you approve",
  },
  {
    stage: "promote",
    does: "Runs your deploy command, if you set one.",
    touches: "your deploy",
    asks: "you approve",
  },
  {
    stage: "note",
    does: "Seals the run and records it.",
    touches: "nothing",
    asks: "never",
  },
]

/** How long ago the flow was last saved, in words. Empty when it never was. */
export function savedAgo(savedAt: number, now: number): string {
  if (savedAt <= 0) {
    return ""
  }
  const minutes = Math.max(0, Math.round((now - savedAt) / 60000))
  if (minutes < 1) {
    return "Saved just now"
  }
  if (minutes < 60) {
    return `Saved ${minutes} min ago`
  }
  const hours = Math.round(minutes / 60)
  return hours < 24
    ? `Saved ${hours} h ago`
    : `Saved ${Math.round(hours / 24)} d ago`
}

/** One phase of a run: the stages it holds, and what it means for a person. */
export interface StageGroup {
  name: string
  stages: string[]
  text: string
  /** The Tailwind span of its heading over the ten-stage row. */
  span: string
}

/** The four phases the welcome page draws over the engine's ten stages. */
export const stageGroups: StageGroup[] = [
  {
    name: "Frame",
    stages: ["intake", "frame"],
    span: "col-span-2",
    text: "Span reads the story, narrows what to build and writes a plan. It asks you when the story is unclear.",
  },
  {
    name: "Build",
    stages: ["build"],
    span: "col-span-1",
    text: "An agent edits files and commits on its own branch. Your main branch is not touched here.",
  },
  {
    name: "Prove",
    stages: ["prove", "qa", "inspect"],
    span: "col-span-3",
    text: "Your checks run, you read the diff, and the behavior is tested against the story.",
  },
  {
    name: "Ship",
    stages: ["open", "merge", "promote", "note"],
    span: "col-span-4",
    text: "The branch opens for review and merges, deploys if you set that up, and the run is sealed.",
  },
]
