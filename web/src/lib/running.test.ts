import { describe, expect, it } from "vitest"

import { runningOf } from "@/lib/running"

const base = { stage: "build", agent: "claude-code", branch: "feat/x", workspace: "/w/billing-api", transcript: [] as string[] }

describe("runningOf", () => {
  it("names the agent and the stage, and what the run does when the stage seals", () => {
    const running = runningOf(base)
    expect(running.title).toBe("Claude Code is working in build.")
    expect(running.next).toBe("Build seals, then prove runs on its own.")
    expect(running.pause).toBe("Pausing stops the agent. The run holds at build, which runs again on resume.")
  })

  it("says a stage without an agent finishes before the run holds", () => {
    expect(runningOf({ ...base, stage: "prove" }).pause).toBe(
      "The run holds after prove seals and before qa starts. Nothing is lost."
    )
  })

  it("says where the run is: the folder name and the branch", () => {
    expect(runningOf(base).where).toBe("billing-api · feat/x")
    expect(runningOf({ ...base, branch: "" }).where).toBe("billing-api")
  })

  it("shows the last line the agent wrote as what it is doing now, if any", () => {
    expect(runningOf({ ...base, transcript: ["a", "", "editing retry.go"] }).now).toBe("editing retry.go")
    expect(runningOf(base).now).toBe("")
  })

  it("has nothing to say after the last stage", () => {
    const last = runningOf({ ...base, stage: "note" })
    expect(last.next).toBe("Note seals and the run is done.")
    expect(last.pause).toBe("")
  })

  it("falls back to a plain name when the agent is not one it knows", () => {
    expect(runningOf({ ...base, agent: "" }).title).toBe("The agent is working in build.")
  })
})
