import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react"
import { MemoryRouter, Route, Routes } from "react-router"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

import type {
  BoardView,
  BulkResult,
  HistoryView,
  UndoPlan,
} from "@/lib/shape-flows"

const flows = vi.hoisted(() => ({
  getBoard: vi.fn(),
  getActivity: vi.fn(),
  bulk: vi.fn(),
  getHistory: vi.fn(),
  getUndoPlan: vi.fn(),
  undoStep: vi.fn(),
  readJira: vi.fn(),
  readMarkdown: vi.fn(),
  importStory: vi.fn(),
  exportText: vi.fn(),
  listShares: vi.fn(),
  createShare: vi.fn(),
  revokeShare: vi.fn(),
  openShared: vi.fn(),
}))
const getJSON = vi.fn()
const getShape = vi.fn()

vi.mock("@/lib/shape-flows", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/shape-flows")>()
  return { ...actual, ...flows }
})
vi.mock("@/lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/api")>()
  return {
    ...actual,
    getJSON: (url: string) => getJSON(url),
    getShape: (...args: unknown[]) => getShape(...args),
  }
})

import { ShapeBoard } from "@/pages/ShapeBoard"
import { ShapeHistory } from "@/pages/ShapeHistory"
import { ShapeShared } from "@/pages/ShapeShared"
import { ShapeTransfer } from "@/pages/ShapeTransfer"

afterEach(cleanup)
beforeEach(() => {
  for (const fn of Object.values(flows)) {
    fn.mockReset()
  }
  getJSON.mockReset()
  getJSON.mockResolvedValue({})
  getShape.mockReset()
  flows.getActivity.mockResolvedValue([])
  flows.listShares.mockResolvedValue([])
})

function at(path: string, pattern: string, page: React.ReactNode) {
  render(
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route path={pattern} element={page} />
      </Routes>
    </MemoryRouter>
  )
}

const card = {
  id: "a1",
  ref: "SES-0142",
  title: "Customers retry failed webhooks",
  status: "interviewing",
  repo: "api",
  agent: "codex",
  model: "",
  openQuestions: 2,
  assumed: 0,
  stories: 0,
  jiraKey: "",
  sealedAt: "",
  doing: "asked you 1 question (Q1)",
}
const board: BoardView = {
  inFlight: 1,
  needYou: 2,
  repos: ["api"],
  columns: [
    {
      key: "interviewing",
      name: "Interviewing",
      count: 1,
      hidden: 0,
      cards: [card],
    },
    { key: "ready", name: "Ready", count: 0, hidden: 0, cards: [] },
    { key: "split", name: "Split", count: 0, hidden: 0, cards: [] },
    { key: "sealed", name: "Sealed", count: 0, hidden: 0, cards: [] },
  ],
}

describe("sessions board", () => {
  it("plans a bulk action before it changes anything, then applies it", async () => {
    flows.getBoard.mockResolvedValue(board)
    const plan: BulkResult = {
      action: "assume",
      dryRun: true,
      done: 1,
      skipped: 0,
      failed: 0,
      items: [
        {
          id: "a1",
          ref: "SES-0142",
          ok: true,
          skipped: false,
          changed: ["Q1 assumed: everyone"],
          code: "",
          message: "",
        },
      ],
    }
    flows.bulk
      .mockResolvedValueOnce(plan)
      .mockResolvedValueOnce({ ...plan, dryRun: false })
    at("/shaping/board", "/shaping/board", <ShapeBoard />)
    fireEvent.click(await screen.findByLabelText("Select SES-0142"))
    expect(screen.getByText("1 selected")).toBeTruthy()
    fireEvent.click(screen.getByText("Answer 1 session with the assumption"))
    expect(await screen.findByText(/Q1 assumed: everyone/)).toBeTruthy()
    expect(flows.bulk).toHaveBeenCalledWith(".", "assume", ["a1"], true)
    flows.getBoard.mockResolvedValue({ ...board, needYou: 0 })
    fireEvent.click(
      screen
        .getAllByText("Answer 1 session with the assumption")
        .pop() as HTMLElement
    )
    await waitFor(() =>
      expect(flows.bulk).toHaveBeenLastCalledWith(".", "assume", ["a1"], false)
    )
  })

  it("offers seal only when every chosen session is ready", async () => {
    flows.getBoard.mockResolvedValue(board)
    at("/shaping/board", "/shaping/board", <ShapeBoard />)
    fireEvent.click(await screen.findByLabelText("Select SES-0142"))
    expect(screen.queryByText(/^Seal /)).toBeNull()
  })
})

const history: HistoryView = {
  id: "a1",
  ref: "SES-0142",
  title: "Retry webhooks",
  status: "ready",
  sealed: false,
  counts: { everyone: 2, you: 1, agents: 1 },
  events: [
    {
      id: "h2",
      at: "2026-11-25T09:33:00Z",
      actor: "you",
      kind: "answer",
      summary: "Answered Q2",
      ref: "Question 2",
      undoable: true,
      why: "",
      dependents: ["h3"],
      undoneBy: "",
    },
    {
      id: "h1",
      at: "2026-11-25T09:18:00Z",
      actor: "you",
      kind: "idea",
      summary: "Saved the idea",
      ref: "Session",
      undoable: false,
      why: "this is where the session starts",
      dependents: [],
      undoneBy: "",
    },
  ],
}

