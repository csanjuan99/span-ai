import { cleanup, render, screen } from "@testing-library/react"
import { MemoryRouter } from "react-router"
import { afterEach, describe, expect, it } from "vitest"

import { StartedRuns } from "@/components/started-runs/StartedRuns"

afterEach(cleanup)

function show(props: Parameters<typeof StartedRuns>[0]) {
  render(
    <MemoryRouter>
      <StartedRuns {...props} />
    </MemoryRouter>
  )
}

describe("StartedRuns", () => {
  it("links every run, named by its repo", () => {
    show({
      runs: [
        { id: "span-1", stage: "intake", repo: "api" },
        { id: "span-2", stage: "intake", repo: "web" },
      ],
      partial: false,
      notes: "",
    })
    expect(
      screen.getByRole("link", { name: /api.*span-1/ }).getAttribute("href")
    ).toBe("/spans/span-1")
    expect(
      screen.getByRole("link", { name: /web.*span-2/ }).getAttribute("href")
    ).toBe("/spans/span-2")
  })

  it("says which repos did not start when the start was partial", () => {
    show({
      runs: [{ id: "span-1", stage: "intake", repo: "api" }],
      partial: true,
      notes: "repo web did not start: another run is active in this checkout\n",
    })
    expect(screen.getByText(/some repos did not start/i)).toBeTruthy()
    expect(screen.getByText(/repo web did not start/)).toBeTruthy()
  })

  it("does not claim a partial start when everything started", () => {
    show({
      runs: [{ id: "span-1", stage: "intake", repo: "" }],
      partial: false,
      notes: "",
    })
    expect(screen.queryByText(/did not start/i)).toBeNull()
    expect(screen.getByRole("link", { name: /span-1/ })).toBeTruthy()
  })
})
