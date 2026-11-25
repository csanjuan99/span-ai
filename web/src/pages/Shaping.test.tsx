import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react"
import { MemoryRouter, Route, Routes } from "react-router"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

import type { ShapeSession } from "@/lib/types"

const listShapes = vi.fn()
const createShape = vi.fn()
const archiveShape = vi.fn()
const getJSON = vi.fn()

vi.mock("@/lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/api")>()
  return {
    ...actual,
    listShapes: (...args: unknown[]) => listShapes(...args),
    createShape: (...args: unknown[]) => createShape(...args),
    archiveShape: (...args: unknown[]) => archiveShape(...args),
    getJSON: (url: string) => getJSON(url),
  }
})

import { Shaping } from "@/pages/Shaping"

afterEach(cleanup)

const row: ShapeSession = {
  id: "sh-1",
  workspace: ".",
  repo: "api",
  idea: "let people pay",
  status: "ready",
  rules: [],
  examples: [],
  questions: [],
  split: [],
  proposals: [],
  story: "",
  digest: "",
  jira: null,
  agent: "codex",
  ref: "SES-0001",
  title: "let people pay",
  openQuestions: 0,
  jiraKey: "BILL-9",
  updatedAt: new Date().toISOString(),
  archived: false,
  failure: null,
}

function open(url = "/shaping") {
  render(
    <MemoryRouter initialEntries={[url]}>
      <Routes>
        <Route path="/shaping" element={<Shaping />} />
        <Route path="/shaping/:id" element={<p>session page</p>} />
      </Routes>
    </MemoryRouter>
  )
}

beforeEach(() => {
  listShapes.mockReset()
  createShape.mockReset()
  archiveShape.mockReset()
  getJSON.mockReset()
  listShapes.mockResolvedValue([row])
  getJSON.mockImplementation((url: string) => {
    if (url.startsWith("/repos")) {
      return Promise.resolve({ repos: [{ name: "api" }, { name: "web" }] })
    }
    return Promise.resolve({})
  })
})

describe("shape a story: start", () => {
  it("lists the latest sessions with the action each needs", async () => {
    open()
    const action = await screen.findByRole("link", { name: "Review" })
    expect(action.getAttribute("href")).toBe("/shaping/sh-1?workspace=.")
    expect(screen.getByText("SES-0001")).toBeTruthy()
    expect(screen.getByText(/BILL-9/)).toBeTruthy()
    expect(
      screen.getByRole("link", { name: "See all 1" }).getAttribute("href")
    ).toBe("/shaping?workspace=.&view=sessions")
  })

  it("starts a session from the idea and the chosen repo, then opens it", async () => {
    createShape.mockResolvedValue({ ...row, id: "sh-2" })
    open()
    await screen.findByRole("link", { name: "Review" })
    fireEvent.change(screen.getByLabelText("Your idea"), {
      target: { value: "refund payments" },
    })
    fireEvent.change(await screen.findByLabelText("Repository"), {
      target: { value: "web" },
    })
    fireEvent.click(screen.getByRole("button", { name: "Start interview" }))
    expect(await screen.findByText("session page")).toBeTruthy()
    expect(createShape).toHaveBeenCalledWith({
      workspace: ".",
      repo: "web",
      idea: "refund payments",
    })
  })

  it("starts from a Jira story when the idea is just its key", async () => {
    createShape.mockResolvedValue({ ...row, id: "sh-3" })
    open()
    await screen.findByRole("link", { name: "Review" })
    fireEvent.change(screen.getByLabelText("Your idea"), {
      target: { value: "bill-2031" },
    })
    expect(
      screen.getByText("Starts from the Jira story BILL-2031.")
    ).toBeTruthy()
    fireEvent.click(screen.getByRole("button", { name: "Start interview" }))
    await screen.findByText("session page")
    expect(createShape).toHaveBeenCalledWith({
      workspace: ".",
      issue: "BILL-2031",
      idea: "",
    })
  })

  it("shows a clear pending state and blocks a second start", async () => {
    createShape.mockReturnValue(new Promise(() => undefined))
    open()
    fireEvent.change(await screen.findByLabelText("Your idea"), {
      target: { value: "refund payments" },
    })
    fireEvent.click(screen.getByRole("button", { name: "Start interview" }))
    const pending = await screen.findByRole("button", { name: "Thinking…" })
    fireEvent.click(pending)
    await waitFor(() => expect(createShape).toHaveBeenCalledTimes(1))
  })

  it("reads the folder from the address", async () => {
    open("/shaping?workspace=%2Fwork%2Fbilling")
    await screen.findByRole("link", { name: "Review" })
    expect(listShapes).toHaveBeenCalledWith("/work/billing")
  })
})

