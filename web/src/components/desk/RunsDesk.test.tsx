import { cleanup, fireEvent, render, screen } from "@testing-library/react"
import { MemoryRouter } from "react-router"
import { afterEach, describe, expect, it, vi } from "vitest"

import { RunsDesk } from "@/components/desk/RunsDesk"
import { head } from "@/test/heads"

afterEach(cleanup)

const runs = [
  head({
    id: "a",
    verdict: "hold",
    stage: "merge",
    issue: "SPAN-1",
    title: "Retry webhooks",
    reason: "conflict in 3 files",
  }),
  head({
    id: "b",
    verdict: "running",
    issue: "SPAN-3",
    title: "Cache rates",
    branch: "feat/rates",
  }),
  head({
    id: "c",
    verdict: "done",
    stage: "note",
    issue: "SPAN-4",
    title: "Validate codes",
  }),
]
const now = new Date("2026-11-24T15:00:00Z")

function show(onNew = vi.fn()) {
  render(
    <MemoryRouter>
      <RunsDesk
        runs={runs}
        now={now}
        onNew={onNew}
        onDeleted={() => undefined}
      />
    </MemoryRouter>
  )
  return onNew
}

describe("RunsDesk", () => {
  it("counts the runs and the ones that need me", () => {
    show()
    expect(screen.getByText("3 runs · 1 needs you")).toBeTruthy()
  })

  it("groups runs and says why one waits", () => {
    show()
    expect(screen.getByRole("region", { name: "Waiting for me" })).toBeTruthy()
    expect(screen.getByText(/merge · conflict in 3 files/)).toBeTruthy()
    expect(screen.getByRole("region", { name: "Running" })).toBeTruthy()
  })

  it("filters to what needs me", () => {
    show()
    fireEvent.click(screen.getByRole("button", { name: /Needs me/ }))
    expect(screen.queryByRole("region", { name: "Running" })).toBeNull()
    expect(screen.getByRole("region", { name: "Waiting for me" })).toBeTruthy()
  })

  it("searches by title and says so when nothing matches", () => {
    show()
    const box = screen.getByRole("searchbox")
    fireEvent.change(box, { target: { value: "rates" } })
    expect(screen.queryByText("Retry webhooks")).toBeNull()
    fireEvent.change(box, { target: { value: "checkout" } })
    expect(screen.getByText("No runs match “checkout”")).toBeTruthy()
    fireEvent.click(screen.getByRole("button", { name: "Clear search" }))
    expect(screen.getByText("Retry webhooks")).toBeTruthy()
  })

  it("asks for a new run", () => {
    const onNew = show()
    fireEvent.click(screen.getByRole("button", { name: "New run" }))
    expect(onNew).toHaveBeenCalledTimes(1)
  })
})
