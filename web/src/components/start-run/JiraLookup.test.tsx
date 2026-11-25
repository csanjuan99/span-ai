import { cleanup, fireEvent, render, screen } from "@testing-library/react"
import { afterEach, describe, expect, it, vi } from "vitest"

import { JiraLookup, type JiraProps } from "@/components/start-run/JiraLookup"
import { noSession } from "@/lib/jira-session"

afterEach(cleanup)

const connected = {
  configured: true,
  email: "carlos@acme.com",
  baseUrl: "https://acme.atlassian.net",
}

function props(over: Partial<JiraProps>): JiraProps {
  return {
    session: connected,
    issue: "SPAN-503",
    onIssue: vi.fn(),
    url: "",
    onUrl: vi.fn(),
    email: "",
    onEmail: vi.fn(),
    token: "",
    onToken: vi.fn(),
    stories: [],
    searched: "",
    lookup: { phase: "idle" },
    note: "",
    onLoadOne: vi.fn(),
    onSearch: vi.fn(),
    onSaveSession: vi.fn(),
    onPick: vi.fn(),
    onPasteInstead: vi.fn(),
    ...over,
  }
}

describe("JiraLookup", () => {
  it("keeps the key and offers an optional connection before Jira is connected", () => {
    const onSaveSession = vi.fn()
    render(<JiraLookup {...props({ session: noSession, onSaveSession })} />)
    expect(screen.getByLabelText("Story key")).toHaveProperty(
      "value",
      "SPAN-503"
    )
    expect(screen.getByRole("button", { name: "Look up" })).toHaveProperty(
      "disabled",
      true
    )
    expect(screen.getByText("Optional")).toBeTruthy()
    fireEvent.click(
      screen.getByRole("button", { name: "Connect and look up SPAN-503" })
    )
    expect(onSaveSession).toHaveBeenCalledTimes(1)
  })

  it("names the host while it asks and what it read when it found the story", () => {
    const { rerender } = render(
      <JiraLookup
        {...props({ lookup: { phase: "asking", key: "SPAN-503" } })}
      />
    )
    expect(screen.getByRole("status").textContent).toContain(
      "acme.atlassian.net"
    )
    rerender(
      <JiraLookup
        {...props({
          lookup: {
            phase: "found",
            story: {
              key: "SPAN-503",
              title: "Return a clear error",
              status: "To Do",
              type: "Story",
              url: "https://acme.atlassian.net/browse/SPAN-503",
              acceptance:
                "- A refund larger than the charge is refused\n- The error says why",
            },
          },
        })}
      />
    )
    expect(screen.getByText("Return a clear error")).toBeTruthy()
    expect(screen.getByText("To Do")).toBeTruthy()
    expect(screen.getByText("AC2")).toBeTruthy()
    expect(
      screen.getByRole("link", { name: "Open in Jira" }).getAttribute("href")
    ).toBe("https://acme.atlassian.net/browse/SPAN-503")
  })

  it("turns a 404 into a search by words and a 403 into a link to Jira", () => {
    const onSearch = vi.fn()
    const { rerender } = render(
      <JiraLookup
        {...props({
          onSearch,
          lookup: {
            phase: "failed",
            key: "SPAN-5030",
            reason: "jira answered 404",
          },
        })}
      />
    )
    expect(
      screen.getByText("SPAN-5030 is not in the project SPAN")
    ).toBeTruthy()
    fireEvent.click(screen.getByRole("button", { name: "Search by text" }))
    expect(onSearch).toHaveBeenCalledWith("SPAN-5030")
    rerender(
      <JiraLookup
        {...props({
          lookup: {
            phase: "failed",
            key: "SPAN-512",
            reason: "jira rejected the account",
          },
        })}
      />
    )
    expect(
      screen.getByText("Your Jira account cannot open SPAN-512")
    ).toBeTruthy()
    expect(
      screen
        .getByRole("link", { name: "Open SPAN-512 in Jira" })
        .getAttribute("href")
    ).toBe("https://acme.atlassian.net/browse/SPAN-512")
  })

  it("retries an offline lookup and lists the stories a search found", () => {
    const onLoadOne = vi.fn()
    const onPick = vi.fn()
    const { rerender } = render(
      <JiraLookup
        {...props({
          onLoadOne,
          lookup: {
            phase: "failed",
            key: "SPAN-503",
            reason: "jira did not respond",
          },
        })}
      />
    )
    expect(
      screen.getByText("Span cannot reach acme.atlassian.net")
    ).toBeTruthy()
    fireEvent.click(screen.getByRole("button", { name: "Retry look up" }))
    expect(onLoadOne).toHaveBeenCalledTimes(1)
    rerender(
      <JiraLookup
        {...props({
          onPick,
          searched: "refund",
          stories: [
            {
              key: "SPAN-503",
              summary: "Return a clear error",
              status: "To Do",
            },
            { key: "SPAN-437", summary: "Email customers", status: "Done" },
          ],
        })}
      />
    )
    expect(screen.getByText(/2 stories/).textContent).toContain("“refund”")
    fireEvent.click(screen.getAllByRole("button", { name: "Use" })[1])
    expect(onPick).toHaveBeenCalledWith("SPAN-437")
  })
})
