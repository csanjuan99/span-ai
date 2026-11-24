/** What the run page asks of the person right now, in the order the page already resolves it. */
export interface NeedsInput {
  /** The agent CLI cannot work until a person acts. */
  agentBlocked: boolean
  questions: boolean
  decision: boolean
  /** The run is resolving a merge conflict. */
  conflict: boolean
  /** A stage held by its mode waits for approval. */
  approval: boolean
  /** The run holds only because another run is editing its checkout. */
  busy: boolean
  /** A check failed and the run is paused on it. */
  failed: boolean
  stop: boolean
  /** Nothing is executing this run, even though the cursor is not held. */
  idle: boolean
  hold: boolean
  done: boolean
}

export type NeedsKind =
  | "agent-blocked"
  | "questions"
  | "decision"
  | "conflict"
  | "approval"
  | "busy"
  | "failed"
  | "stopped"
  | "idle"
  | "held"

/** `hold` waits for a person; `fail` is a stop or a failed check. */
export interface Needs {
  kind: NeedsKind
  tone: "hold" | "fail"
  label: string
}

/**
 * The single thing the run waits on, or null while it goes on by itself. One answer, so the
 * page shows one block above the tabs instead of several that compete. A blocked agent comes
 * first because nothing else can move; a busy checkout and a conflict come before a failed
 * check because the failing statement is still on disk but is not what the run waits for.
 */
export function needsYou(input: NeedsInput): Needs | null {
  if (input.done) {
    return null
  }
  if (input.agentBlocked) {
    return { kind: "agent-blocked", tone: "hold", label: "Agent blocked" }
  }
  if (input.questions) {
    return { kind: "questions", tone: "hold", label: "Waiting for your answers" }
  }
  if (input.decision) {
    return { kind: "decision", tone: "hold", label: "Decision needed" }
  }
  if (input.conflict) {
    return { kind: "conflict", tone: "hold", label: "Merge conflict" }
  }
  if (input.approval) {
    return { kind: "approval", tone: "hold", label: "Approval needed" }
  }
  if (input.busy) {
    return { kind: "busy", tone: "hold", label: "Waiting for another run" }
  }
  if (input.failed) {
    return { kind: "failed", tone: "fail", label: "Check failed" }
  }
  if (input.stop) {
    return { kind: "stopped", tone: "fail", label: "Run stopped" }
  }
  if (input.idle) {
    return { kind: "idle", tone: "hold", label: "No agent is running" }
  }
  if (input.hold) {
    return { kind: "held", tone: "hold", label: "Run paused" }
  }
  return null
}
