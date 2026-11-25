import { act, cleanup, fireEvent, render, screen } from "@testing-library/react"
import { MemoryRouter, Route, Routes, useLocation } from "react-router"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

import { CommandPalette, HelpPopover } from "@/components/help"
import { ThemeProvider } from "@/components/theme-provider"
import { setRunContext } from "@/lib/help"

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

describe("This run", () => {
  afterEach(() => {
    setRunContext(null)
  })

  function give(resume = vi.fn(), stop = vi.fn()) {
    act(() => {
      setRunContext({
        label: "SPAN-482",
        commands: [
          {
            id: "resume",
            title: "Resume verify",
            hint: "Runs the stage again.",
            key: "r",
            run: resume,
          },
          {
            id: "stop",
            title: "Stop now",
            hint: "Ends the run.",
            key: "s",
            shift: true,
            run: stop,
          },
        ],
      })
    })
    return { resume, stop }
  }

  it("lists what the run page offers first, under its name", () => {
    mount()
    give()
    press("k", { metaKey: true })
    expect(screen.getByText("This run · SPAN-482")).toBeTruthy()
    const options = screen.getAllByRole("option")
    expect(options[0]?.textContent).toContain("Resume verify")
  })

  it("runs an action from its key, and Shift+S only with Shift", () => {
    mount()
    const { resume, stop } = give()
    press("r")
    expect(resume).toHaveBeenCalledTimes(1)
    press("s")
    expect(stop).not.toHaveBeenCalled()
    press("S", { shiftKey: true })
    expect(stop).toHaveBeenCalledTimes(1)
  })

  it("runs an action picked in the palette and closes it", () => {
    mount()
    const { resume } = give()
    press("k", { metaKey: true })
    fireEvent.click(screen.getByRole("option", { name: /Resume verify/ }))
    expect(resume).toHaveBeenCalledTimes(1)
    expect(screen.queryByRole("dialog", { name: "Command palette" })).toBeNull()
  })

  it("lists its keys on the shortcut sheet", () => {
    mount()
    give()
    press("?")
    expect(screen.getByText("On this run · SPAN-482")).toBeTruthy()
    expect(screen.getByText("Stop now")).toBeTruthy()
  })

  it("leaves g then d to the Desk, not to a run key", () => {
    mount()
    const { resume } = give()
    press("g")
    press("d")
    expect(resume).not.toHaveBeenCalled()
    expect(screen.getByTestId("where").textContent).toBe("/")
  })

  it("does nothing once the page clears its context", () => {
    mount()
    const { resume } = give()
    act(() => {
      setRunContext(null)
    })
    press("r")
    expect(resume).not.toHaveBeenCalled()
  })
})
