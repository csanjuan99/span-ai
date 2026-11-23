import { describe, expect, it } from "vitest"

import { activeStep, doneCount, setupSteps, type SetupFacts } from "@/lib/setup"

const none: SetupFacts = {
  folderReady: false,
  folderDetail: "",
  agentReady: false,
  agentDetail: "",
  jiraConfigured: false,
  jiraDetail: "",
  runs: 0,
}

describe("setupSteps", () => {
  it("lists the four steps in the order a person meets them", () => {
    expect(setupSteps(none).map((step) => step.id)).toEqual(["folder", "agent", "jira", "story"])
  })

  it("marks a step done only from what the desk read", () => {
    const steps = setupSteps({ ...none, folderReady: true, folderDetail: "3 repositories found", agentReady: true, agentDetail: "Pi · m" })
    expect(steps.map((step) => step.done)).toEqual([true, true, false, false])
    expect(steps[0]?.detail).toBe("3 repositories found")
    expect(steps[1]?.detail).toBe("Pi · m")
    expect(doneCount(steps)).toBe(2)
  })

  it("points at the first step still open", () => {
    expect(activeStep(setupSteps({ ...none, folderReady: true }))?.id).toBe("agent")
    expect(activeStep(setupSteps({ folderReady: true, folderDetail: "", agentReady: true, agentDetail: "", jiraConfigured: true, jiraDetail: "", runs: 1 }))).toBeUndefined()
  })

  it("says why Jira is not connected when it could not be read", () => {
    expect(setupSteps({ ...none, jiraProblem: "jira is down" })[2]?.detail).toBe("jira is down")
  })
})
