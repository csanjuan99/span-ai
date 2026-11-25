import { cleanup, fireEvent, render, screen } from "@testing-library/react"
import { MemoryRouter } from "react-router"
import { afterEach, describe, expect, it, vi } from "vitest"

import { AppBar } from "@/components/app-bar/AppBar"
import { openPaletteEvent } from "@/lib/help"

function bar(at: string) {
  render(
    <MemoryRouter initialEntries={[at]}>
      <AppBar path="~/work/api" />
    </MemoryRouter>
  )
}

describe("AppBar", () => {
  afterEach(cleanup)

  it("marks the page you are on", () => {
    bar("/settings")
    expect(
      screen
        .getByRole("link", { name: "Settings" })
        .getAttribute("aria-current")
    ).toBe("page")
    expect(
      screen.getByRole("link", { name: "Desk" }).getAttribute("aria-current")
    ).toBeNull()
  })

  it("leaves both links unmarked on a run page", () => {
    bar("/spans/abc")
    expect(
      screen.getByRole("link", { name: "Desk" }).getAttribute("aria-current")
    ).toBeNull()
  })

  it("asks the palette to open when search is clicked", () => {
    const opened = vi.fn()
    window.addEventListener(openPaletteEvent, opened)
    bar("/")
    fireEvent.click(screen.getByRole("button", { name: /Search/ }))
    window.removeEventListener(openPaletteEvent, opened)
    expect(opened).toHaveBeenCalledTimes(1)
  })

  it("shows where the page is", () => {
    bar("/")
    expect(screen.getByText("~/work/api")).toBeTruthy()
  })
})
