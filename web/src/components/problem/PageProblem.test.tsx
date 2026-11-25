import { cleanup, render, screen } from "@testing-library/react"
import { MemoryRouter } from "react-router"
import { afterEach, describe, expect, it } from "vitest"

import { PageProblem } from "@/components/problem/PageProblem"

afterEach(cleanup)

describe("PageProblem", () => {
  it("says the server is unreachable, what is kept and what to do", () => {
    render(
      <MemoryRouter>
        <PageProblem problem="unreachable" id="x" />
      </MemoryRouter>
    )
    expect(
      screen.getByRole("region", { name: "Server unreachable" })
    ).toBeTruthy()
    expect(
      screen.getByText("Your runs, branches and files are untouched.")
    ).toBeTruthy()
    expect(
      screen.getByRole("button", { name: "Retry connection" })
    ).toBeTruthy()
    expect(screen.getByText("span serve")).toBeTruthy()
  })

  it("names the run that is not on this desk and links back", () => {
    render(
      <MemoryRouter>
        <PageProblem problem="notfound" id="SPAN-499" />
      </MemoryRouter>
    )
    expect(
      screen.getByRole("heading", { name: "Run SPAN-499 is not on this desk" })
    ).toBeTruthy()
    expect(
      screen.getByRole("link", { name: "Back to Desk" }).getAttribute("href")
    ).toBe("/")
  })
})
