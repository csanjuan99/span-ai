import { cleanup, render, screen } from "@testing-library/react"
import { MemoryRouter, Route, Routes } from "react-router"
import { afterEach, describe, expect, it, vi } from "vitest"

import type { SpanState } from "@/hooks/use-span"
import type { Conflict, Cursor, Snapshot } from "@/lib/types"

let state: SpanState

vi.mock("@/hooks/use-span", () => ({
  useSpan: () => state,
}))

import { SpanPage } from "@/pages/SpanPage"

// The stage selects measure themselves; jsdom has no ResizeObserver.
class NoResize {
  observe(): void {}
  unobserve(): void {}
  disconnect(): void {}
}
vi.stubGlobal("ResizeObserver", NoResize)

afterEach(cleanup)

const choosing: Conflict = {
  base: "main",
  baseSHA: "abc",
  state: "choosing",
  note: "",
  groups: [
    {
      id: "g1",
      kind: "content",
      files: ["src/cart.ts"],
      summary: "Both sides changed the total.",
      chosen: "",
      options: [
        {
          id: "a",
          title: "Keep the branch",
          summary: "",
          risk: "low",
          action: "ours",
          command: "",
          diff: "",
        },
        {
          id: "b",
          title: "Take the base",
          summary: "",
          risk: "medium",
          action: "theirs",
          command: "",
          diff: "",
        },
      ],
    },
  ],
}

function cursor(over: Partial<Cursor>): Cursor {
  return {
    id: "s",
    stage: "merge",
    iteration: 1,
    hold: true,
    stop: false,
    done: false,
    reason: "",
    gitSHA: "",
    changeURL: "",
    branch: "feature",
    base: "main",
    workspace: "",
    issue: "",
    questions: [],
    ...over,
  }
}

function held(over: Partial<Cursor>): SpanState {
  const span: Snapshot = {
    cursor: cursor(over),
    statements: [
      {
        stage: "merge",
        iteration: 1,
        verdict: "hold",
        reason: "the change has conflicts",
        commands: [],
      },
    ],
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

function renderPage() {
  render(
    <MemoryRouter initialEntries={["/spans/s"]}>
      <Routes>
        <Route path="/spans/:id" element={<SpanPage />} />
      </Routes>
    </MemoryRouter>
  )
}

describe("SpanPage on a merge conflict", () => {
  it("shows the conflict panel and not the generic failure card", () => {
    state = held({
      reason: "conflict: choose how to resolve",
      conflict: choosing,
    })
    renderPage()
    expect(screen.getByRole("region", { name: /hits a conflict/ })).toBeTruthy()
    expect(screen.queryByRole("region", { name: "Failing check" })).toBeNull()
    expect(screen.queryByRole("button", { name: "Fix and retry" })).toBeNull()
    expect(screen.queryByRole("button", { name: "Resume" })).toBeNull()
  })

  it("shows a neutral note and the normal resume after an abort", () => {
    state = held({ reason: "conflict: aborted" })
    renderPage()
    expect(screen.queryByRole("region", { name: /hits a conflict/ })).toBeNull()
    expect(screen.queryByRole("region", { name: "Failing check" })).toBeNull()
    expect(screen.getByRole("button", { name: "Resume" })).toBeTruthy()
    expect(screen.getAllByText(/merge was aborted/).length).toBeGreaterThan(0)
  })

  it("keeps the failure card for an ordinary failed check", () => {
    state = held({ reason: "the change has conflicts" })
    renderPage()
    expect(screen.getByRole("region", { name: "Failing check" })).toBeTruthy()
  })
})
