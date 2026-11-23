import { stages, type Cursor, type Statement, type StatementCommand } from "@/lib/types"

/** The check that failed, and what a person needs before choosing an action. */
export interface Failure {
  stage: string
  iteration: number
  cap: number
  reason: string
  inconclusive: boolean
  command: StatementCommand | null
}

/** How the server words a hold taken because another run is editing the same checkout. */
const checkoutBusyPrefix = "another run is active in this checkout"

/** Whether a hold reason means the checkout was taken, so the run only has to wait. */
export function isCheckoutBusy(reason: string): boolean {
  return reason.startsWith(checkoutBusyPrefix)
}

/** One action a paused run offers. There is never a paused run with none. */
export interface HoldAction {
  label: string
  hint: string
}

function rank(item: Statement): number {
  const at = stages.findIndex((name) => name === item.stage)
  return item.iteration * 100 + (at < 0 ? 0 : at)
}

/**
 * The newest statement that did not advance, when the run is paused on it. The
 * statement on disk is the record: the page reads the same bytes the prompt did.
 */
export function activeFailure(input: { statements: Statement[]; cursor: Cursor | null }): Failure | null {
  const cursor = input.cursor
  if (cursor === null || (!cursor.hold && !cursor.stop)) {
    return null
  }
  const failed = input.statements.filter((item) => item.verdict !== "advance")
  if (failed.length === 0) {
    return null
  }
  const atStage = failed.filter((item) => item.stage === cursor.stage)
  const pool = atStage.length > 0 ? atStage : failed
  const newest = pool.reduce((best, item) => (rank(item) >= rank(best) ? item : best))
  const commands = newest.commands ?? []
  const command = commands.find((item) => item.exitCode !== 0 || (item.error ?? "") !== "") ?? commands[0] ?? null
  return {
    stage: newest.stage,
    iteration: newest.iteration,
    cap: newest.cap ?? 0,
    reason: newest.reason,
    inconclusive: Boolean(newest.inconclusive),
    command,
  }
}

/**
 * The one action a paused run offers. Inconclusive asks for verification before a
 * change; a failed check hands the output to build; a question goes to frame.
 */
export function holdAction(input: {
  failure: Failure | null
  questions: number
  stop: boolean
  done: boolean
  /** Nothing is executing this run, even though the cursor is not held. */
  idle?: boolean
  /** The run holds only because another run is editing its checkout. */
  busy?: boolean
}): HoldAction | null {
  if (input.done || input.stop) {
    return null
  }
  // A busy checkout outranks an old failure: the failing statement is still on disk, but
  // the run is waiting for the other run, not for a fix.
  if (input.busy === true && input.questions === 0) {
    return {
      label: "Resume when the other run ends",
      hint: "Another run is editing this checkout. Let it finish or stop it, then resume this one.",
    }
  }
  if (input.questions > 0) {
    return { label: "Answer and continue", hint: "Frame reads the answer and starts again." }
  }
  if (input.failure === null) {
    if (input.idle === true) {
      return {
        label: "Continue",
        hint: "Nothing is running this span. Continue picks it up at the stage the cursor points to.",
      }
    }
    return { label: "Resume", hint: "This stage starts again from the cursor." }
  }
  if (input.failure.inconclusive) {
    return {
      label: "Re-verify",
      hint: "The same command passed and failed on this commit. Verify before changing code.",
    }
  }
  return {
    label: "Fix and retry",
    hint: "Build receives this output and fixes the failure instead of restarting the feature.",
  }
}

/** The attempt counter a reader can act on. Zero means unlimited is not allowed. */
export function attemptLine(failure: Failure): string {
  if (failure.cap <= 0) {
    return ""
  }
  return `attempt ${failure.iteration} of ${failure.cap}`
}
