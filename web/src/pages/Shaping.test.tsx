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
const getJSON = vi.fn()

vi.mock("@/lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/api")>()
  return {
    ApiError: actual.ApiError,
    listShapes: (...args: unknown[]) => listShapes(...args),
    createShape: (...args: unknown[]) => createShape(...args),
    getJSON: (url: string) => getJSON(url),
  }
})

import { Shaping } from "@/pages/Shaping"

afterEach(cleanup)

const row: ShapeSession = {
  id: "sh-1",
  workspace: ".",
  repo: "",
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
}

function open() {
  render(
    <MemoryRouter initialEntries={["/shaping"]}>
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
  getJSON.mockReset()
  listShapes.mockResolvedValue([row])
  getJSON.mockResolvedValue({ repos: [{ name: "api" }, { name: "web" }] })
})

describe("shaping list", () => {
  it("lists the sessions of the workspace as links", async () => {
    open()
    const link = await screen.findByRole("link", { name: /let people pay/ })
    expect(link.getAttribute("href")).toBe("/shaping/sh-1?workspace=.")
    expect(listShapes).toHaveBeenCalledWith(".")
  })

  it("starts a session from the idea and the chosen repo, then opens it", async () => {
    createShape.mockResolvedValue({ ...row, id: "sh-2" })
    open()
    await screen.findByRole("link", { name: /let people pay/ })
    fireEvent.change(screen.getByLabelText("Idea"), {
      target: { value: "refund payments" },
    })
    fireEvent.click(screen.getByRole("button", { name: "Find repos" }))
    fireEvent.change(await screen.findByLabelText("Repository"), {
      target: { value: "web" },
    })
    fireEvent.click(screen.getByRole("button", { name: "Start shaping" }))
    expect(await screen.findByText("session page")).toBeTruthy()
    expect(createShape).toHaveBeenCalledWith({
      workspace: ".",
      repo: "web",
      idea: "refund payments",
    })
  })

  it("shows a clear pending state and blocks a second start", async () => {
    createShape.mockReturnValue(new Promise(() => undefined))
    open()
    fireEvent.change(await screen.findByLabelText("Idea"), {
      target: { value: "refund payments" },
    })
    fireEvent.click(screen.getByRole("button", { name: "Start shaping" }))
    const pending = await screen.findByRole("button", { name: "Thinking…" })
    fireEvent.click(pending)
    await waitFor(() => expect(createShape).toHaveBeenCalledTimes(1))
  })
})
