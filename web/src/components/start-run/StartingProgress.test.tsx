import { cleanup, render, screen } from "@testing-library/react"
import { afterEach, describe, expect, it } from "vitest"

import { StartingProgress } from "@/components/start-run/StartingProgress"

afterEach(cleanup)

describe("StartingProgress", () => {
  it("says the folder is being checked until it has been", () => {
    render(<StartingProgress pending={{ runs: 1, checkedAt: "" }} />)
    expect(screen.getByText("Checking the folder")).toBeTruthy()
    expect(screen.queryByText("Checked the folder")).toBeNull()
    expect(
      screen.getByRole("heading", { name: "Starting the run" })
    ).toBeTruthy()
  })

  it("lists the steps reached, with the time the folder passed", () => {
    render(<StartingProgress pending={{ runs: 2, checkedAt: "14:51:02" }} />)
    expect(screen.getByText("Checked the folder")).toBeTruthy()
    expect(screen.getByText("14:51:02")).toBeTruthy()
    expect(screen.getByText("Starting intake in each repository")).toBeTruthy()
    expect(
      screen.getByRole("heading", { name: "Starting 2 runs" })
    ).toBeTruthy()
  })

  it("shows the engine's ten stages with intake first in line", () => {
    render(<StartingProgress pending={{ runs: 1, checkedAt: "" }} />)
    const rail = screen.getByRole("list", { name: "Stages" })
    expect(rail.querySelectorAll("li").length).toBe(10)
    expect(rail.querySelector("li")?.textContent).toContain("intake")
  })
})
