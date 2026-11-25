import { describe, expect, it } from "vitest"

import { activeFailure, approvalLede, approvalOf, attemptLine, checksBefore, holdAction, holderOf, isCheckoutBusy } from "@/lib/failure"
import type { Cursor, Statement } from "@/lib/types"

function cursor(over: Partial<Cursor> = {}): Cursor {
  return {
    id: "s",
    stage: "prove",
    iteration: 2,
    hold: true,
    stop: false,
    done: false,
    reason: "go test ./... exited 1",
    gitSHA: "abc123",
    changeURL: "",
    branch: "",
    base: "",
    workspace: "",
    issue: "",
    questions: [],
    ...over,
  }
}

function statement(over: Partial<Statement> = {}): Statement {
  return {
    stage: "prove",
    iteration: 2,
    verdict: "hold",
    reason: "go test ./... exited 1",
    commands: [
      {
        command: "go test ./...",
        exitCode: 1,
        output: "FAIL src/cart.test.ts > totals\nexpected 2 to be 3",
        logPath: "iter/2/logs/prove-1.log",
      },
    ],
    cap: 3,
    ...over,
  }
}

describe("activeFailure", () => {
  it("reads the failing command and its output from the statement", () => {
    const failure = activeFailure({ statements: [statement()], cursor: cursor() })
    expect(failure).not.toBeNull()
    expect(failure?.command?.command).toBe("go test ./...")
    expect(failure?.command?.output).toContain("expected 2 to be 3")
    expect(failure?.command?.logPath).toBe("iter/2/logs/prove-1.log")
    expect(attemptLine(failure!)).toBe("attempt 2 of 3")
  })

  it("prefers the statement of the stage the cursor stopped on", () => {
    const failure = activeFailure({
      statements: [statement({ stage: "qa", iteration: 2, reason: "gate" }), statement()],
      cursor: cursor(),
    })
    expect(failure?.stage).toBe("prove")
  })

  it("says nothing while the run is not paused", () => {
    expect(activeFailure({ statements: [statement()], cursor: cursor({ hold: false }) })).toBeNull()
  })

  it("says nothing when every statement advanced", () => {
    expect(activeFailure({ statements: [statement({ verdict: "advance" })], cursor: cursor() })).toBeNull()
  })
})

describe("holdAction", () => {
  const failure = activeFailure({ statements: [statement()], cursor: cursor() })

  it("offers one action for a failed check", () => {
    const action = holdAction({ failure, questions: 0, stop: false, done: false })
    expect(action?.label).toBe("Fix and retry")
  })

  it("asks for verification before a change when the result flipped", () => {
    const flipped = activeFailure({
      statements: [statement({ inconclusive: true })],
      cursor: cursor({ reason: "inconclusive" }),
    })
    expect(holdAction({ failure: flipped, questions: 0, stop: false, done: false })?.label).toBe("Re-verify")
  })

  it("sends a question to frame", () => {
    expect(holdAction({ failure: null, questions: 1, stop: false, done: false })?.label).toBe("Answer and continue")
  })

  it("never leaves a paused run with no action", () => {
    expect(holdAction({ failure: null, questions: 0, stop: false, done: false })?.label).toBe("Resume")
  })

  it("offers to continue a run that is not executing and not held", () => {
    const action = holdAction({ failure: null, questions: 0, stop: false, done: false, idle: true })
    expect(action?.label).toBe("Continue")
    expect(action?.hint).toContain("Nothing is running")
  })

  it("offers nothing on a stopped or finished run", () => {
    expect(holdAction({ failure, questions: 0, stop: true, done: false })).toBeNull()
    expect(holdAction({ failure, questions: 0, stop: false, done: true })).toBeNull()
  })
})

describe("a run held because the checkout was taken", () => {
  it("is recognised by the server's message", () => {
    expect(isCheckoutBusy("another run is active in this checkout (pid 4 run span-3)")).toBe(true)
    expect(isCheckoutBusy("go test ./... exited 1")).toBe(false)
    expect(isCheckoutBusy("")).toBe(false)
  })

  it("offers to resume once the other run ends, instead of a generic retry", () => {
    const action = holdAction({ failure: null, questions: 0, stop: false, done: false, busy: true })
    expect(action?.label).toBe("Resume when the other run ends")
    expect(action?.hint).toContain("Another run")
  })

  it("keeps the generic resume for any other hold", () => {
    const action = holdAction({ failure: null, questions: 0, stop: false, done: false, busy: false })
    expect(action?.label).toBe("Resume")
  })
})

