import { cleanup, fireEvent, render, screen } from "@testing-library/react"
import { afterEach, beforeEach, describe, expect, it } from "vitest"

import { WhatsNew } from "@/components/help/WhatsNew"

beforeEach(() => localStorage.clear())
afterEach(cleanup)

describe("WhatsNew", () => {
  it("shows the note once and remembers the dismissal", () => {
    const { unmount } = render(
      <WhatsNew version="0.9.0" items={["A settings page."]} />
    )
    expect(screen.getByText("Span updated to 0.9.0")).toBeTruthy()
    fireEvent.click(screen.getByRole("button", { name: "Dismiss" }))
    expect(screen.queryByText("Span updated to 0.9.0")).toBeNull()
    unmount()
    render(<WhatsNew version="0.9.0" items={["A settings page."]} />)
    expect(screen.queryByText("Span updated to 0.9.0")).toBeNull()
  })

  it("shows again for a newer version", () => {
    localStorage.setItem("span-whats-new", "0.9.0")
    render(<WhatsNew version="0.10.0" items={["Something new."]} />)
    expect(screen.getByText("Span updated to 0.10.0")).toBeTruthy()
  })

  it("shows nothing without items", () => {
    render(<WhatsNew version="1.0.0" items={[]} />)
    expect(screen.queryByRole("complementary")).toBeNull()
  })
})
