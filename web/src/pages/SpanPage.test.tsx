import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react"
import { MemoryRouter, Route, Routes } from "react-router"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

import type { SpanState } from "@/hooks/use-span"
import type { Cursor, Snapshot, Statement } from "@/lib/types"

const approveStage = vi.fn()
const resumeRun = vi.fn()
const pauseRun = vi.fn()
const saveInstruction = vi.fn()
let state: SpanState

vi.mock("@/hooks/use-span", () => ({ useSpan: () => state }))
vi.mock("@/lib/api", async (original) => {
  const real = await original<typeof import("@/lib/api")>()
  return {
    ...real,
    approveStage: (id: string, stage: string, environment: string) =>
      approveStage(id, stage, environment),
    resumeRun: (id: string) => resumeRun(id),
    pauseRun: (id: string) => pauseRun(id),
    saveInstruction: (id: string, stage: string, body: string) =>
      saveInstruction(id, stage, body),
  }
})

import { getRunContext } from "@/lib/help"
import { SpanPage } from "@/pages/SpanPage"

function cursor(over: Partial<Cursor>): Cursor {
  return {
    id: "span-1",
    stage: "merge",
    iteration: 1,
    hold: true,
    stop: false,
    done: false,
    reason: "merge is hold",
    gitSHA: "abc123",
    changeURL: "https://example.test/pull/9",
    branch: "feat/x",
    base: "dev",
    workspace: "",
    issue: "",
    questions: [],
    ...over,
  }
}

/** A sealed stage, for the ledger. */
const advance = (stage: string): Statement => ({
  stage,
  iteration: 1,
  verdict: "advance",
  reason: "",
  commands: [],
})

function held(over: Partial<Cursor>, statements: Statement[] = []): SpanState {
  const span: Snapshot = {
    cursor: cursor(over),
    statements,
    acceptance: [],
    gate: null,
  }
  return {
    span,
    acceptance: null,
    implementation: null,
    transcript: [],
    frames: [],
    actions: [],
    agent: null,
    agentAt: "",
    note: () => undefined,
    error: "",
    connected: true,
    problem: null,
  }
}

function tree() {
  return (
    <MemoryRouter initialEntries={["/spans/span-1"]}>
      <Routes>
        <Route path="/spans/:id" element={<SpanPage />} />
      </Routes>
    </MemoryRouter>
  )
}

function page() {
  render(tree())
}

const patch =
  "diff --git a/pay.go b/pay.go\n--- a/pay.go\n+++ b/pay.go\n@@ -1 +1,2 @@\n a\n+b\n"

beforeEach(() => {
  approveStage.mockReset()
  resumeRun.mockReset()
  pauseRun.mockReset()
  saveInstruction.mockReset()
  approveStage.mockResolvedValue(undefined)
})

afterEach(cleanup)

describe("a stage held by its mode", () => {
  it("offers the approval instead of a retry, and posts it for that stage", async () => {
    const earlier: Statement = {
      stage: "merge",
      iteration: 1,
      verdict: "hold",
      reason: "merge is hold",
      digest: "d",
      commands: [],
    }
    state = held({}, [earlier])
    page()
    expect(screen.queryByText("Fix and retry")).toBeNull()
    fireEvent.click(screen.getByRole("button", { name: "Approve merge" }))
    await waitFor(() =>
      expect(approveStage).toHaveBeenCalledWith("span-1", "merge", "")
    )
    expect(resumeRun).not.toHaveBeenCalled()
  })

  it("lists the checks that passed before open", () => {
    state = held({ stage: "open", reason: "open is hold" }, [
      {
        ...advance("prove"),
        commands: [{ command: "go test ./...", exitCode: 0, output: "" }],
      },
    ])
    page()
    expect(
      screen.getByText(
        /prove passed on this commit. The stage is held by its mode/
      )
    ).toBeTruthy()
    expect(
      screen.getByRole("heading", { name: "Checks that passed on this commit" })
    ).toBeTruthy()
    expect(screen.getByText("$ go test ./...")).toBeTruthy()
  })

  it("keeps the ordinary retry for a hold caused by a failure", () => {
    state = held({ stage: "merge", reason: "merge outcome unknown: network" })
    page()
    expect(screen.queryByRole("button", { name: "Approve merge" })).toBeNull()
  })
})

