import { cleanup, fireEvent, render, screen } from "@testing-library/react"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

import { NotificationPrefs } from "@/components/notify/NotificationPrefs"
import { loadPrefs } from "@/lib/notify"

function box(name: string): HTMLElement {
  return screen.getByRole("checkbox", { name })
}

function on(name: string): boolean {
  return box(name).getAttribute("aria-checked") === "true"
}

beforeEach(() => localStorage.clear())
afterEach(() => {
  cleanup()
  vi.unstubAllGlobals()
})

describe("NotificationPrefs", () => {
  it("lists every event with the toggles it has, as the saved choices say", () => {
    render(<NotificationPrefs />)
    expect(on("A run needs me: toast")).toBe(true)
    expect(on("A run needs me: tab and badge")).toBe(true)
    expect(on("A run needs me: system")).toBe(false)
  })

  it("saves a toggle at once", () => {
    render(<NotificationPrefs />)
    fireEvent.click(box("A run finished: toast"))
    expect(loadPrefs().events.finished.toast).toBe(false)
  })

  it("asks the browser only when a system toggle goes on", () => {
    const ask = vi.fn(() => Promise.resolve("granted"))
    vi.stubGlobal("Notification", {
      permission: "default",
      requestPermission: ask,
    })
    render(<NotificationPrefs />)
    expect(ask).not.toHaveBeenCalled()
    fireEvent.click(box("A run needs me: system"))
    expect(ask).toHaveBeenCalledTimes(1)
    expect(loadPrefs().events.needs.system).toBe(true)
  })

  it("says system notifications are not available where the browser has none", () => {
    vi.stubGlobal("Notification", undefined)
    render(<NotificationPrefs />)
    expect(
      screen.getByText("This browser has no system notifications")
    ).toBeTruthy()
    expect((box("A run needs me: system") as HTMLButtonElement).disabled).toBe(
      true
    )
  })

  it("says what the browser answered when it blocks them", () => {
    vi.stubGlobal("Notification", {
      permission: "denied",
      requestPermission: vi.fn(),
    })
    render(<NotificationPrefs />)
    expect(screen.getByText("System notifications are blocked")).toBeTruthy()
    expect((box("A run needs me: system") as HTMLButtonElement).disabled).toBe(
      true
    )
  })

  it("keeps the in-front choice", () => {
    render(<NotificationPrefs />)
    const toggle = screen.getByRole("switch", {
      name: "Only when Span is in the background",
    })
    expect(toggle.getAttribute("aria-checked")).toBe("true")
    fireEvent.click(toggle)
    expect(loadPrefs().background).toBe(false)
  })

  it("turns the sound on, off by default", () => {
    render(<NotificationPrefs />)
    const toggle = screen.getByRole("switch", { name: "Sound" })
    expect(toggle.getAttribute("aria-checked")).toBe("false")
    fireEvent.click(toggle)
    expect(toggle.getAttribute("aria-checked")).toBe("true")
    expect(loadPrefs().sound).toBe(true)
  })

  it("lists the agent-blocked and run-stopped events", () => {
    render(<NotificationPrefs />)
    expect(screen.getByText("Agent blocked")).toBeTruthy()
    expect(screen.getByText("A run stopped")).toBeTruthy()
  })

  it("asks the browser only when Allow notifications is pressed", () => {
    const requestPermission = vi.fn().mockResolvedValue("granted")
    vi.stubGlobal("Notification", { permission: "default", requestPermission })
    render(<NotificationPrefs />)
    expect(requestPermission).not.toHaveBeenCalled()
    fireEvent.click(screen.getByRole("button", { name: "Allow notifications" }))
    expect(requestPermission).toHaveBeenCalledTimes(1)
  })
})
