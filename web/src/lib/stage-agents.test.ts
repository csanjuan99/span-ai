import { describe, expect, it } from "vitest"

import {
  agentForStage,
  defaultAgentStage,
  restartStage,
  stageAgentLine,
  usesAgent,
} from "@/lib/stage-agents"

describe("stage agents", () => {
  it("knows which stages call a model", () => {
    expect(usesAgent("build")).toBe(true)
    expect(usesAgent("inspect")).toBe(true)
    expect(usesAgent("prove")).toBe(false)
    expect(usesAgent("promote")).toBe(false)
  })

  it("opens on the stage the run stopped at when it calls a model", () => {
    expect(defaultAgentStage("inspect")).toBe("inspect")
    expect(defaultAgentStage("qa")).toBe("build")
    expect(defaultAgentStage("")).toBe("build")
  })

  it("reads the choice for one stage and leaves the others to the policy", () => {
    const choices = { build: "pi" }
    expect(agentForStage("build", choices)).toBe("pi")
    expect(agentForStage("frame", choices)).toBe("")
    expect(agentForStage("frame", undefined)).toBe("")
  })

  it("summarizes the exceptions without listing the stages that follow the policy", () => {
    expect(stageAgentLine({ build: "pi", inspect: "claude" })).toBe("build uses pi, inspect uses claude")
    expect(stageAgentLine({})).toBe("")
    expect(stageAgentLine(undefined)).toBe("")
  })
})

describe("restartStage", () => {
  it("starts again at the stage that changes the code, which is build for every check after it", () => {
    expect(restartStage("prove")).toBe("build")
    expect(restartStage("qa")).toBe("build")
    expect(restartStage("merge")).toBe("build")
  })

  it("restarts frame and build at themselves, and falls back to build", () => {
    expect(restartStage("frame")).toBe("frame")
    expect(restartStage("build")).toBe("build")
    expect(restartStage("intake")).toBe("build")
    expect(restartStage("")).toBe("build")
  })
})
