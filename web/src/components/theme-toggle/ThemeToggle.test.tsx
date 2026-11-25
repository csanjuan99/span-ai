import { cleanup, fireEvent, render, screen } from "@testing-library/react"
import { afterEach, beforeEach, describe, expect, it } from "vitest"

import { ThemeProvider } from "@/components/theme-provider"
import { ThemeToggle } from "@/components/theme-toggle/ThemeToggle"

function mount() {
  render(
    <ThemeProvider>
      <ThemeToggle />
    </ThemeProvider>
  )
}

describe("ThemeToggle", () => {
  afterEach(cleanup)

  beforeEach(() => {
    localStorage.clear()
    window.matchMedia = ((query: string) => ({
      matches: false,
      media: query,
      addEventListener: () => undefined,
      removeEventListener: () => undefined,
    })) as unknown as typeof window.matchMedia
  })

  it("draws nothing when no provider is mounted, so a page can render on its own", () => {
    const { container } = render(<ThemeToggle />)
    expect(container.firstChild).toBeNull()
  })

  it("starts by following the system and offers light next", () => {
    mount()
    expect(
      screen.getByRole("button", { name: "Theme: system. Switch to light" })
    ).toBeTruthy()
  })

  it("cycles system, light, dark and puts the theme on the page", () => {
    mount()
    fireEvent.click(screen.getByRole("button", { name: /Switch to light/ }))
    expect(document.documentElement.classList.contains("light")).toBe(true)
    fireEvent.click(screen.getByRole("button", { name: /Switch to dark/ }))
    expect(document.documentElement.classList.contains("dark")).toBe(true)
    expect(localStorage.getItem("theme")).toBe("dark")
    fireEvent.click(screen.getByRole("button", { name: /Switch to system/ }))
    expect(screen.getByRole("button", { name: /Theme: system/ })).toBeTruthy()
  })
})
