import type { ConflictHoldKind } from "@/lib/conflict"
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

/**
 * The run that holds the checkout, when the hold reason names it ("(pid 4321 run span-7f3a)").
 * Empty when the holder is another kind of process or the reason is about something else.
 */
export function holderOf(reason: string): string {
  if (!isCheckoutBusy(reason)) {
    return ""
  }
  return /\(pid \d+ run ([\w.-]+)\)/.exec(reason)?.[1] ?? ""
}

/** One action a paused run offers. There is never a paused run with none. */
export interface HoldAction {
  label: string
  hint: string
}

/** What a person can approve when a stage is held by its mode rather than by a failure. */
export interface Approval extends HoldAction {
  stage: string
  environment: string
}

const modeHold = /^(\S+) is hold$/

/**
 * The approval a run is waiting for. A stage in `hold` mode pauses before its effect and only
 * a person releases it, so the page offers that, never "Fix and retry", which would reach the
 * same hold again. Any other reason is a failure or a wait and has no approval.
 */
export function approvalOf(cursor: Cursor | null): Approval | null {
  if (cursor === null || !cursor.hold || cursor.stop || cursor.done) {
    return null
  }
  const named = modeHold.exec(cursor.reason)?.[1]
  if (named === undefined) {
    return null
  }
  if (cursor.stage === "open" || cursor.stage === "merge") {
    return named === cursor.stage
      ? {
          stage: cursor.stage,
          environment: "",
          label: `Approve ${cursor.stage}`,
          hint: `${cursor.stage}: hold waits for a person. Approving lets this commit ${cursor.stage === "merge" ? "merge" : "open its change"}; a retry would hold again.`,
        }
      : null
  }
  if (cursor.stage === "promote") {
    return {
      stage: "promote",
      environment: named,
      label: `Approve promote to ${named}`,
      hint: `${named}: hold waits for a person. Approving runs this environment for this commit.`,
    }
  }
  return null
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

/** `choose` has the conflict panel's controls; an abort only needs the normal resume. */
function conflictAction(kind: ConflictHoldKind): HoldAction | null {
  if (kind === "choose") {
    return null
  }
  return {
    label: "Resume",
    hint: "The merge was aborted and the branch is as it was before. Resume tries the merge stage again.",
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
  /** The run holds in the merge-conflict phase. The conflict panel owns `choose`. */
  conflict?: ConflictHoldKind | null
}): HoldAction | null {
  if (input.done || input.stop) {
    return null
  }
  // A conflict hold is not a failed check: the merge statement is on disk, but what the
  // run waits for is a choice on the cards, which has its own controls.
  if (input.conflict != null) {
    return conflictAction(input.conflict)
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
