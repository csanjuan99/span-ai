import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react"
import { MemoryRouter, Route, Routes } from "react-router"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

import type { ShapePreview, ShapeSession as Session } from "@/lib/types"

const getShape = vi.fn()
const answerShape = vi.fn()
const acceptShapeContext = vi.fn()
const sealShape = vi.fn()
const previewShapeIssue = vi.fn()
const createShapeIssue = vi.fn()
const startRun = vi.fn()

vi.mock("@/lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/api")>()
  return {
    ApiError: actual.ApiError,
    getShape: (...args: unknown[]) => getShape(...args),
    answerShape: (...args: unknown[]) => answerShape(...args),
    acceptShapeContext: (...args: unknown[]) => acceptShapeContext(...args),
    sealShape: (...args: unknown[]) => sealShape(...args),
    previewShapeIssue: (...args: unknown[]) => previewShapeIssue(...args),
    createShapeIssue: (...args: unknown[]) => createShapeIssue(...args),
    startRun: (form: FormData) => startRun(form),
  }
})

import { ApiError } from "@/lib/api"
import { ShapeSession } from "@/pages/ShapeSession"

class NoResize {
  observe(): void {}
  unobserve(): void {}
  disconnect(): void {}
}
vi.stubGlobal("ResizeObserver", NoResize)

afterEach(cleanup)

function session(over: Partial<Session> = {}): Session {
  return {
    id: "sh-1",
    workspace: "/w",
    repo: "api",
    idea: "let people pay",
    status: "interviewing",
    rules: [
      {
        id: "R1",
        text: "Only admins refund",
        source: { kind: "code", ref: "internal/refund.go" },
      },
    ],
    examples: [
      {
        id: "E1",
        rule: "R1",
        text: "An agent tries to refund and is refused",
        source: { kind: "answer", ref: "Q1" },
      },
    ],
    questions: [
      {
        id: "Q1",
        text: "Who can refund?",
        blocks: "R1",
        assume: "Admins",
        status: "open",
        answer: "",
      },
      {
        id: "Q2",
        text: "Is there a limit?",
        blocks: "R2",
        assume: "No limit",
        status: "open",
        answer: "",
      },
    ],
    split: [],
    proposals: [],
    story: "",
    digest: "",
    jira: null,
    ...over,
  }
}

const preview: ShapePreview = {
  project: "PRJ",
  board: { id: 7, name: "PRJ Kanban" },
  backlogStatuses: ["Backlog"],
  summary: "Refund payments",
  description: "As an admin I refund",
}

function open() {
  render(
    <MemoryRouter initialEntries={["/shaping/sh-1?workspace=%2Fw"]}>
      <Routes>
        <Route path="/shaping/:id" element={<ShapeSession />} />
        <Route path="/spans/:id" element={<p>run page</p>} />
      </Routes>
    </MemoryRouter>
  )
}

beforeEach(() => {
  for (const mock of [
    getShape,
    answerShape,
    acceptShapeContext,
    sealShape,
    previewShapeIssue,
    createShapeIssue,
    startRun,
  ]) {
    mock.mockReset()
  }
  getShape.mockResolvedValue(session())
})

describe("example map and questions", () => {
  it("shows each fact with its source, examples under their rule, and the proposed assumption", async () => {
    open()
    expect(await screen.findByText("Only admins refund")).toBeTruthy()
    expect(screen.getByText("code internal/refund.go")).toBeTruthy()
    expect(
      screen.getByText("An agent tries to refund and is refused")
    ).toBeTruthy()
    expect(screen.getByText("answer Q1")).toBeTruthy()
    expect(screen.getByText("Assumption: Admins")).toBeTruthy()
    expect(screen.getByText("Blocks R2")).toBeTruthy()
  })

  it("sends the typed answer and the accepted assumption in one request", async () => {
    answerShape.mockResolvedValue(session({ status: "ready" }))
    open()
    fireEvent.change(await screen.findByLabelText("Answer Q1"), {
      target: { value: "Only admins" },
    })
    fireEvent.click(
      screen.getByRole("button", { name: "Accept assumption for Q2" })
    )
    fireEvent.click(screen.getByRole("button", { name: "Send answers" }))
    await waitFor(() => expect(answerShape).toHaveBeenCalledTimes(1))
    expect(answerShape).toHaveBeenCalledWith(
      "sh-1",
      "/w",
      [{ question: "Q1", text: "Only admins" }],
      ["Q2"]
    )
  })

  it("shows a pending state and blocks a second submit while the turn runs", async () => {
    answerShape.mockReturnValue(new Promise(() => undefined))
    open()
    fireEvent.click(
      await screen.findByRole("button", { name: "Accept assumption for Q1" })
    )
    fireEvent.click(screen.getByRole("button", { name: "Send answers" }))
    const pending = await screen.findByRole("button", { name: "Thinking…" })
    expect((pending as HTMLButtonElement).disabled).toBe(true)
    fireEvent.click(pending)
    expect(answerShape).toHaveBeenCalledTimes(1)
  })
})

