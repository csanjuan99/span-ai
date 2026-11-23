import { stages } from "@/lib/types"

function stageAt(name: string): number {
  return stages.findIndex((item) => item === name)
}

/** The stage a note should target before the person picks one. */
export function noteStage(input: { current: string; hold: boolean; stop: boolean }): string {
  if (input.hold || input.stop) {
    return input.current || "frame"
  }
  const at = stageAt(input.current)
  if (at >= 0 && at < stages.length - 1) {
    return stages[at + 1] ?? input.current
  }
  return input.current || "frame"
}

/** When a saved note is actually read. The stage already running does not see it. */
export function instructionTiming(input: { current: string; selected: string; hold: boolean; stop: boolean }): string {
  if (input.selected === "intake") {
    return "Intake does not call an agent. A note saved here is kept, and it does not steer this run."
  }
  if ((input.hold || input.stop) && input.selected === input.current) {
    return "Resume reads this note when this stage starts again."
  }
  const currentAt = stageAt(input.current)
  const selectedAt = stageAt(input.selected)
  if (currentAt >= 0 && selectedAt > currentAt) {
    return "This stage has not started. It reads this note when it starts."
  }
  return "This stage has already started. It reads a note only when it starts, so this note waits until the next time it runs."
}

/** What the current stage is doing, in the words a person waiting can use. */
export function stageLine(input: {
  stage: string
  reason: string
  hold: boolean
  stop: boolean
  done: boolean
}): string {
  if (input.done) {
    return input.reason !== "" ? input.reason : "This run is finished."
  }
  if (input.stop) {
    return input.reason !== "" ? input.reason : "This run stopped."
  }
  if (input.hold) {
    return input.reason !== "" ? input.reason : "Paused. Read the ledger, then resume the run."
  }
  const running: Record<string, string> = {
    intake: "Saving the story. This finishes on its own, then an agent reads it.",
    frame: "An agent is reading the story. The ledger updates when that read finishes.",
    build: "The agent is writing the change. The transcript grows while this stage is open.",
    prove: "Running the project's tests.",
    qa: "Matching each acceptance line to a test.",
    inspect: "Reading the diff.",
    open: "Opening the change.",
    merge: "Merging the change.",
    promote: "Running the deploy command.",
    note: "Writing the closing note.",
  }
  return running[input.stage] ?? "This stage is still running."
}
