import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react"
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

afterEach(() => {
  cleanup()
  vi.unstubAllGlobals()
})

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
    expect(screen.getByText("No cases found")).toBeTruthy()
    expect(screen.getByText("A pass. Nothing was proven.")).toBeTruthy()
    expect(
      screen.getByRole("button", { name: "Accept with no tests" })
    ).toBeTruthy()
  })

  it.each([
    ["never_ran", "Gate never ran", "Any pass or fail."],
    ["unsealed", "Sealed build missing", "A pass for this code."],
  ])(
    "reads the %s state as an honest band, with no accept shortcut",
    async (state, band, notShown) => {
      reply({
        "/qa/cases": { known: false, cases: [] },
        "/qa": {
          ...failing,
          state,
          headline: "Span cannot call this a pass",
          reason: "Nothing ran against a sealed build.",
          kept: "The branch as it is.",
          next: "Resume the run.",
          gates: [],
          exit: 11,
        },
      })
      render(<QaTab id="r1" />)
      expect(await screen.findByText(band)).toBeTruthy()
      expect(screen.getByText(new RegExp(notShown))).toBeTruthy()
      expect(screen.getByText("The branch as it is.")).toBeTruthy()
      expect(screen.getByText("Resume the run.")).toBeTruthy()
      expect(screen.queryByRole("button", { name: /Accept with/ })).toBeNull()
    }
  )

  it("offers the failing cases and the full log from the Next panel", async () => {
    reply({
      "/qa/cases": { known: true, cases: [] },
      "/qa": {
        ...failing,
        command: "go test ./...",
        exitCode: 1,
        counts: { passed: 41, failed: 2, skipped: 0, seconds: 6.4 },
        kept: "The log and the counts.",
      },
    })
    render(<QaTab id="r1" />)
    expect(await screen.findByText("Next")).toBeTruthy()
    expect(screen.getAllByText("exit 1").length).toBeGreaterThan(0)
    expect(
      screen.getByText("The 2 failing cases with their output.")
    ).toBeTruthy()
    expect(screen.getByRole("button", { name: "Open full log" })).toBeTruthy()
    fireEvent.click(
      screen.getByRole("button", { name: "See the 2 failing cases" })
    )
    expect(screen.queryByText("Next")).toBeNull()
  })

  it("names the failing cases without repeating the word when the run has no counts", async () => {
    reply({
      "/qa/cases": { known: true, cases: [] },
      "/qa": {
        ...failing,
        command: "go test ./...",
        exitCode: 1,
        counts: null,
      },
    })
    render(<QaTab id="r1" />)
    expect(await screen.findByText("Next")).toBeTruthy()
    expect(
      screen.getByText("The failing cases with their output.")
    ).toBeTruthy()
    expect(
      screen.getByRole("button", { name: "See the failing cases" })
    ).toBeTruthy()
  })

  it("offers to rerun the failing cases with what it does and keeps", async () => {
    const failed = (name: string) => ({
      name,
      gate: "fast",
      result: "fail",
      elapsedMs: 20,
      path: "retry_test.go",
      file: "retry_test.go",
      line: 58,
      message: "",
      output: [],
      isNew: false,
      history: [],
    })
    reply({
      "/qa/cases": {
        known: true,
        cases: [failed("TestRetryBackoff"), failed("TestRetryStopsAfterMax")],
      },
      "/qa": failing,
    })
    render(<QaTab id="r1" />)
    fireEvent.click(await screen.findByRole("button", { name: "Cases" }))
    expect(
      await screen.findByRole("button", { name: "Rerun 2 failed cases" })
    ).toBeTruthy()
    expect(
      screen.getByText("Runs the 2 failing cases again on the branch checkout.")
    ).toBeTruthy()
    expect(
      screen.getByText("The first result stays in the history.")
    ).toBeTruthy()
  })
})