describe("history", () => {
  beforeEach(() => {
    getShape.mockResolvedValue({
      id: "a1",
      ref: "SES-0137",
      workspace: ".",
      repo: "acme/billing-api",
      idea: "Show tax per line",
      status: "ready",
      agent: "claude-code",
      model: "",
      startedAt: "2026-11-23T09:18:00Z",
    })
  })

  it("carries the session header with its actions and meta line", async () => {
    flows.getHistory.mockResolvedValue(history)
    at("/shaping/a1/history", "/shaping/:id/history", <ShapeHistory />)
    expect(await screen.findByText("Show tax per line")).toBeTruthy()
    for (const label of ["History", "Compare", "Export"]) {
      expect(screen.getByRole("link", { name: label })).toBeTruthy()
    }
    expect(screen.getByText("acme/billing-api")).toBeTruthy()
    expect(screen.getByText("claude-code · Auto")).toBeTruthy()
  })

  it("shows what an undo changes and undoes it with the steps that depend on it", async () => {
    flows.getHistory.mockResolvedValue(history)
    const plan: UndoPlan = {
      target: history.events[0],
      dependents: [{ ...history.events[1], id: "h3", summary: "Example E6" }],
      now: "Answered by you",
      after: "Open again",
      changes: ["Question 2 opens"],
      kept: "Your other answers",
      next: "Answer it again",
      sealed: false,
      undoOnlyThis: false,
    }
    flows.getUndoPlan.mockResolvedValue(plan)
    flows.undoStep.mockResolvedValue(["h2", "h3"])
    at("/shaping/a1/history", "/shaping/:id/history", <ShapeHistory />)
    fireEvent.click(await screen.findByLabelText("Undo h2"))
    expect(await screen.findByText("Question 2 opens")).toBeTruthy()
    fireEvent.click(screen.getByText("Undo h2 and 1 that depend on it"))
    await waitFor(() =>
      expect(flows.undoStep).toHaveBeenCalledWith("a1", ".", "h2", true)
    )
  })

  it("filters by who and locks a sealed story", async () => {
    flows.getHistory.mockResolvedValue({ ...history, sealed: true })
    at("/shaping/a1/history", "/shaping/:id/history", <ShapeHistory />)
    expect(await screen.findByText(/This story is sealed/)).toBeTruthy()
    expect(screen.queryByLabelText("Undo h2")).toBeNull()
    fireEvent.click(screen.getByText("You"))
    await waitFor(() =>
      expect(flows.getHistory).toHaveBeenLastCalledWith("a1", ".", "you", "")
    )
  })
})

describe("transfer", () => {
  it("reads a Jira story before it starts a session", async () => {
    flows.readJira.mockResolvedValue({
      summary: "Tax per line",
      status: "Done",
      paragraphs: 2,
      criteria: 3,
      comments: "not read",
    })
    flows.importStory.mockResolvedValue("new1")
    at("/shaping/import", "/shaping/import", <ShapeTransfer />)
    fireEvent.change(screen.getByLabelText("Jira key"), {
      target: { value: "BILL-2031" },
    })
    fireEvent.click(screen.getByText("See what Span reads"))
    expect(await screen.findByText("Tax per line")).toBeTruthy()
    expect(flows.importStory).not.toHaveBeenCalled()
    fireEvent.click(screen.getByText("Start session"))
    await waitFor(() =>
      expect(flows.importStory).toHaveBeenCalledWith({
        workspace: ".",
        repo: "",
        jira: "BILL-2031",
      })
    )
  })

  it("keeps export and share for a sealed story", async () => {
    getShape.mockResolvedValue({
      id: "a1",
      ref: "SES-0137",
      status: "interviewing",
    })
    at("/shaping/a1/transfer", "/shaping/:id/transfer", <ShapeTransfer />)
    expect(
      await screen.findByText(/Only a sealed story can be exported/)
    ).toBeTruthy()
    expect(screen.getByText(/Only a sealed story can be shared/)).toBeTruthy()
  })

  it("exports and shares a sealed story", async () => {
    getShape.mockResolvedValue({ id: "a1", ref: "SES-0137", status: "sealed" })
    flows.exportText.mockResolvedValue("# Tax per line")
    flows.listShares.mockResolvedValue([
      {
        token: "abcdefgh1234",
        createdAt: "",
        expiresAt: "2026-12-01T00:00:00Z",
        opens: 2,
        revoked: false,
      },
    ])
    flows.revokeShare.mockResolvedValue(undefined)
    at("/shaping/a1/transfer", "/shaping/:id/transfer", <ShapeTransfer />)
    expect(await screen.findByText("# Tax per line")).toBeTruthy()
    expect(flows.exportText).toHaveBeenCalledWith("a1", ".", {
      format: "md",
      sources: false,
      assumptions: false,
      digest: true,
    })
    fireEvent.click(await screen.findByLabelText("Revoke abcdefgh"))
    await waitFor(() =>
      expect(flows.revokeShare).toHaveBeenCalledWith("a1", ".", "abcdefgh1234")
    )
  })
})

describe("shared link", () => {
  it("shows the story read-only, and says why a link does not open", async () => {
    flows.openShared.mockResolvedValueOnce({
      ref: "SES-0137",
      title: "Tax per line",
      repo: "api",
      story: "As a user",
      digest: "abc",
      sealedAt: "",
      rules: [{ id: "R1", text: "Tax shows" }],
      examples: [{ id: "E1", rule: "R1", text: "100 -> 20" }],
    })
    at("/shaping/a1/shared/t1", "/shaping/:id/shared/:token", <ShapeShared />)
    expect(await screen.findByText("Tax per line")).toBeTruthy()
    expect(screen.getByText("100 -> 20")).toBeTruthy()
    expect(screen.getByText(/Read-only link/)).toBeTruthy()
    cleanup()
    flows.openShared.mockRejectedValueOnce(new Error("this link was revoked"))
    at("/shaping/a1/shared/t2", "/shaping/:id/shared/:token", <ShapeShared />)
    expect(await screen.findByText("this link was revoked")).toBeTruthy()
  })
})