describe("a failed run resumed into a busy checkout", () => {
  it("offers the wait, not a retry of the old failure", () => {
    const failure = activeFailure({ statements: [statement()], cursor: cursor() })
    const action = holdAction({ failure, questions: 0, stop: false, done: false, busy: true })
    expect(action?.label).toBe("Resume when the other run ends")
  })
})

describe("approvalOf", () => {
  it("offers the approval of a stage held by its mode, not a retry", () => {
    expect(approvalOf(cursor({ stage: "merge", reason: "merge is hold" }))).toEqual({
      stage: "merge",
      environment: "",
      label: "Approve merge",
      hint: "merge: hold waits for a person. Approving lets this commit merge; a retry would hold again.",
    })
    expect(approvalOf(cursor({ stage: "open", reason: "open is hold" }))?.label).toBe("Approve open")
  })

  it("names the environment a promote is held at", () => {
    const got = approvalOf(cursor({ stage: "promote", reason: "preview is hold" }))
    expect(got?.stage).toBe("promote")
    expect(got?.environment).toBe("preview")
    expect(got?.label).toBe("Approve promote to preview")
  })

  it("is absent for every other hold, a finished run and a missing cursor", () => {
    expect(approvalOf(cursor({ stage: "merge", reason: "merge outcome unknown: network" }))).toBeNull()
    expect(approvalOf(cursor({ stage: "prove", reason: "prove is hold" }))).toBeNull()
    expect(approvalOf(cursor({ stage: "merge", reason: "merge is hold", hold: false }))).toBeNull()
    expect(approvalOf(cursor({ stage: "merge", reason: "merge is hold", done: true }))).toBeNull()
    expect(approvalOf(null)).toBeNull()
  })
})

describe("a run held on a merge conflict", () => {
  const failure = activeFailure({ statements: [statement()], cursor: cursor() })

  it("offers no generic action while the person chooses: the conflict panel owns it", () => {
    expect(holdAction({ failure, questions: 0, stop: false, done: false, conflict: "choose" })).toBeNull()
  })

  it("offers the normal resume after an abort, with a neutral hint", () => {
    const action = holdAction({ failure, questions: 0, stop: false, done: false, conflict: "aborted" })
    expect(action?.label).toBe("Resume")
    expect(action?.hint).toContain("merge was aborted")
  })

  it("keeps the old behavior when there is no conflict", () => {
    expect(holdAction({ failure, questions: 0, stop: false, done: false, conflict: null })?.label).toBe("Fix and retry")
  })
})

describe("holderOf", () => {
  it("reads the run that holds the checkout from the hold reason", () => {
    expect(holderOf("another run is active in this checkout (pid 4321 run span-7f3a)")).toBe("span-7f3a")
  })

  it("is empty when the holder is not a run, or the reason is not about the checkout", () => {
    expect(holderOf("another run is active in this checkout (pid 4321 shape)")).toBe("")
    expect(holderOf("another run is active in this checkout (another process)")).toBe("")
    expect(holderOf("prove exited 1")).toBe("")
  })
})

describe("an approval's explanation", () => {
  it("names the sealed stages and the checks that passed before it", () => {
    expect(approvalLede("open", ["prove", "qa", "inspect"])).toBe(
      "prove, qa and inspect passed on this commit. The stage is held by its mode, so it waits for a person. A retry would hold again."
    )
    expect(
      checksBefore(
        [
          {
            stage: "prove",
            iteration: 1,
            verdict: "advance",
            reason: "",
            commands: [{ command: "go test ./...", exitCode: 0, output: "" }],
          },
          {
            stage: "open",
            iteration: 1,
            verdict: "hold",
            reason: "open is hold",
            commands: [{ command: "gh pr create", exitCode: 0 }],
          },
        ],
        "open"
      ).map((command) => command.command)
    ).toEqual(["go test ./..."])
  })
})
