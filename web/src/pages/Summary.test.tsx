import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react"
import { MemoryRouter, Route, Routes } from "react-router"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

import type { SpanState } from "@/hooks/use-span"
import type { Cursor, Statement } from "@/lib/types"

let state: SpanState
vi.mock("@/hooks/use-span", () => ({ useSpan: () => state }))

import { Summary } from "@/pages/Summary"

const cursor: Cursor = {
  id: "run-1",
  stage: "note",
  iteration: 1,
  hold: false,
  stop: false,
  done: true,
  reason: "",
  gitSHA: "c91f3ab0000",
  changeURL: "https://github.com/o/r/pull/8",
  branch: "feat/x",
  base: "main",
  workspace: "/w/api",
  issue: "SPAN-1",
  questions: [],
}
const merge: Statement = {
  stage: "merge",
  iteration: 1,
  verdict: "advance",
  reason: "",
  commands: [],
}
const patch =
  "diff --git a/a.go b/a.go\n--- a/a.go\n+++ b/a.go\n@@ -1 +1 @@\n-x\n+y\n"

function loaded(over: Partial<SpanState> = {}): SpanState {
  return {
    span: {
      cursor,
      statements: [merge],
      acceptance: [],
      gate: null,
      label: "SPAN-1-run-1-api",
    },
    acceptance: null,
    implementation: {
      stat: " a.go | 2 +-\n 1 file changed, 1 insertion(+), 1 deletion(-)\n",
      patch,
      truncated: false,
      commits: 1,
    },
    transcript: [],
    frames: [],
    actions: [],
    agent: null,
    agentAt: "",
    note: () => undefined,
    error: "",
    connected: true,
    problem: null,
    ...over,
  }
}

function page() {
  render(
    <MemoryRouter initialEntries={["/spans/run-1/summary"]}>
      <Routes>
        <Route path="/spans/:id/summary" element={<Summary />} />
      </Routes>
    </MemoryRouter>
  )
}

const writeText = vi.fn()

beforeEach(() => {
  writeText.mockReset()
  Object.defineProperty(navigator, "clipboard", {
    value: { writeText },
    configurable: true,
  })
})
afterEach(cleanup)

describe("Summary", () => {
  it("shows what landed, the files and the link", () => {
    state = loaded()
    page()
    expect(screen.getByText("SPAN-1-run-1-api")).toBeTruthy()
    expect(screen.getByText("Merged into main as c91f3ab")).toBeTruthy()
    expect(screen.getByText("a.go")).toBeTruthy()
    expect(screen.getByRole("link", { name: "PR #8" })).toBeTruthy()
  })

  it("copies the summary as markdown", async () => {
    writeText.mockResolvedValue(undefined)
    state = loaded()
    page()
    fireEvent.click(screen.getByRole("button", { name: "Copy as markdown" }))
    await waitFor(() => expect(writeText).toHaveBeenCalledTimes(1))
    expect(String(writeText.mock.calls[0][0])).toContain("# SPAN-1-run-1-api")
    expect(await screen.findByText("Copied as markdown.")).toBeTruthy()
  })

  it("says so when the clipboard refuses", async () => {
    writeText.mockRejectedValue(new Error("denied"))
    state = loaded()
    page()
    fireEvent.click(screen.getByRole("button", { name: "Copy as markdown" }))
    expect(await screen.findByText(/The clipboard refused/)).toBeTruthy()
  })

  it("says it is loading, and that an unreadable run has no summary", () => {
    state = loaded({ span: null })
    page()
    expect(screen.getByText("Loading the summary…")).toBeTruthy()
    cleanup()
    state = loaded({ span: null, problem: "notfound" })
    page()
    expect(screen.getByText(/no summary/)).toBeTruthy()
  })

  it("warns when the run is not finished", () => {
    const open = loaded()
    open.span = { ...open.span!, cursor: { ...cursor, done: false } }
    state = open
    page()
    expect(screen.getByText(/not finished/)).toBeTruthy()
  })
})
