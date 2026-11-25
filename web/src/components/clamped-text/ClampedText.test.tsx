import { cleanup, render, screen } from "@testing-library/react"
import { afterEach, describe, expect, it } from "vitest"

import { ClampedText } from "@/components/clamped-text/ClampedText"

afterEach(cleanup)

describe("ClampedText", () => {
  it("shows a short line whole, without color codes", () => {
    render(<ClampedText text={"[0;36mmerged[0m"} />)
    expect(screen.getByText("merged")).toBeTruthy()
    expect(document.querySelector("details")).toBeNull()
  })

  it("folds a long line, keeping the start visible and all of it one click away", () => {
    const long = `${"a".repeat(300)}END`
    render(<ClampedText text={long} />)
    const fold = document.querySelector("details")
    expect(fold).not.toBeNull()
    expect(fold?.open).toBe(false)
    expect(fold?.querySelector("summary")?.textContent).toContain("…")
    expect(fold?.querySelector("summary")?.textContent).not.toContain("END")
    expect(fold?.textContent).toContain("END")
  })
})
