import { act, cleanup, render, screen } from "@testing-library/react"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

import { StartUp } from "@/components/startup/StartUp"

function motion(reduced: boolean) {
  vi.stubGlobal("matchMedia", () => ({ matches: reduced }))
}

beforeEach(() => {
  vi.useFakeTimers()
  window.sessionStorage.clear()
  motion(false)
})
afterEach(() => {
  cleanup()
  vi.useRealTimers()
  vi.unstubAllGlobals()
})

describe("StartUp", () => {
  it("fills the stages and then leaves within two seconds", () => {
    render(<StartUp />)
    expect(screen.getByRole("status", { name: "Starting Span" })).toBeTruthy()
    for (let tick = 0; tick < 20; tick += 1) {
      act(() => {
        vi.advanceTimersByTime(100)
      })
    }
    expect(screen.queryByRole("status", { name: "Starting Span" })).toBeNull()
  })

  it("shows once per tab", () => {
    const first = render(<StartUp />)
    first.unmount()
    render(<StartUp />)
    expect(screen.queryByRole("status", { name: "Starting Span" })).toBeNull()
  })

  it("does not show with reduced motion", () => {
    motion(true)
    render(<StartUp />)
    expect(screen.queryByRole("status", { name: "Starting Span" })).toBeNull()
  })
})