describe("the block that holds what the run waits on", () => {
  const decision = {
    id: "d1",
    class: "scope",
    stage: "frame",
    title: "Where should delivery attempts be recorded?",
    happened: "Two designs fit the story.",
    evidence: [],
    options: [
      {
        id: "a",
        label: "A table",
        consequence: "One migration.",
        reversible: true,
      },
    ],
    recommendation: "a",
  }

  it("holds the questions of a waiting run, above the tabs", () => {
    state = held({
      stage: "frame",
      reason: "open questions: 1",
      questions: ["What is the retry limit?"],
    })
    page()
    const block = screen.getByRole("region", { name: "Needs you" })
    expect(block.textContent).toContain("Waiting · 1 question")
    expect(block.textContent).toContain("What is the retry limit?")
    expect(
      screen.getByRole("button", { name: "Answer and continue" })
    ).toBeTruthy()
  })

  it("holds an open decision, and names the case in words", () => {
    state = held({ stage: "frame", reason: "decision needed", decision })
    page()
    const block = screen.getByRole("region", { name: "Needs you" })
    expect(block.textContent).toContain("Decision · 1 open")
    expect(block.textContent).toContain(
      "Where should delivery attempts be recorded?"
    )
    expect(block.textContent).toContain("recommended")
  })

  it("names the approval and keeps it as the action", () => {
    state = held({})
    page()
    expect(
      screen.getByRole("region", { name: "Needs you" }).textContent
    ).toContain("Approval needed")
  })

  it("draws no block while the run goes on by itself", () => {
    state = held({ hold: false, reason: "" })
    page()
    expect(screen.queryByRole("region", { name: "Needs you" })).toBeNull()
  })
})

describe("the side of the page", () => {
  it("keeps what is safe next to the controls, never inside a tab", () => {
    state = held({ stage: "build" }, [advance("intake"), advance("frame")])
    page()
    const kept = screen.getByRole("region", { name: "Safe and kept" })
    expect(kept.textContent).toContain("2 stages sealed")
    expect(kept.textContent).toContain("feat/x")
    expect(screen.getByRole("region", { name: "Run controls" })).toBeTruthy()
  })

  it("offers pause and stop only while the run is going on", () => {
    state = held({ hold: false, reason: "" })
    page()
    expect(
      screen.getByRole("button", { name: "Pause" })
    ).toBeTruthy()
    cleanup()
    state = held({})
    page()
    expect(
      screen.queryByRole("button", { name: "Pause" })
    ).toBeNull()
    expect(screen.queryByRole("button", { name: "Stop now" })).toBeNull()
  })

  it("says a stopped run will not continue inside the block that names it", () => {
    state = held({ hold: false, stop: true, reason: "stopped by a person" })
    page()
    const block = screen.getByRole("region", { name: "Needs you" })
    expect(block.textContent).toContain("Run stopped")
    expect(block.textContent).toContain("This run will not continue.")
  })
})

describe("the proof at the top of the page", () => {
  it("shows the proof once the diff is known", () => {
    state = {
      ...held({ stage: "prove" }),
      implementation: {
        stat: " pay.go | 1 +\n",
        patch,
        truncated: false,
        commits: 0,
      },
    }
    page()
    const proof = screen.getByRole("region", { name: "Proof of work" })
    expect(proof.textContent).toContain("1 file")
    expect(proof.textContent).toContain("+1")
  })

  it("says where the run stands before there is a diff to read", () => {
    state = held({ stage: "frame", questions: ["Which queue?"] })
    page()
    const text = screen.getByRole("region", {
      name: "Proof of work",
    }).textContent
    expect(text).toContain("Frame paused")
    expect(text).toContain("frame does not touch the repo")
    expect(text).toContain("prove comes after frame")
  })
})

