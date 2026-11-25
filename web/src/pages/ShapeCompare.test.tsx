import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react"
import { MemoryRouter, Route, Routes } from "react-router"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

import type { ShapeSession } from "@/lib/types"

const listShapes = vi.fn()
const compareShapes = vi.fn()

vi.mock("@/lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/api")>()
  return {
    ...actual,
    listShapes: (...args: unknown[]) => listShapes(...args),
  }
})

vi.mock("@/lib/shape-compare", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/shape-compare")>()
  return {
    ...actual,
    compareShapes: (...args: unknown[]) => compareShapes(...args),
  }
})

import { ShapeCompare } from "@/pages/ShapeCompare"

afterEach(cleanup)

function session(id: string, repo: string): ShapeSession {
  return {
    id,
    workspace: ".",
    repo,
    idea: id,
    status: "ready",
    rules: [],
    examples: [],
    questions: [],
    split: [],
    proposals: [],
    story: "",
    digest: "",
    jira: null,
    ref: id.toUpperCase(),
    title: id,
  }
}

function open() {
  render(
    <MemoryRouter initialEntries={["/shaping/compare?workspace=."]}>
      <Routes>
        <Route path="/shaping/compare" element={<ShapeCompare />} />
      </Routes>
    </MemoryRouter>
  )
}

beforeEach(() => {
  listShapes.mockReset()
  compareShapes.mockReset()
  listShapes.mockResolvedValue([session("one", "api"), session("two", "api")])
  compareShapes.mockResolvedValue({
    repo: "api",
    sessions: [],
    same: [],
    conflicts: [],
    duplicates: [
      {
        a: "one",
        b: "two",
        aRef: "ONE",
        bRef: "TWO",
        percent: 82,
        aItems: ["R1"],
        bItems: ["R1"],
        dismissed: false,
      },
    ],
    unique: 0,
    estimate: true,
  })
})

describe("compare sessions", () => {
  it("compares the sessions checked on the page and calls the match an estimate", async () => {
    open()
    fireEvent.click(await screen.findByRole("checkbox", { name: "Compare ONE" }))
    fireEvent.click(screen.getByRole("checkbox", { name: "Compare TWO" }))
    await waitFor(() =>
      expect(compareShapes).toHaveBeenCalledWith(".", ["one", "two"])
    )
    expect((await screen.findAllByText(/estimate/)).length).toBeGreaterThan(0)
    expect(screen.queryByText(/82%/)).toBeNull()
  })
})
