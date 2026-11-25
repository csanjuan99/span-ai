/** Where the first-run flow stands. It is kept in this browser so a person can leave and come back. */
export interface OnboardingState {
  /** 1 welcome, 2 connect, 3 first run. */
  step: number
  skipped: boolean
  done: boolean
  /** The project folder typed in step 2, so the next visit starts with it filled in. */
  folder: string
}

const key = "span:onboarding"
const first: OnboardingState = {
  step: 1,
  skipped: false,
  done: false,
  folder: "",
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
  const next = { ...readOnboarding(), ...patch }
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
