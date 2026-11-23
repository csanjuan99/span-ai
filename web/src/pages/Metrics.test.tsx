import { cleanup, render, screen, waitFor } from "@testing-library/react"
import { MemoryRouter } from "react-router"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

const getJSON = vi.fn()

vi.mock("@/lib/api", () => ({ getJSON: (url: string) => getJSON(url) }))

import { Metrics } from "@/pages/Metrics"

afterEach(cleanup)
beforeEach(() => getJSON.mockReset())

function show() {
  render(
    <MemoryRouter>
      <Metrics />
    </MemoryRouter>,
  )
}

describe("Metrics", () => {
  it("says what is missing instead of drawing empty tables when there are no runs", async () => {
    getJSON.mockResolvedValue({ runs: { total: 0 }, stages: [], agents: [], recent: [] })
    show()
    await waitFor(() => expect(screen.getByText(/No runs yet/)).toBeTruthy())
    expect(screen.queryByRole("table")).toBeNull()
  })

  it("shows the stages, the agent and model, and the recent runs", async () => {
    getJSON.mockResolvedValue({
      runs: { total: 2, done: 1, stopped: 0, held: 1, running: 0, firstTry: 1 },
      stages: [{ stage: "build", runs: 2, advance: 1, hold: 1, stop: 0, medianSeconds: 200, p90Seconds: 3900 }],
      agents: [{ agent: "pi", model: "ollama/m", stages: 2, advance: 1, hold: 1, stop: 0 }],
      recent: [{ id: "r1", label: "SAL-6", stage: "build", verdict: "hold", iterations: 2, seconds: 90, agents: ["build pi/ollama/m"] }],
    })
    show()
    await waitFor(() => expect(screen.getByText("SAL-6")).toBeTruthy())
    expect(screen.getByText("3m 20s")).toBeTruthy()
    expect(screen.getByText("1h 05m")).toBeTruthy()
    expect(screen.getByText(/ollama\/m$/, { selector: "th span" })).toBeTruthy()
    expect(screen.getByText(/not recorded by runs yet/)).toBeTruthy()
  })
})
