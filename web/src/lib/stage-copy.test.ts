import { describe, expect, it } from "vitest"

import { instructionTiming, noteStage, stageLine } from "@/lib/stage-copy"

describe("stageLine", () => {
  it("tells a person waiting on intake that the story is being saved", () => {
    const line = stageLine({ stage: "intake", reason: "", hold: false, stop: false, done: false })
    expect(line).toBe("Saving the story. This finishes on its own, then an agent reads it.")
  })

  it("keeps a hold reason as the sentence", () => {
    expect(stageLine({ stage: "frame", reason: "Acceptance criteria are missing.", hold: true, stop: false, done: false })).toBe(
      "Acceptance criteria are missing.",
    )
  })
})

describe("instructionTiming", () => {
  it("points a running stage at the next one", () => {
    expect(noteStage({ current: "frame", hold: false, stop: false })).toBe("build")
  })

  it("keeps a paused stage on itself", () => {
    expect(noteStage({ current: "frame", hold: true, stop: false })).toBe("frame")
  })

  it("says a later stage reads the note when it starts", () => {
    expect(instructionTiming({ current: "frame", selected: "build", hold: false, stop: false })).toBe(
      "This stage has not started. It reads this note when it starts.",
    )
  })

  it("says the stage already running waits for the next time it starts", () => {
    expect(instructionTiming({ current: "frame", selected: "frame", hold: false, stop: false })).toBe(
      "This stage has already started. It reads a note only when it starts, so this note waits until the next time it runs.",
    )
  })

  it("says intake does not steer an agent", () => {
    expect(instructionTiming({ current: "intake", selected: "intake", hold: false, stop: false })).toBe(
      "Intake does not call an agent. A note saved here is kept, and it does not steer this run.",
    )
  })

  it("says resume reads the note for the paused stage", () => {
    expect(instructionTiming({ current: "frame", selected: "frame", hold: true, stop: false })).toBe(
      "Resume reads this note when this stage starts again.",
    )
  })
})
