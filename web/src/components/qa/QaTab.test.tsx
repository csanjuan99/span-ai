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

  const twoCases = {
    known: true,
    total: 43,
    passed: 41,
    failed: 2,
    new: 8,
    cases: [
      {
        name: "TestRetryBackoff",
        result: "fail",
        elapsedMs: 20,
        path: "webhook/retry_test.go",
        file: "retry_test.go",
        line: 58,
        output: ["delay 2s, want 4s"],
        new: true,
      },
      {
        name: "TestSendOK",
        result: "pass",
        elapsedMs: 10,
        path: "webhook/client_test.go",
        file: "client_test.go",
        line: 21,
      },
    ],
  }

  async function openCases() {
    reply({ "/qa/cases": twoCases, "/qa": failing })
    render(<QaTab id="r1" />)
    fireEvent.click(await screen.findByRole("button", { name: "Cases" }))
    await screen.findByText("TestRetryBackoff")
  }

  it("lists the cases in a table with where each one is and shows a failure's output", async () => {
    await openCases()
    for (const column of ["Result", "Case", "File:line", "Time", "New"]) {
      expect(screen.getAllByRole("columnheader", { name: column }).length).toBe(
        2
      )
    }
    expect(screen.getByText("retry_test.go:58")).toBeTruthy()
    expect(screen.getByText("delay 2s, want 4s")).toBeTruthy()
    expect(
      screen.getByText(
        "Failures first within each file. 43 cases, 41 passed, 2 failed."
      )
    ).toBeTruthy()
    expect(screen.queryByRole("button", { name: "Rerun this case" })).toBeNull()
    fireEvent.click(screen.getByRole("button", { name: "Open this case" }))
    expect(screen.getByRole("button", { name: "Rerun this case" })).toBeTruthy()
  })

  it("lists every case in one table when grouping by file is switched off", async () => {
    await openCases()
    const group = screen.getByRole("switch", { name: "Group by file" })
    expect(group.getAttribute("aria-checked")).toBe("true")
    expect(screen.getByText("webhook/client_test.go")).toBeTruthy()
    fireEvent.click(group)
    await waitFor(() => expect(screen.getAllByRole("table").length).toBe(1))
    expect(screen.queryByText("webhook/client_test.go")).toBeNull()
    const asked = vi.mocked(fetch).mock.calls.map(([url]) => String(url))
    expect(asked.at(-1)).not.toContain("group=file")
    expect(
      screen.getByText("Failures first. 43 cases, 41 passed, 2 failed.")
    ).toBeTruthy()
  })
})