describe("the order on a narrow screen", () => {
  it("puts what is safe before the tabs and the controls after them, as the design draws it", () => {
    state = held({ stage: "build" }, [advance("intake")])
    page()
    const classes = (name: string) =>
      screen.getByRole("region", { name }).className
    expect(classes("Safe and kept")).toContain("order-2")
    expect(
      screen.getByRole("tablist").closest("[data-slot='tabs']")?.className
    ).toContain("order-3")
    expect(classes("Run controls")).toContain("order-4")
    state = held({
      stage: "frame",
      reason: "open questions: 1",
      questions: ["What is the limit?"],
    })
    cleanup()
    page()
    expect(
      screen.getByRole("region", { name: "Needs you" }).parentElement?.className
    ).toContain("order-1")
  })
})

describe("the card while the run goes on by itself", () => {
  const working = (over: Partial<Cursor> = {}): SpanState => ({
    ...held({
      stage: "build",
      hold: false,
      reason: "",
      workspace: "/w/billing-api",
      ...over,
    }),
    transcript: ["editing retry.go"],
    agent: {
      id: "span-1",
      running: true,
      relaunch: false,
      name: "claude-code",
      pid: 7,
      stage: "build",
      elapsed: "2m",
      summary: "running",
    },
  })

  it("says nothing is needed, what the agent does now and what happens next", () => {
    state = working()
    page()
    const card = screen.getByRole("region", { name: "Running" })
    expect(card.textContent).toContain("Nothing needed from you")
    expect(card.textContent).toContain("Claude Code is working in build.")
    expect(card.textContent).toContain("editing retry.go")
    expect(card.textContent).toContain(
      "Build seals, then prove runs on its own."
    )
    expect(card.textContent).toContain("billing-api · feat/x")
  })

  it("carries the pause action, so it is offered once and not twice", () => {
    state = working()
    page()
    expect(
      screen.getAllByRole("button", { name: "Pause" })
    ).toHaveLength(1)
    expect(
      screen.getByRole("region", { name: "Running" }).querySelector("button")
        ?.textContent
    ).toBe("Pause")
    expect(screen.getByRole("button", { name: "Stop now" })).toBeTruthy()
    expect(
      screen.getByText(
        "Ends the run. Sealed stages and the branch stay as they are."
      )
    ).toBeTruthy()
  })

  it("shows a requested pause as pending, with the button spent", () => {
    state = working()
    if (state.span !== null) {
      state.span.pauseRequested = true
    }
    page()
    const card = screen.getByRole("region", { name: "Running" })
    const spent = within(card).getByRole("button", { name: "Pause requested" })
    expect(spent.hasAttribute("disabled")).toBe(true)
    expect(
      within(card).queryByRole("button", { name: "Pause" })
    ).toBeNull()
    expect(card.textContent).toContain("Pausing stops the agent. The run holds at build")
  })

  it("marks the pause pending as soon as the server accepts it", async () => {
    pauseRun.mockResolvedValue({
      paused: true,
      reason: "the run holds before the next stage",
    })
    state = working()
    page()
    fireEvent.click(
      screen.getByRole("button", { name: "Pause" })
    )
    expect(
      await screen.findByRole("button", { name: "Pause requested" })
    ).toBeTruthy()
  })

  it("keeps the pause button when nothing was running to pause", async () => {
    pauseRun.mockResolvedValue({
      paused: false,
      reason: "this span is not running",
    })
    state = working()
    page()
    fireEvent.click(
      screen.getByRole("button", { name: "Pause" })
    )
    await waitFor(() => expect(pauseRun).toHaveBeenCalled())
    expect(
      screen.queryByRole("button", { name: "Pause requested" })
    ).toBeNull()
  })

  it("is not drawn when the run waits on a person", () => {
    state = held({ stage: "build" })
    page()
    expect(screen.queryByRole("region", { name: "Running" })).toBeNull()
  })

  it("is not drawn once the run is done", () => {
    state = { ...working({ done: true }) }
    page()
    expect(screen.queryByRole("region", { name: "Running" })).toBeNull()
  })
})

