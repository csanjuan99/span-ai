import { cleanup, render, screen } from "@testing-library/react"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

vi.mock("@/pages/Desk", () => ({ Desk: () => <p>desk page</p> }))
vi.mock("@/pages/Onboarding", () => ({ Onboarding: () => <p>welcome page</p> }))
vi.mock("@/components/startup/StartUp", () => ({ StartUp: () => null }))
vi.mock("@/components/help", () => ({ CommandPalette: () => null }))

import { App } from "@/App"
import { saveOnboarding } from "@/lib/onboarding"

describe("App first visit", () => {
  beforeEach(() => {
    localStorage.clear()
    window.history.pushState({}, "", "/")
  })
  afterEach(cleanup)

  it("sends a first visit to the welcome page", () => {
    render(<App />)
    expect(screen.getByText("welcome page")).toBeTruthy()
    expect(window.location.pathname).toBe("/welcome")
  })

  it("goes straight to the desk once the welcome was skipped", () => {
    saveOnboarding({ skipped: true })
    render(<App />)
    expect(screen.getByText("desk page")).toBeTruthy()
  })
})