describe("split", () => {
  it("lists the split, asks to narrow the story and offers no seal", async () => {
    getShape.mockResolvedValue(
      session({
        status: "split",
        questions: [],
        split: [{ story: "Refund by admins", leftOut: "Partial refunds" }],
      })
    )
    open()
    expect(await screen.findByText("Refund by admins")).toBeTruthy()
    expect(screen.getByText(/narrow the story/i)).toBeTruthy()
    expect(screen.getByText(/Partial refunds/)).toBeTruthy()
    expect(screen.queryByRole("button", { name: "Seal story" })).toBeNull()
  })
})

describe("context proposals", () => {
  it("sends only the ticked proposals", async () => {
    getShape.mockResolvedValue(
      session({
        proposals: [
          {
            id: "P1",
            section: "Glossary",
            text: "Refund: money back",
            status: "proposed",
          },
          { id: "P2", section: "Personas", text: "Admin", status: "proposed" },
        ],
      })
    )
    acceptShapeContext.mockResolvedValue(session())
    open()
    fireEvent.click(await screen.findByRole("checkbox", { name: /Admin/ }))
    fireEvent.click(
      screen.getByRole("button", { name: "Add to product context" })
    )
    await waitFor(() =>
      expect(acceptShapeContext).toHaveBeenCalledWith("sh-1", "/w", ["P2"])
    )
  })
})

