import { cleanup, fireEvent, render, screen } from "@testing-library/react"
import { afterEach, describe, expect, it, vi } from "vitest"

import { ErrorBoundary } from "@/components/error-boundary/ErrorBoundary"

let broken = true

function Fragile() {
  if (broken) {
    throw new Error("the stage list was not an array")
  }
  return <p>the page is back</p>
}

afterEach(() => {
  cleanup()
  vi.restoreAllMocks()
  broken = true
})

describe("ErrorBoundary", () => {
  it("shows the fault instead of a blank page", () => {
    const logged = vi.spyOn(console, "error").mockImplementation(() => undefined)
    render(
      <ErrorBoundary>
        <Fragile />
      </ErrorBoundary>,
    )
    expect(screen.getByText("Something on this page broke.")).toBeTruthy()
    expect(screen.getByText("the stage list was not an array")).toBeTruthy()
    expect(screen.getByRole("button", { name: "Try again" })).toBeTruthy()
    expect(screen.getByRole("button", { name: "Reload page" })).toBeTruthy()
    // The fault is reported, not swallowed.
    expect(logged).toHaveBeenCalled()
  })

  it("renders the children again once Try again is pressed and the fault is gone", () => {
    vi.spyOn(console, "error").mockImplementation(() => undefined)
    render(
      <ErrorBoundary>
        <Fragile />
      </ErrorBoundary>,
    )
    broken = false
    fireEvent.click(screen.getByRole("button", { name: "Try again" }))
    expect(screen.getByText("the page is back")).toBeTruthy()
    expect(screen.queryByText("Something on this page broke.")).toBeNull()
  })

  it("starts over when its reset key changes", () => {
    vi.spyOn(console, "error").mockImplementation(() => undefined)
    const { rerender } = render(
      <ErrorBoundary resetKey="/spans/a">
        <Fragile />
      </ErrorBoundary>,
    )
    broken = false
    rerender(
      <ErrorBoundary resetKey="/spans/b">
        <Fragile />
      </ErrorBoundary>,
    )
    expect(screen.getByText("the page is back")).toBeTruthy()
  })

  it("reloads the whole page for a fault that comes straight back", () => {
    vi.spyOn(console, "error").mockImplementation(() => undefined)
    const reload = vi.fn()
    render(
      <ErrorBoundary reload={reload}>
        <Fragile />
      </ErrorBoundary>,
    )
    fireEvent.click(screen.getByRole("button", { name: "Try again" }))
    expect(screen.getByText("Something on this page broke.")).toBeTruthy()
    fireEvent.click(screen.getByRole("button", { name: "Reload page" }))
    expect(reload).toHaveBeenCalledTimes(1)
  })
})
