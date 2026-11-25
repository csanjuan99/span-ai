import { fireEvent, render, screen, waitFor } from "@testing-library/react"
import { afterEach, describe, expect, it, vi } from "vitest"

import { QaTab } from "@/components/qa/QaTab"

function reply(routes: Record<string, unknown>) {
  vi.stubGlobal(
    "fetch",
    vi.fn((url: string) => {
      const key = Object.keys(routes).find((route) => url.includes(route))
      return Promise.resolve(
        new Response(JSON.stringify(key === undefined ? {} : routes[key]), {
          status: key === undefined ? 404 : 200,
        })
      )
    })
  )
}

afterEach(() => vi.unstubAllGlobals())

const failing = {
  run: "r1",
  state: "ok",
  verdict: "fail",
  headline: "The fast gate failed",
  exit: 10,
  gates: [
    {
      name: "fast",
      verdict: "fail",
      ran: true,
      exitCode: 1,
      durationMs: 6400,
      lines: 14,
      command: "go test ./...",
    },
  ],
  cases: 2,
}

describe("QaTab", () => {
  it("says a failing gate failed and offers its log and the risk form", async () => {
    reply({
      "/qa/cases": {
        known: true,
        cases: [
          {
            name: "TestRetry",
            result: "fail",
            path: "a_test.go",
            line: 5,
            message: "want 4s",
          },
        ],
      },
      "/qa": failing,
    })
    render(<QaTab id="r1" />)
    expect(await screen.findByText("The fast gate failed")).toBeTruthy()
    expect(screen.getByRole("button", { name: "Overview" })).toBeTruthy()
    expect(screen.getByRole("button", { name: "Cases" })).toBeTruthy()
    expect(screen.getByRole("button", { name: "Log" })).toBeTruthy()
    expect(screen.getByRole("button", { name: "Transcript" })).toBeTruthy()
    expect(screen.getByRole("button", { name: "Open log" })).toBeTruthy()
    expect(
      (
        screen.getByRole("button", {
          name: "Record the risk",
        }) as HTMLButtonElement
      ).disabled
    ).toBe(true)
    fireEvent.click(screen.getByRole("button", { name: "Cases" }))
    fireEvent.click(await screen.findByRole("button", { name: /TestRetry/ }))
    expect(screen.getByText("want 4s")).toBeTruthy()
  })

  it("never shows a pass for a run that held", async () => {
    reply({
      "/qa/cases": {
        known: false,
        cases: [],
        note: "The check printed no cases.",
      },
      "/qa": {
        ...failing,
        verdict: "held",
        state: "no_cases",
        headline: "It found 0 cases",
        gates: [],
        exit: 11,
      },
    })
    render(<QaTab id="r1" />)
    await waitFor(() =>
      expect(screen.getByText("It found 0 cases")).toBeTruthy()
    )
    expect(screen.queryByText("Pass")).toBeNull()
    expect(screen.getByText("Held")).toBeTruthy()
  })
})
