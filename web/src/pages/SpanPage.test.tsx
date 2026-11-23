import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react"
import { MemoryRouter, Route, Routes } from "react-router"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

import type { SpanState } from "@/hooks/use-span"
import type { Cursor, Snapshot, Statement } from "@/lib/types"

const approveStage = vi.fn()
const resumeRun = vi.fn()
let state: SpanState

vi.mock("@/hooks/use-span", () => ({ useSpan: () => state }))
vi.mock("@/lib/api", async (original) => {
  const real = await original<typeof import("@/lib/api")>()
  return {
    ...real,
    approveStage: (id: string, stage: string, environment: string) => approveStage(id, stage, environment),
    resumeRun: (id: string) => resumeRun(id),
  }
})

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

function held(over: Partial<Cursor>, statements: Statement[] = []): SpanState {
  const span: Snapshot = { cursor: cursor(over), statements, acceptance: [], gate: null }
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
  }
}

function page() {
  render(
    <MemoryRouter initialEntries={["/spans/span-1"]}>
      <Routes>
        <Route path="/spans/:id" element={<SpanPage />} />
      </Routes>
    </MemoryRouter>,
  )
}

beforeEach(() => {
  approveStage.mockReset()
  resumeRun.mockReset()
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
    await waitFor(() => expect(approveStage).toHaveBeenCalledWith("span-1", "merge", ""))
    expect(resumeRun).not.toHaveBeenCalled()
  })

  it("keeps the ordinary retry for a hold caused by a failure", () => {
    state = held({ stage: "merge", reason: "merge outcome unknown: network" })
    page()
    expect(screen.queryByRole("button", { name: "Approve merge" })).toBeNull()
  })
})