describe("ready and seal", () => {
  const ready = () =>
    session({
      status: "ready",
      questions: [],
      story: "# Refund story\n\nAs an admin",
    })

  it("shows the story and seals it", async () => {
    getShape.mockResolvedValue(ready())
    sealShape.mockResolvedValue(
      session({
        status: "sealed",
        story: "# Refund story",
        digest: "abcdef0123456789ffff",
      })
    )
    open()
    expect(await screen.findByText(/# Refund story/)).toBeTruthy()
    fireEvent.click(screen.getByRole("button", { name: "Seal story" }))
    expect(await screen.findByText("abcdef012345")).toBeTruthy()
  })

  it("shows the open question ids when the seal is refused", async () => {
    getShape.mockResolvedValue(ready())
    sealShape.mockRejectedValue(
      new ApiError(
        "shape_not_ready",
        "story is not ready, open questions: Q3, Q4"
      )
    )
    open()
    fireEvent.click(await screen.findByRole("button", { name: "Seal story" }))
    expect(await screen.findByText(/open questions: Q3, Q4/)).toBeTruthy()
  })
})

describe("sealed story and Jira", () => {
  const sealed = (jira: Session["jira"] = null) =>
    session({
      status: "sealed",
      questions: [],
      story: "# Refund story",
      digest: "abcdef0123456789ffff",
      jira,
    })

  it("previews first and creates only after the confirm button", async () => {
    getShape.mockResolvedValue(sealed())
    previewShapeIssue.mockResolvedValue(preview)
    createShapeIssue.mockResolvedValue(
      sealed({ key: "PRJ-9", url: "https://j/PRJ-9", status: "Backlog" })
    )
    open()
    fireEvent.change(await screen.findByLabelText("Project key"), {
      target: { value: "PRJ" },
    })
    fireEvent.change(screen.getByLabelText("Board id (optional)"), {
      target: { value: "7" },
    })
    expect(screen.queryByRole("button", { name: "Create in Jira" })).toBeNull()
    fireEvent.click(screen.getByRole("button", { name: "Preview" }))
    expect(await screen.findByText("Refund payments")).toBeTruthy()
    expect(screen.getByText("PRJ Kanban")).toBeTruthy()
    expect(screen.getByText("Backlog")).toBeTruthy()
    expect(previewShapeIssue).toHaveBeenCalledWith("sh-1", "/w", "PRJ", "7")
    expect(createShapeIssue).not.toHaveBeenCalled()
    fireEvent.click(screen.getByRole("button", { name: "Create in Jira" }))
    const link = await screen.findByRole("link", { name: "PRJ-9" })
    expect(link.getAttribute("href")).toBe("https://j/PRJ-9")
    expect(createShapeIssue).toHaveBeenCalledWith("sh-1", "/w", "PRJ", "7")
  })

  it("names a disabled backlog", async () => {
    getShape.mockResolvedValue(sealed())
    previewShapeIssue.mockRejectedValue(
      new ApiError(
        "jira_backlog_disabled",
        "the Backlog column has no statuses"
      )
    )
    open()
    fireEvent.change(await screen.findByLabelText("Project key"), {
      target: { value: "PRJ" },
    })
    fireEvent.click(screen.getByRole("button", { name: "Preview" }))
    expect(await screen.findByText(/backlog is not enabled/i)).toBeTruthy()
  })

  it("shows the board list of an ambiguous project", async () => {
    getShape.mockResolvedValue(sealed())
    previewShapeIssue.mockRejectedValue(
      new ApiError("jira_board_ambiguous", "boards: 7 Alpha, 8 Beta")
    )
    open()
    fireEvent.change(await screen.findByLabelText("Project key"), {
      target: { value: "PRJ" },
    })
    fireEvent.click(screen.getByRole("button", { name: "Preview" }))
    expect(await screen.findByText(/boards: 7 Alpha, 8 Beta/)).toBeTruthy()
  })

  it("says the issue exists when it could not reach the backlog", async () => {
    getShape.mockResolvedValueOnce(sealed())
    previewShapeIssue.mockResolvedValue(preview)
    createShapeIssue.mockRejectedValue(
      new ApiError("jira_not_in_backlog", "PRJ-9 is in To Do")
    )
    getShape.mockResolvedValue(
      sealed({ key: "PRJ-9", url: "https://j/PRJ-9", status: "To Do" })
    )
    open()
    fireEvent.change(await screen.findByLabelText("Project key"), {
      target: { value: "PRJ" },
    })
    fireEvent.click(screen.getByRole("button", { name: "Preview" }))
    fireEvent.click(
      await screen.findByRole("button", { name: "Create in Jira" })
    )
    expect(await screen.findByText(/PRJ-9 is in To Do/)).toBeTruthy()
    expect(screen.getByText(/issue exists/i)).toBeTruthy()
    expect(screen.queryByRole("button", { name: "Create in Jira" })).toBeNull()
  })

  it("never offers create again once the session has a key", async () => {
    getShape.mockResolvedValue(
      sealed({ key: "PRJ-9", url: "https://j/PRJ-9", status: "Backlog" })
    )
    open()
    expect(await screen.findByRole("link", { name: "PRJ-9" })).toBeTruthy()
    expect(screen.queryByLabelText("Project key")).toBeNull()
    expect(screen.queryByRole("button", { name: "Preview" })).toBeNull()
    expect(screen.queryByRole("button", { name: "Create in Jira" })).toBeNull()
  })

  it("starts a run with the story and the Jira key", async () => {
    getShape.mockResolvedValue(
      sealed({ key: "PRJ-9", url: "https://j/PRJ-9", status: "Backlog" })
    )
    startRun.mockResolvedValue({
      id: "run-1",
      stage: "intake",
      runs: [{ id: "run-1", stage: "intake", repo: "api" }],
      partial: false,
      notes: "",
    })
    open()
    fireEvent.click(await screen.findByRole("button", { name: "Start run" }))
    expect(await screen.findByText("run page")).toBeTruthy()
    const form = startRun.mock.calls[0]?.[0] as FormData
    expect(form.get("story")).toBe("# Refund story")
    expect(form.get("issue")).toBe("PRJ-9")
    expect(form.get("workspace")).toBe("/w")
    expect(form.get("repo")).toBe("api")
  })

  it("starts a run without an issue when Jira was skipped", async () => {
    getShape.mockResolvedValue(sealed())
    startRun.mockResolvedValue({
      id: "run-1",
      stage: "intake",
      runs: [{ id: "run-1", stage: "intake", repo: "" }],
      partial: false,
      notes: "",
    })
    open()
    fireEvent.click(await screen.findByRole("button", { name: "Start run" }))
    await screen.findByText("run page")
    expect((startRun.mock.calls[0]?.[0] as FormData).has("issue")).toBe(false)
  })
})