describe("the card once the run is over", () => {
  const all = [
    "intake",
    "frame",
    "build",
    "prove",
    "qa",
    "inspect",
    "open",
    "merge",
    "promote",
    "note",
  ].map(advance)

  it("says where the work went and offers the change", () => {
    state = held(
      {
        done: true,
        hold: false,
        reason: "",
        gitSHA: "c91f3abdeadbeef",
        base: "main",
        issue: "SPAN-482",
        changeURL: "https://github.com/o/r/pull/8",
      },
      all
    )
    page()
    const card = screen.getByRole("region", { name: "Done" })
    expect(card.textContent).toContain("all 10 stages sealed")
    expect(card.textContent).toContain("Merged into main as c91f3ab")
    expect(card.textContent).toContain("SPAN-482")
    expect(
      screen.getByRole("link", { name: /Open PR #8/ }).getAttribute("href")
    ).toBe("https://github.com/o/r/pull/8")
  })

  it("does not claim a merge when the merge stage never sealed", () => {
    state = held(
      { done: true, hold: false, reason: "", gitSHA: "c91f3abdeadbeef" },
      all.slice(0, 3)
    )
    page()
    expect(screen.getByRole("region", { name: "Done" }).textContent).toContain(
      "Finished at c91f3ab"
    )
  })

  it("is not drawn while the run is still going", () => {
    state = held({ done: false })
    page()
    expect(screen.queryByRole("region", { name: "Done" })).toBeNull()
  })
})

describe("how long the run has waited", () => {
  const stamp = (minutesAgo: number) =>
    new Date(Date.now() - minutesAgo * 60_000).toISOString()

  it("says when it was held and how long ago", () => {
    state = held({ stage: "prove" }, [{ ...advance("build"), time: stamp(16) }])
    page()
    expect(screen.getByText(/^held \d\d:\d\d · 16 min ago$/)).toBeTruthy()
  })

  it("says nothing while the run goes on, or when the ledger has no time", () => {
    state = held({ hold: false, reason: "" }, [
      { ...advance("build"), time: stamp(5) },
    ])
    page()
    expect(screen.queryByText(/min ago/)).toBeNull()
    cleanup()
    state = held({ stage: "prove" }, [advance("build")])
    page()
    expect(screen.queryByText(/min ago/)).toBeNull()
  })
})

describe("a checkout another run holds", () => {
  const reason = (who: string) =>
    `another run is active in this checkout (${who})`

  it("links to the run that holds it", () => {
    state = held({ stage: "build", reason: reason("pid 4321 run span-7f3a") })
    page()
    expect(
      screen
        .getByRole("link", { name: "Open the other run" })
        .getAttribute("href")
    ).toBe("/spans/span-7f3a")
    expect(
      screen.getByText("Nothing yet. Build starts from the last sealed stage.")
    ).toBeTruthy()
    expect(
      screen.getByText("Build starts as soon as the checkout is free.")
    ).toBeTruthy()
  })

  it("offers no link when the holder is not a run", () => {
    state = held({ stage: "build", reason: reason("pid 4321 shape") })
    page()
    expect(
      screen.queryByRole("link", { name: "Open the other run" })
    ).toBeNull()
  })
})

describe("a note and a resume in one click", () => {
  const writeNote = () => {
    // The form opens from the note row, as the design draws it.
    fireEvent.click(
      screen.getAllByRole("button", {
        name: /^Leave a note for/,
      })[0] as HTMLElement
    )
    fireEvent.change(screen.getByLabelText("Instruction body"), {
      target: { value: "Keep the backoff at 2x." },
    })
  }

  it("saves the note, then resumes, in that order", async () => {
    state = held({ stage: "prove" })
    saveInstruction.mockResolvedValue(undefined)
    resumeRun.mockResolvedValue(undefined)
    page()
    writeNote()
    fireEvent.click(screen.getByRole("button", { name: "Save and resume" }))
    await waitFor(() => expect(resumeRun).toHaveBeenCalledOnce())
    expect(saveInstruction).toHaveBeenCalledWith(
      "span-1",
      "prove",
      "Keep the backoff at 2x."
    )
    expect(saveInstruction.mock.invocationCallOrder[0]).toBeLessThan(
      resumeRun.mock.invocationCallOrder[0] ?? 0
    )
  })

  it("does not resume when the note was not saved", async () => {
    state = held({ stage: "prove" })
    saveInstruction.mockRejectedValue(new Error("disk full"))
    page()
    writeNote()
    fireEvent.click(screen.getByRole("button", { name: "Save and resume" }))
    expect((await screen.findAllByText("disk full")).length).toBeGreaterThan(0)
    expect(resumeRun).not.toHaveBeenCalled()
  })

  it("offers only the plain save while the run goes on, since there is nothing to resume", () => {
    state = {
      ...held({ stage: "build", hold: false, reason: "" }),
      agent: {
        id: "span-1",
        running: true,
        relaunch: false,
        name: "claude-code",
        pid: 7,
        stage: "build",
        elapsed: "1m",
        summary: "running",
      },
    }
    page()
    fireEvent.click(
      screen.getAllByRole("button", {
        name: /^Leave a note for/,
      })[0] as HTMLElement
    )
    expect(
      screen.getByRole("button", { name: "Save instruction" })
    ).toBeTruthy()
    expect(screen.queryByRole("button", { name: "Save and resume" })).toBeNull()
  })
})

describe("what a finished run shows on the page", () => {
  const diff = { stat: " pay.go | 1 +\n", patch, truncated: false, commits: 1 }

  it("lists the files changed once the run is done", () => {
    state = {
      ...held({ done: true, hold: false, reason: "" }, [advance("merge")]),
      implementation: diff,
    }
    page()
    expect(
      screen.getByRole("region", { name: "Files changed" }).textContent
    ).toContain("pay.go")
  })

  it("switches to the Implementation tab from Open full diff", () => {
    state = {
      ...held({ done: true, hold: false, reason: "" }, [advance("merge")]),
      implementation: diff,
    }
    page()
    fireEvent.click(screen.getByText("Open full diff"))
    expect(
      screen
        .getByRole("tab", { name: "Implementation" })
        .getAttribute("aria-selected")
    ).toBe("true")
  })

  it("shows the commits of the branch above the diff", () => {
    state = {
      ...held({ done: true, hold: false, reason: "" }, [advance("merge")]),
      implementation: {
        ...diff,
        commitLog: [
          {
            sha: "abc1234",
            subject: "feat: retry with backoff",
            time: "14:31",
          },
        ],
      },
    }
    page()
    fireEvent.mouseDown(screen.getByRole("tab", { name: "Implementation" }))
    fireEvent.click(screen.getByRole("tab", { name: "Commits" }))
    expect(screen.getByText("feat: retry with backoff")).toBeTruthy()
  })

  it("does not list them while the run is going", () => {
    state = { ...held({ stage: "prove" }), implementation: diff }
    page()
    expect(screen.queryByRole("region", { name: "Files changed" })).toBeNull()
    expect(screen.queryByRole("region", { name: "Run details" })).toBeNull()
  })
})

describe("a finished run leaves its controls behind", () => {
  it("offers no restart, note or agent choice once the run is done", () => {
    state = held({ done: true, hold: false, reason: "" }, [advance("merge")])
    page()
    expect(screen.queryByRole("region", { name: "Run controls" })).toBeNull()
    expect(screen.getByRole("region", { name: "Safe and kept" })).toBeTruthy()
  })

  it("marks the last stage sealed instead of current", () => {
    state = held({ done: true, stage: "note", hold: false, reason: "" }, [
      advance("note"),
    ])
    page()
    expect(
      screen
        .getByRole("region", { name: "Stages" })
        .querySelector("[aria-current]")
    ).toBeNull()
  })

  it("says a held run with no diff is held at its stage instead of calling it low risk", () => {
    state = {
      ...held({ stage: "prove" }),
      implementation: { stat: "", patch: "", truncated: false, commits: 0 },
    }
    page()
    const proof = screen.getByRole("region", { name: "Proof of work" })
    expect(proof.textContent).toContain("Held at prove")
    expect(proof.textContent).toContain("No code yet")
    expect(proof.textContent).not.toContain("low risk")
  })
})

describe("a run that cannot be shown yet", () => {
  it("draws the skeleton while the run is read", () => {
    state = { ...held({}), span: null }
    page()
    expect(screen.getByRole("status", { name: "Loading run" })).toBeTruthy()
  })

  it("says the run is not on this desk", () => {
    state = { ...held({}), span: null, problem: "notfound" }
    page()
    expect(
      screen.getByRole("heading", { name: "Run span-1 is not on this desk" })
    ).toBeTruthy()
    expect(screen.queryByRole("status", { name: "Loading run" })).toBeNull()
  })

  it("says the server is unreachable", () => {
    state = { ...held({}), span: null, problem: "unreachable" }
    page()
    expect(
      screen.getByRole("button", { name: "Retry connection" })
    ).toBeTruthy()
  })

  it("says it is reconnecting once the connection has dropped, and keeps the page", () => {
    state = held({})
    const view = render(tree())
    state = { ...held({}), connected: false }
    view.rerender(tree())
    expect(screen.getByText(/Reconnecting to Span/)).toBeTruthy()
  })
})

describe("a finished run whose diff is gone", () => {
  it("says so in the Implementation tab and links to the change", () => {
    state = {
      ...held(
        {
          done: true,
          hold: false,
          gitSHA: "c91f3ab0000",
          base: "main",
          changeURL: "https://github.com/a/b/pull/8",
        },
        [advance("merge")]
      ),
      implementation: { stat: "", patch: "", truncated: false, commits: 0 },
    }
    page()
    fireEvent.mouseDown(screen.getByRole("tab", { name: "Implementation" }))
    fireEvent.click(screen.getByRole("tab", { name: "Implementation" }))
    expect(screen.getByRole("region", { name: "No diff to read" })).toBeTruthy()
    const card = screen.getByRole("region", { name: "No diff to read" })
    expect(
      within(card)
        .getByRole("link", { name: "Open PR #8" })
        .getAttribute("href")
    ).toBe("https://github.com/a/b/pull/8")
  })
})

describe("the ways out of a failed check", () => {
  const failed = () =>
    held({ stage: "prove", reason: "prove failed: 2 tests (exit 1)" }, [
      advance("build"),
      {
        stage: "prove",
        iteration: 1,
        verdict: "hold",
        reason: "prove failed: 2 tests (exit 1)",
        commands: [{ command: "go test ./...", exitCode: 1, output: "FAIL" }],
      },
    ])

  it("says what a note does and offers the restart at build as a danger action", () => {
    state = failed()
    page()
    expect(screen.getByText("prove · exit 1")).toBeTruthy()
    expect(
      screen.getByText(/Save an instruction for prove, then resume/)
    ).toBeTruthy()
    expect(
      screen.getByText(/The attempt so far stays on disk as its own iteration/)
    ).toBeTruthy()
    fireEvent.click(screen.getByRole("button", { name: "Restart from build…" }))
    expect(
      screen.getByRole("button", { name: "Start again at build" })
    ).toBeTruthy()
  })

  it("names the repair attempt when the statement records a cap", () => {
    state = failed()
    const prove = state.span?.statements.find((item) => item.stage === "prove")
    if (prove !== undefined) {
      prove.cap = 3
    }
    page()
    expect(screen.getByText("prove · attempt 1 of 3 · exit 1")).toBeTruthy()
  })

  it("keeps the restart row collapsed until it is asked for", () => {
    state = failed()
    page()
    expect(
      screen.queryByRole("button", { name: "Start again at build" })
    ).toBeNull()
    expect(
      screen.getByRole("button", { name: /^Restart from a stage…/ })
    ).toBeTruthy()
  })
})

describe("a finished run whose build was never sealed", () => {
  it("says so, and reads the proof as not sealed, only when the ledger shows it", () => {
    state = held({ done: true, hold: false, reason: "" }, [
      advance("intake"),
      advance("frame"),
    ])
    page()
    expect(
      screen.getByRole("region", { name: "Build never sealed" })
    ).toBeTruthy()
    expect(
      screen.getByRole("region", { name: "Proof of work" }).textContent
    ).toContain("Not sealed")
  })

  it("draws no such card when build was sealed", () => {
    state = held({ done: true, hold: false, reason: "" }, [
      advance("build"),
      advance("merge"),
    ])
    page()
    expect(
      screen.queryByRole("region", { name: "Build never sealed" })
    ).toBeNull()
  })
})

describe("the run's actions in the command palette", () => {
  const ids = () => getRunContext()?.commands.map((command) => command.id) ?? []

  it("offers resume, note and restart on a held run, and reading actions always", () => {
    state = held({ stage: "prove" })
    page()
    expect(ids()).toEqual(
      expect.arrayContaining([
        "resume",
        "note",
        "rewind",
        "transcript",
        "diff",
        "copy-summary",
      ])
    )
    expect(ids()).not.toContain("stop")
  })

  it("resumes the run when its command runs", async () => {
    state = held({ stage: "prove" })
    resumeRun.mockResolvedValue(undefined)
    page()
    getRunContext()
      ?.commands.find((command) => command.id === "resume")
      ?.run()
    await waitFor(() => expect(resumeRun).toHaveBeenCalledOnce())
  })

  it("offers resume and relaunch when the agent is blocked", () => {
    state = held({
      stage: "build",
      reason: "agent blocked",
      agentBlock: {
        kind: "quota",
        agent: "claude-code",
        detail: "usage limit reached",
      },
    })
    state = {
      ...state,
      agent: {
        id: "span-1",
        running: false,
        relaunch: true,
        name: "",
        pid: 0,
        stage: "build",
        elapsed: "",
        summary: "paused",
      },
    }
    page()
    expect(
      screen.getByRole("heading", { name: "claude-code is out of quota" })
    ).toBeTruthy()
    expect(screen.getByRole("button", { name: "Resume" })).toBeTruthy()
    expect(screen.getByRole("button", { name: "Relaunch" })).toBeTruthy()
    expect(
      screen.getByText(
        "Nothing in the code changes. Build starts again with the same instructions."
      )
    ).toBeTruthy()
    expect(
      screen.getByText("Build runs with the agent you choose in Run controls.")
    ).toBeTruthy()
    expect(
      screen.getAllByRole("button", { name: /Restart from a stage/ }).length
    ).toBeGreaterThan(0)
  })

  it("offers only the reading actions once the run is done", () => {
    state = held({ done: true, hold: false, reason: "" }, [advance("build")])
    page()
    expect(ids()).toEqual([
      "transcript",
      "diff",
      "copy-summary",
      "tab-ledger",
      "tab-implementation",
      "tab-qa",
    ])
  })

  it("switches the tab from the palette keys 1, 2 and 3", () => {
    state = held({ done: true, hold: false, reason: "" }, [advance("build")])
    page()
    act(() => {
      getRunContext()
        ?.commands.find((command) => command.id === "tab-qa")
        ?.run()
    })
    expect(
      screen.getByRole("tab", { name: "QA" }).getAttribute("aria-selected")
    ).toBe("true")
  })

  it("copies the summary as markdown", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined)
    vi.stubGlobal("navigator", { ...navigator, clipboard: { writeText } })
    state = held({ done: true, hold: false, reason: "" }, [advance("build")])
    page()
    getRunContext()
      ?.commands.find((command) => command.id === "copy-summary")
      ?.run()
    await waitFor(() => expect(writeText).toHaveBeenCalledOnce())
    expect(String(writeText.mock.calls[0]?.[0])).toContain("#")
    vi.unstubAllGlobals()
  })
})