describe("shape a story: first session", () => {
  it("offers an example idea and starts with nothing else chosen", async () => {
    listShapes.mockResolvedValue([])
    createShape.mockResolvedValue({ ...row, id: "sh-4" })
    open()
    fireEvent.click(
      await screen.findByRole("button", { name: "Try an example idea" })
    )
    fireEvent.click(screen.getByRole("button", { name: "Start interview" }))
    await screen.findByText("session page")
    expect(createShape).toHaveBeenCalledWith({
      workspace: ".",
      idea: expect.stringContaining("webhook delivery failed"),
    })
    expect(screen.queryByText("Recent sessions")).toBeNull()
  })
})

describe("shape a story: sessions", () => {
  const sealed: ShapeSession = {
    ...row,
    id: "sh-5",
    status: "sealed",
    ref: "SES-0005",
    title: "Refund a charge",
  }

  beforeEach(() => {
    listShapes.mockResolvedValue([row, sealed])
  })

  it("counts the sessions per state and says how many need you", async () => {
    open("/shaping?view=sessions")
    expect(await screen.findByText("2 sessions · 1 needs you")).toBeTruthy()
    const chips = screen.getByRole("group", { name: "Filter by status" })
    expect(chips.textContent).toContain("All2")
    expect(chips.textContent).toContain("Ready1")
    expect(chips.textContent).toContain("Sealed1")
  })

  it("asks the server for the filter, and clears it", async () => {
    open("/shaping?view=sessions")
    await screen.findByText("SES-0005")
    listShapes.mockResolvedValue([])
    fireEvent.click(screen.getByRole("button", { name: /^Sealed/ }))
    fireEvent.change(screen.getByRole("searchbox"), {
      target: { value: "refund" },
    })
    expect(
      await screen.findByText("No sealed session matches “refund”")
    ).toBeTruthy()
    expect(listShapes).toHaveBeenLastCalledWith(".", {
      status: "sealed",
      query: "refund",
      repo: "",
      agent: "",
      sort: "activity",
      archived: "",
    })
    // The old rows stayed until the server answered, so the empty state is its answer.
    expect(listShapes).toHaveBeenCalledTimes(2)
    listShapes.mockResolvedValue([row, sealed])
    fireEvent.click(screen.getByRole("button", { name: "Clear filters" }))
    expect(await screen.findByText("SES-0005")).toBeTruthy()
  })

  it("archives a session and reads the list again", async () => {
    archiveShape.mockResolvedValue({ ...sealed, archived: true })
    open("/shaping?view=sessions")
    fireEvent.click(
      await screen.findByRole("button", { name: "Archive Refund a charge" })
    )
    await waitFor(() =>
      expect(archiveShape).toHaveBeenCalledWith("sh-5", ".", true)
    )
    await waitFor(() => expect(listShapes).toHaveBeenCalledTimes(2))
  })

  it("does not offer to delete a sealed session", async () => {
    open("/shaping?view=sessions")
    await screen.findByText("SES-0005")
    expect(
      screen.queryByRole("button", { name: "Delete Refund a charge" })
    ).toBeNull()
    expect(
      screen.getByRole("button", { name: "Delete let people pay" })
    ).toBeTruthy()
  })
})
