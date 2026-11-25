import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react"
import { MemoryRouter, Route, Routes, useLocation } from "react-router"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

import type {
  ShapePreview,
  ShapeQuestion,
  ShapeSession as Session,
} from "@/lib/types"

const mocks = {
  getShape: vi.fn(),
  answerShape: vi.fn(),
  settleShapeContext: vi.fn(),
  previewShapeContext: vi.fn(),
  decideShapeSplit: vi.fn(),
  forkShape: vi.fn(),
  getShapeSource: vi.fn(),
  resumeShape: vi.fn(),
  sealShape: vi.fn(),
  previewShapeIssue: vi.fn(),
  createShapeIssue: vi.fn(),
  startRun: vi.fn(),
}

vi.mock("@/lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/api")>()
  return {
    ...actual,
    getJSON: () => Promise.reject(new Error("no agents here")),
    getShape: (...args: unknown[]) => mocks.getShape(...args),
    answerShape: (...args: unknown[]) => mocks.answerShape(...args),
    settleShapeContext: (...args: unknown[]) =>
      mocks.settleShapeContext(...args),
    previewShapeContext: (...args: unknown[]) =>
      mocks.previewShapeContext(...args),
    decideShapeSplit: (...args: unknown[]) => mocks.decideShapeSplit(...args),
    forkShape: (...args: unknown[]) => mocks.forkShape(...args),
    getShapeSource: (...args: unknown[]) => mocks.getShapeSource(...args),
    resumeShape: (...args: unknown[]) => mocks.resumeShape(...args),
    sealShape: (...args: unknown[]) => mocks.sealShape(...args),
    previewShapeIssue: (...args: unknown[]) => mocks.previewShapeIssue(...args),
    createShapeIssue: (...args: unknown[]) => mocks.createShapeIssue(...args),
    startRun: (form: FormData) => mocks.startRun(form),
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

const question = (
  id: string,
  status: ShapeQuestion["status"],
  over: Partial<ShapeQuestion> = {}
): ShapeQuestion => ({
  id,
  text: `Question ${id}?`,
  blocks: "R1",
  assume: `assume ${id}`,
  status,
  answer: status === "open" ? "" : `answer ${id}`,
  answerId: "",
  ...over,
})

function session(over: Partial<Session> = {}): Session {
  return {
    id: "sh-1",
    workspace: "/w",
    repo: "api",
    idea: "let people refund",
    status: "interviewing",
    rules: [
      {
        id: "R1",
        text: "Only admins refund",
        source: { kind: "code", ref: "internal/refund.go:12" },
      },
      {
        id: "R2",
        text: "A refund is logged",
        source: { kind: "prompt", ref: "" },
      },
    ],
    examples: [
      {
        id: "E1",
        rule: "R1",
        text: "An agent is refused",
        source: { kind: "answer", ref: "Q1" },
      },
      {
        id: "E2",
        rule: "R2",
        text: "The log has the amount",
        source: { kind: "prompt", ref: "" },
      },
    ],
    questions: [question("Q1", "open"), question("Q2", "open")],
    split: [],
    proposals: [],
    story: "",
    digest: "",
    jira: null,
    ref: "SES-AB12CD",
    agent: "claude-code",
    model: "",
    ...over,
  }
}

const ready = (over: Partial<Session> = {}) =>
  session({
    status: "ready",
    story: "# Refund\n\n## Why\nBecause.",
    questions: [
      question("Q1", "answered", { answerId: "A1" }),
      question("Q2", "assumed", { answerId: "A2" }),
    ],
    ...over,
  })

const sealed = (over: Partial<Session> = {}) =>
  ready({
    status: "sealed",
    digest: "abcdef0123456789ffff",
    sealedAt: "2026-11-23T16:20:00Z",
    ...over,
  })

const preview: ShapePreview = {
  project: "PRJ",
  board: { id: 7, name: "PRJ Kanban" },
  backlogStatuses: ["Backlog", "Ready for grooming"],
  summary: "Refund payments",
  description: "As an admin I refund",
}

function open() {
  render(
    <MemoryRouter initialEntries={["/shaping/sh-1?workspace=%2Fw"]}>
      <Routes>
        <Route path="/shaping/:id" element={<ShapeSession />} />
        <Route path="/" element={<ArrivalProbe />} />
      </Routes>
    </MemoryRouter>
  )
}

beforeEach(() => {
  localStorage.clear()
  for (const mock of Object.values(mocks)) {
    mock.mockReset()
  }
  mocks.getShape.mockResolvedValue(session())
})

/** Stands in for the desk: prints what the session handed over to start a run from. */
function ArrivalProbe() {
  const state = useLocation().state as { shape?: Record<string, string> } | null
  return <pre data-testid="arrival">{JSON.stringify(state?.shape ?? null)}</pre>
}

const click = (name: string | RegExp) =>
  fireEvent.click(screen.getByRole("button", { name }))

const clickWhenReady = async (name: string | RegExp) =>
  fireEvent.click(await screen.findByRole("button", { name }))

describe("header and steps", () => {
  it("names the session, how it stands and where the person is", async () => {
    open()
    expect(
      await screen.findByRole("heading", { name: "let people refund" })
    ).toBeTruthy()
    expect(screen.getByText("SES-AB12CD")).toBeTruthy()
    expect(screen.getByText("Interviewing")).toBeTruthy()
    expect(screen.getByText("claude-code · Auto")).toBeTruthy()
    const current = screen
      .getAllByRole("listitem")
      .find((item) => item.getAttribute("aria-current") === "step")
    expect(current?.textContent).toContain("Questions")
  })
})

describe("questions", () => {
  it("keeps the answer buttons 44px tall on a phone and 40px on a desk", async () => {
    open()
    await screen.findByText("2 questions block the story")
    const answer = screen.getAllByRole("button", { name: /^Answer Q/ })[0]
    expect(answer.className).toContain("h-11")
    expect(answer.className).toContain("md:h-10")
  })

  it("sends every choice in one request, after each question is settled", async () => {
    mocks.answerShape.mockResolvedValue(ready())
    open()
    await screen.findByText("2 questions block the story")
    const next = screen.getByRole("button", { name: "Continue to example map" })
    expect((next as HTMLButtonElement).disabled).toBe(true)
    expect(
      screen.getByText("Settle 2 more questions to continue.")
    ).toBeTruthy()
    click("Use the assumption for Q1")
    click("Answer Q2")
    fireEvent.change(screen.getByLabelText("Your answer to Q2"), {
      target: { value: "  3 per hour  " },
    })
    expect((next as HTMLButtonElement).disabled).toBe(true)
    click("Save answer for Q2")
    expect((next as HTMLButtonElement).disabled).toBe(false)
    fireEvent.click(next)
    await waitFor(() =>
      expect(mocks.answerShape).toHaveBeenCalledWith(
        "sh-1",
        "/w",
        [{ question: "Q2", text: "3 per hour" }],
        ["Q1"]
      )
    )
  })

  it("lets a staged choice be changed before it is sent", async () => {
    open()
    await screen.findByText("2 questions block the story")
    click("Use the assumption for Q1")
    click("Change Q1")
    expect(
      screen.getByRole("button", { name: "Use the assumption for Q1" })
    ).toBeTruthy()
  })

  it("keeps staged choices when the page is reloaded", async () => {
    open()
    await screen.findByText("2 questions block the story")
    click("Use the assumption for Q1")
    cleanup()
    open()
    await screen.findByText("2 questions block the story")
    expect(screen.getByRole("button", { name: "Change Q1" })).toBeTruthy()
    expect(
      screen.getByRole("button", { name: "Use the assumption for Q2" })
    ).toBeTruthy()
  })

  it("shows the working state while the agent runs, then the result", async () => {
    let finish: (value: Session) => void = () => {}
    mocks.answerShape.mockReturnValue(
      new Promise<Session>((resolve) => (finish = resolve))
    )
    open()
    await screen.findByText("2 questions block the story")
    click("Use the assumption for Q1")
    click("Use the assumption for Q2")
    click("Continue to example map")
    expect(
      await screen.findByText("The agent is reading and drafting")
    ).toBeTruthy()
    finish(ready())
    expect(await screen.findByText("Example map")).toBeTruthy()
  })

  it("settled questions show who settled them, and an assumed one can be answered instead", async () => {
    mocks.getShape.mockResolvedValue(
      session({
        questions: [
          question("Q1", "answered", { answerId: "A1" }),
          question("Q2", "assumed", { answerId: "A2" }),
          question("Q3", "open"),
        ],
      })
    )
    mocks.answerShape.mockResolvedValue(ready())
    open()
    await screen.findByText("1 question blocks the story")
    expect(screen.getByText("Answered by you")).toBeTruthy()
    expect(screen.getByText("Your answer · A1")).toBeTruthy()
    expect(screen.getByText("The agent assumed")).toBeTruthy()
    click("Use the assumption for Q3")
    click("Answer Q2 instead")
    fireEvent.change(screen.getByLabelText("Your answer to Q2"), {
      target: { value: "admins" },
    })
    click("Save answer for Q2")
    click("Continue to example map")
    await waitFor(() =>
      expect(mocks.answerShape).toHaveBeenCalledWith(
        "sh-1",
        "/w",
        [{ question: "Q2", text: "admins" }],
        ["Q3"]
      )
    )
  })
})

describe("failure and drafting", () => {
  it("offers to run the turn again when the last one stopped", async () => {
    mocks.getShape.mockResolvedValue(
      session({
        failure: {
          at: "2026-11-25T09:30:00Z",
          agent: "claude-code",
          kind: "usage_limit",
          message: "limit reached",
        },
      })
    )
    mocks.resumeShape.mockResolvedValue(session())
    open()
    expect(await screen.findByText("The agent hit a usage limit.")).toBeTruthy()
    expect(
      screen.getByText(/answers and the map so far are kept/i)
    ).toBeTruthy()
    click("Try the turn again")
    await waitFor(() =>
      expect(mocks.resumeShape).toHaveBeenCalledWith("sh-1", "/w")
    )
  })

  it("lets a session with no question and no story be drafted", async () => {
    mocks.getShape.mockResolvedValue(session({ questions: [] }))
    mocks.resumeShape.mockResolvedValue(ready())
    open()
    await clickWhenReady("Draft the story")
    expect(await screen.findByText("Example map")).toBeTruthy()
    expect(mocks.resumeShape).toHaveBeenCalledWith("sh-1", "/w")
  })
})

describe("example map", () => {
  beforeEach(() => {
    mocks.getShape.mockResolvedValue(ready())
    mocks.getShapeSource.mockResolvedValue({
      path: "internal/refund.go",
      line: 12,
      lines: [
        { n: 11, text: "func a() {", hit: false },
        { n: 12, text: "  refuse()", hit: true },
      ],
    })
  })

  it("shows each fact with its source, and an example resting on an assumption", async () => {
    open()
    expect(await screen.findByText("Only admins refund")).toBeTruthy()
    expect(
      screen.getByRole("button", {
        name: "Source: code · internal/refund.go:12",
      })
    ).toBeTruthy()
    expect(
      screen.getByRole("button", { name: "Source: answer · A1" })
    ).toBeTruthy()
    expect(
      screen.getByText("2 rules, 2 examples. Every item shows its source.")
    ).toBeTruthy()
  })

  it("opens the code behind a source", async () => {
    open()
    fireEvent.click(
      await screen.findByRole("button", {
        name: "Source: code · internal/refund.go:12",
      })
    )
    expect(await screen.findByText("refuse()")).toBeTruthy()
    expect(mocks.getShapeSource).toHaveBeenCalledWith(
      "sh-1",
      "/w",
      "internal/refund.go:12"
    )
    expect(screen.getByText("Source for R1")).toBeTruthy()
  })

  it("explains a source that is not code in words", async () => {
    open()
    fireEvent.click(
      await screen.findByRole("button", { name: "Source: answer · A1" })
    )
    expect(await screen.findByText(/Question Q1\? → answer Q1/)).toBeTruthy()
    expect(mocks.getShapeSource).not.toHaveBeenCalled()
  })

  it("goes to the seal when there is no split and no note", async () => {
    open()
    fireEvent.click(
      await screen.findByRole("button", { name: "Looks right: continue" })
    )
    expect(await screen.findByText("The story")).toBeTruthy()
    expect(screen.getByText("Draft · not sealed")).toBeTruthy()
  })
})

describe("split", () => {
  const split = (over: Partial<Session> = {}) =>
    session({
      status: "split",
      questions: [],
      split: [
        { story: "Show why it failed", leftOut: "The Retry button" },
        { story: "Retry a delivery", leftOut: "Failure reasons" },
      ],
      ...over,
    })

  async function toSplit() {
    open()
    fireEvent.click(
      await screen.findByRole("button", { name: "Looks right: continue" })
    )
    await screen.findByText("The agent suggests splitting this story")
  }

  it("reopens on the split step once the map was confirmed", async () => {
    mocks.getShape.mockResolvedValue(split())
    await toSplit()
    cleanup()
    open()
    expect(
      await screen.findByText("The agent suggests splitting this story")
    ).toBeTruthy()
    expect(screen.queryByText("Looks right: continue")).toBeNull()
  })

  it("splits into linked sessions and lists them", async () => {
    mocks.getShape.mockResolvedValue(split())
    mocks.decideShapeSplit.mockResolvedValue(
      split({ children: ["c-1", "c-2"] })
    )
    await toSplit()
    click("Split into 2 stories")
    expect(
      await screen.findByText("Split into 2 stories", { selector: "h2" })
    ).toBeTruthy()
    expect(mocks.decideShapeSplit).toHaveBeenCalledWith("sh-1", "/w", "split")
    const links = screen.getAllByRole("link", { name: "Open this story" })
    expect(links[1]?.getAttribute("href")).toBe("/shaping/c-2?workspace=%2Fw")
  })

  it("keeps one story and runs the agent again", async () => {
    mocks.getShape.mockResolvedValue(split())
    mocks.decideShapeSplit.mockResolvedValue(ready({ splitKept: true }))
    await toSplit()
    click("Keep as one story")
    expect(await screen.findByText("Draft · not sealed")).toBeTruthy()
    expect(mocks.decideShapeSplit).toHaveBeenCalledWith("sh-1", "/w", "keep")
  })
})

describe("context notes", () => {
  const notes = (over: Partial<Session> = {}) =>
    ready({
      proposals: [
        {
          id: "P1",
          section: "Webhooks",
          text: "Retry 3 per hour.",
          status: "proposed",
        },
        {
          id: "P2",
          section: "Access",
          text: "Any user can retry.",
          status: "proposed",
        },
      ],
      ...over,
    })

  async function toContext() {
    mocks.getShape.mockResolvedValue(notes())
    open()
    fireEvent.click(
      await screen.findByRole("button", { name: "Looks right: continue" })
    )
    await screen.findByText("2 notes for your product context")
  }

  it("previews what accepting writes and sends the decisions on continue", async () => {
    mocks.previewShapeContext.mockResolvedValue({
      path: "span/context/product.md",
      lines: [
        { n: 41, kind: "ctx", text: "## Webhooks" },
        { n: 42, kind: "add", text: "Retry 3 per hour." },
      ],
    })
    mocks.settleShapeContext.mockResolvedValue(
      notes({
        proposals: [
          {
            id: "P1",
            section: "Webhooks",
            text: "Retry 3 per hour.",
            status: "accepted",
          },
        ],
      })
    )
    await toContext()
    click("Accept P1")
    click("Skip P2")
    const diff = await screen.findByRole("region", { name: "Context change" })
    expect(within(diff).getByText("Retry 3 per hour.")).toBeTruthy()
    expect(mocks.previewShapeContext).toHaveBeenCalledWith("sh-1", "/w", ["P1"])
    expect(screen.getByText("1 accepted, 0 still proposed.")).toBeTruthy()
    expect(mocks.settleShapeContext).not.toHaveBeenCalled()
    click("Continue to seal")
    expect(await screen.findByText("Draft · not sealed")).toBeTruthy()
    expect(mocks.settleShapeContext).toHaveBeenCalledWith("sh-1", "/w", {
      accept: ["P1"],
      skip: ["P2"],
    })
  })

  it("undoes a staged decision and says what is dropped when continuing", async () => {
    await toContext()
    click("Skip P1")
    click("Undo P1")
    expect(screen.getByRole("button", { name: "Accept P1" })).toBeTruthy()
    expect(
      screen.getByText(
        /2 still proposed. They are dropped if you continue now./
      )
    ).toBeTruthy()
    click("Continue to seal")
    expect(await screen.findByText("Draft · not sealed")).toBeTruthy()
    expect(mocks.settleShapeContext).not.toHaveBeenCalled()
  })
})

describe("seal", () => {
  async function toSeal(value: Session) {
    mocks.getShape.mockResolvedValue(value)
    open()
    fireEvent.click(
      await screen.findByRole("button", { name: "Looks right: continue" })
    )
    await screen.findByText("Draft · not sealed")
  }

  it("lists what sealing locks in, then seals", async () => {
    mocks.sealShape.mockResolvedValue(sealed())
    await toSeal(
      ready({
        proposals: [{ id: "P1", section: "A", text: "t", status: "accepted" }],
      })
    )
    expect(screen.getByText("No open questions.")).toBeTruthy()
    expect(screen.getByText("1 assumption will be sealed in")).toBeTruthy()
    fireEvent.click(screen.getByRole("tab", { name: "Preview" }))
    expect(screen.getByText("Because.")).toBeTruthy()
    click("Seal story: cannot be changed")
    expect(await screen.findByText(/This story cannot change./)).toBeTruthy()
    expect(mocks.sealShape).toHaveBeenCalledWith("sh-1", "/w")
  })

  it("lets an assumption be answered instead, which runs the agent again", async () => {
    mocks.answerShape.mockResolvedValue(
      ready({
        questions: [
          question("Q1", "answered", { answerId: "A1" }),
          question("Q2", "answered", { answerId: "A2" }),
        ],
      })
    )
    await toSeal(ready())
    click("Answer an assumption instead")
    click("Answer Q2 instead")
    fireEvent.change(screen.getByLabelText("Your answer to Q2"), {
      target: { value: "five" },
    })
    click("Save answer for Q2")
    click("Continue to example map")
    await waitFor(() =>
      expect(mocks.answerShape).toHaveBeenCalledWith(
        "sh-1",
        "/w",
        [{ question: "Q2", text: "five" }],
        []
      )
    )
  })

  it("shows why the seal was refused", async () => {
    mocks.sealShape.mockRejectedValue(
      new ApiError("shape_not_ready", "open questions: Q3")
    )
    await toSeal(ready())
    click("Seal story: cannot be changed")
    expect(await screen.findByText("open questions: Q3")).toBeTruthy()
  })
})

describe("sealed story", () => {
  it("is read-only, with a digest and four views of the session", async () => {
    mocks.getShape.mockResolvedValue(sealed())
    open()
    expect(await screen.findByText(/This story cannot change./)).toBeTruthy()
    expect(screen.getByText("Read-only")).toBeTruthy()
    expect(
      screen.getByRole("button", { name: /Copy digest/ }).getAttribute("title")
    ).toBe("sha256:abcdef0123456789ffff")
    expect(screen.getByText(/1 assumption was sealed in: Q2/)).toBeTruthy()
    expect(screen.queryByRole("button", { name: /Seal story/ })).toBeNull()
    fireEvent.click(screen.getByRole("tab", { name: "Questions" }))
    expect(screen.getByText("Answered · A1")).toBeTruthy()
    fireEvent.click(screen.getByRole("tab", { name: "Example map" }))
    expect(screen.getByText("Only admins refund")).toBeTruthy()
    expect(screen.queryByRole("button", { name: /Source:/ })).toBeNull()
  })

  it("hands the story and the Jira key to the desk to start a run from", async () => {
    mocks.getShape.mockResolvedValue(
      sealed({
        jira: { key: "PRJ-9", url: "https://j/PRJ-9", status: "Backlog" },
      })
    )
    open()
    await clickWhenReady("Create run")
    const arrival = JSON.parse(
      (await screen.findByTestId("arrival")).textContent ?? "null"
    ) as Record<string, string>
    expect(arrival.story).toBe("# Refund\n\n## Why\nBecause.")
    expect(arrival.issue).toBe("PRJ-9")
    expect(arrival.workspace).toBe("/w")
    expect(arrival.repo).toBe("api")
    expect(mocks.startRun).not.toHaveBeenCalled()
  })

  it("hands over no issue when Jira was skipped", async () => {
    mocks.getShape.mockResolvedValue(sealed())
    open()
    await clickWhenReady("Create run")
    const arrival = JSON.parse(
      (await screen.findByTestId("arrival")).textContent ?? "null"
    ) as Record<string, string>
    expect(arrival.issue).toBe("")
  })

  it("starts a new session from the sealed story and opens it", async () => {
    mocks.getShape.mockResolvedValue(sealed())
    mocks.forkShape.mockResolvedValue(session({ id: "sh-2" }))
    open()
    await clickWhenReady("New session from this story")
    expect(mocks.forkShape).toHaveBeenCalledWith("sh-1", "/w")
    await waitFor(() =>
      expect(mocks.getShape).toHaveBeenCalledWith("sh-2", "/w")
    )
  })

  it("links the Jira issue instead of offering a second preview", async () => {
    mocks.getShape.mockResolvedValue(
      sealed({
        jira: { key: "PRJ-9", url: "https://j/PRJ-9", status: "Backlog" },
      })
    )
    open()
    const link = await screen.findByRole("link", { name: "Open PRJ-9" })
    expect(link.getAttribute("href")).toBe("https://j/PRJ-9")
    expect(screen.queryByRole("button", { name: "Preview in Jira" })).toBeNull()
  })
})

describe("jira", () => {
  async function toJira() {
    mocks.getShape.mockResolvedValue(sealed())
    open()
    click(
      await screen
        .findByRole("button", { name: "Preview in Jira" })
        .then(() => "Preview in Jira")
    )
    await screen.findByText("Nothing sent yet")
  }

  async function loadPreview() {
    fireEvent.change(screen.getByLabelText("Project key"), {
      target: { value: "PRJ" },
    })
    fireEvent.change(screen.getByLabelText("Board id (optional)"), {
      target: { value: "7" },
    })
    click("Preview")
  }

  it("previews the exact request first and creates only after the confirm button", async () => {
    mocks.previewShapeIssue.mockResolvedValue(preview)
    mocks.createShapeIssue.mockResolvedValue(
      sealed({
        jira: { key: "PRJ-9", url: "https://j/PRJ-9", status: "Backlog" },
      })
    )
    await toJira()
    expect(
      (
        screen.getByRole("button", {
          name: "Create in Jira",
        }) as HTMLButtonElement
      ).disabled
    ).toBe(true)
    await loadPreview()
    expect(await screen.findByText("PRJ Kanban")).toBeTruthy()
    expect(screen.getByText("Ready for grooming")).toBeTruthy()
    expect(
      within(screen.getByRole("region", { name: "Payload" })).getByText(
        /"labels": \[/
      )
    ).toBeTruthy()
    expect(mocks.previewShapeIssue).toHaveBeenCalledWith(
      "sh-1",
      "/w",
      "PRJ",
      "7"
    )
    expect(mocks.createShapeIssue).not.toHaveBeenCalled()
    click("Create in Jira")
    expect(await screen.findByText("Created PRJ-9.")).toBeTruthy()
    expect(
      screen
        .getByRole("link", { name: "https://j/PRJ-9".replace("https://", "") })
        .getAttribute("href")
    ).toBe("https://j/PRJ-9")
    expect(mocks.createShapeIssue).toHaveBeenCalledWith(
      "sh-1",
      "/w",
      "PRJ",
      "7"
    )
  })

  it("says when the backlog is off and lets the person pick another project", async () => {
    mocks.previewShapeIssue.mockRejectedValue(
      new ApiError(
        "jira_backlog_disabled",
        "the Backlog column has no statuses"
      )
    )
    await toJira()
    await loadPreview()
    expect(await screen.findByText(/backlog is not enabled/i)).toBeTruthy()
    expect(screen.getByText("Jira refused")).toBeTruthy()
    click("Choose another project")
    expect(screen.queryByText("Jira refused")).toBeNull()
  })

  it("shows the board list of an ambiguous project", async () => {
    mocks.previewShapeIssue.mockRejectedValue(
      new ApiError("jira_board_ambiguous", "boards: 7 Alpha, 8 Beta")
    )
    await toJira()
    await loadPreview()
    expect(await screen.findByText(/boards: 7 Alpha, 8 Beta/)).toBeTruthy()
  })

  it("says the issue exists when it could not reach the backlog", async () => {
    mocks.previewShapeIssue.mockResolvedValue(preview)
    mocks.createShapeIssue.mockRejectedValue(
      new ApiError("jira_not_in_backlog", "PRJ-9 is in To Do")
    )
    await toJira()
    mocks.getShape.mockResolvedValue(
      sealed({
        jira: { key: "PRJ-9", url: "https://j/PRJ-9", status: "To Do" },
      })
    )
    await loadPreview()
    await screen.findByText("PRJ Kanban")
    click("Create in Jira")
    expect(await screen.findByText(/PRJ-9 is in To Do/)).toBeTruthy()
    expect(await screen.findByText("Created PRJ-9.")).toBeTruthy()
  })
})
