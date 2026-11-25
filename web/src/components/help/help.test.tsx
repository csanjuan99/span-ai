import { act, cleanup, fireEvent, render, screen } from "@testing-library/react"
import { MemoryRouter, Route, Routes, useLocation } from "react-router"
import { afterEach, beforeEach, describe, expect, it } from "vitest"

import { CommandPalette, HelpPopover } from "@/components/help"
import { ThemeProvider } from "@/components/theme-provider"

function Where() {
  return <p data-testid="where">{useLocation().pathname}</p>
}

function mount() {
  render(
    <MemoryRouter initialEntries={["/metrics"]}>
      <ThemeProvider>
        <CommandPalette />
        <Routes>
          <Route path="*" element={<Where />} />
        </Routes>
      </ThemeProvider>
    </MemoryRouter>
  )
}

function press(key: string, init: KeyboardEventInit = {}) {
  act(() => {
    document.body.dispatchEvent(
      new KeyboardEvent("keydown", { key, bubbles: true, ...init })
    )
  })
}

beforeEach(() => {
  localStorage.clear()
  window.matchMedia = ((query: string) => ({
    matches: false,
    media: query,
    addEventListener: () => undefined,
    removeEventListener: () => undefined,
  })) as unknown as typeof window.matchMedia
})
afterEach(cleanup)

describe("CommandPalette", () => {
  it("opens on Cmd+K, filters, and goes where the command says", () => {
    mount()
    press("k", { metaKey: true })
    const box = screen.getByRole("combobox", {
      name: "Search or run a command",
    })
    fireEvent.change(box, { target: { value: "settings" } })
    fireEvent.keyDown(box, { key: "Enter" })
    expect(screen.getByTestId("where").textContent).toBe("/settings")
    expect(screen.queryByRole("dialog", { name: "Command palette" })).toBeNull()
  })

  it("moves with the arrow keys and runs the active command", () => {
    mount()
    press("k", { ctrlKey: true })
    const box = screen.getByRole("combobox")
    fireEvent.keyDown(box, { key: "ArrowDown" })
    fireEvent.keyDown(box, { key: "Enter" })
    expect(screen.getByTestId("where").textContent).toBe("/settings")
  })

  it("changes the theme from the palette", () => {
    mount()
    press("k", { metaKey: true })
    fireEvent.click(screen.getByRole("option", { name: /Theme: Dark/ }))
    expect(localStorage.getItem("theme")).toBe("dark")
  })

  it("says when nothing matches", () => {
    mount()
    press("k", { metaKey: true })
    fireEvent.change(screen.getByRole("combobox"), { target: { value: "zzz" } })
    expect(screen.getByText(/No command matches/)).toBeTruthy()
  })

  it("shows the shortcut sheet on ? and from the palette", () => {
    mount()
    press("?", { shiftKey: true })
    expect(
      screen.getByRole("dialog", { name: "Keyboard shortcuts" })
    ).toBeTruthy()
    press("Escape")
    expect(
      screen.queryByRole("dialog", { name: "Keyboard shortcuts" })
    ).toBeNull()
    press("k", { metaKey: true })
    fireEvent.click(screen.getByRole("option", { name: /Keyboard shortcuts/ }))
    expect(
      screen.getByRole("dialog", { name: "Keyboard shortcuts" })
    ).toBeTruthy()
  })

  it("g then d goes to the desk", () => {
    mount()
    press("g")
    press("d")
    expect(screen.getByTestId("where").textContent).toBe("/")
  })
})

describe("HelpPopover", () => {
  it("opens on press, explains, and closes with Got it", () => {
    render(
      <HelpPopover
        title="Safe and kept"
        body="Work Span has already sealed."
        points={[{ mark: "!", text: "Rewind discards sealed work." }]}
      />
    )
    const trigger = screen.getByRole("button", {
      name: "What is “Safe and kept”?",
    })
    expect(trigger.getAttribute("aria-expanded")).toBe("false")
    fireEvent.click(trigger)
    expect(screen.getByText("Work Span has already sealed.")).toBeTruthy()
    expect(screen.getByText("Rewind discards sealed work.")).toBeTruthy()
    fireEvent.click(screen.getByRole("button", { name: "Got it" }))
    expect(screen.queryByText("Work Span has already sealed.")).toBeNull()
  })

  it("closes with Escape on the trigger", () => {
    render(<HelpPopover title="Restart" body="Runs the stage again." />)
    const trigger = screen.getByRole("button", { name: "What is “Restart”?" })
    fireEvent.click(trigger)
    fireEvent.keyDown(trigger, { key: "Escape" })
    expect(screen.queryByText("Runs the stage again.")).toBeNull()
  })
})
