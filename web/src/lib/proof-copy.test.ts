import { describe, expect, it } from "vitest"

import { diffCopy, emptyProof, testsCopy, verdictCopy, type RunPosition } from "@/lib/proof-copy"
import type { Proof } from "@/lib/proof"

const checks = (over: Partial<Proof["checks"]> = {}): Proof["checks"] => ({
  passed: 0,
  total: 0,
  failing: "",
  inspect: "not yet",
  tests: null,
  ...over,
})
const proof = (over: Partial<Proof["checks"]> = {}, files = 0): Proof => ({
  ...emptyProof(),
  diff: { files, added: 0, removed: 0, kinds: "A 1 · M 3" },
  checks: checks(over),
})
const at = (kind: RunPosition["kind"], stage: string, sealed = 3): RunPosition => ({ kind, stage, sealed })

describe("verdictCopy", () => {
  it("says the stage is running and how many stages are sealed", () => {
    expect(verdictCopy(at("running", "build"), proof())).toEqual({ title: "Build running", caption: "sealed stages: 3 of 10" })
  })

  it("says a stage waiting for the person is paused, not failed", () => {
    expect(verdictCopy(at("paused", "frame"), proof())).toEqual({ title: "Frame paused", caption: "frame is not sealed yet" })
  })

  it("holds at the stage and counts the checks that are not green", () => {
    expect(verdictCopy(at("held", "prove"), proof({ passed: 3, total: 5 }))).toEqual({
      title: "Held at prove",
      caption: "2 of 5 checks not green",
    })
  })

  it("says everything before the stage passed when only the stage itself holds", () => {
    expect(verdictCopy(at("held", "merge"), proof({ passed: 4, total: 4 }))).toEqual({
      title: "Held at merge",
      caption: "everything before merge passed",
    })
  })

  it("says no check ran when the agent could not start or the checkout is taken", () => {
    expect(verdictCopy(at("blocked", "build"), proof()).caption).toBe("Agent could not run · no check ran")
    expect(verdictCopy(at("busy", "build"), proof()).caption).toBe("Waiting for the checkout · no check ran")
  })

  it("is ready to approve when the checks are green", () => {
    expect(verdictCopy(at("approval", "open"), proof({ passed: 3, total: 3, inspect: "passed" }))).toEqual({
      title: "Ready to open",
      caption: "3 of 3 checks green · inspect found nothing blocking",
    })
  })

  it("calls a finished run shipped", () => {
    expect(verdictCopy(at("done", "note", 10), proof())).toEqual({ title: "Shipped", caption: "10 of 10 stages sealed" })
  })
})

describe("diffCopy", () => {
  it("says what the stage did to the repo when there is no code yet", () => {
    expect(diffCopy(at("paused", "frame"), proof())).toEqual({ label: "Diff", caption: "frame does not touch the repo" })
    expect(diffCopy(at("busy", "build"), proof()).caption).toBe("build has not started")
    expect(diffCopy(at("blocked", "build"), proof()).caption).toBe("build has not written anything")
  })

  it("calls the diff of a running build live", () => {
    expect(diffCopy(at("running", "build"), proof({}, 4))).toEqual({ label: "Diff so far", caption: "A 1 · M 3 · live" })
  })
})

describe("testsCopy", () => {
  it("says tests have not run yet and when they will", () => {
    expect(testsCopy(at("running", "build"), proof())).toEqual({ value: "Not run yet", caption: "prove runs right after build" })
    expect(testsCopy(at("held", "build"), proof())).toEqual({ value: "Not run", caption: "prove comes after build" })
  })

  it("leaves the cell to the real counts once a check ran", () => {
    expect(testsCopy(at("held", "prove"), proof({ total: 2, passed: 1 }))).toBeNull()
  })
})

describe("a run that ended without sealing its build", () => {
  it("reads not sealed, with tests not run", () => {
    expect(verdictCopy(at("unsealed", "build"), proof())).toEqual({ title: "Not sealed", caption: "no proof results to show" })
    expect(testsCopy(at("unsealed", "note", 10), proof())).toEqual({ value: "Not run", caption: "build never sealed" })
  })
})
