import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react"
import { MemoryRouter } from "react-router"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

const getJSON = vi.fn()
const postForm = vi.fn()

vi.mock("@/lib/api", () => ({
  getJSON: (url: string) => getJSON(url),
  postForm: (url: string, body: unknown) => postForm(url, body),
  rememberServeToken: () => undefined,
  serveToken: () => "",
  startRun: () =>
    Promise.resolve({
      id: "s",
      stage: "intake",
      runs: [{ id: "s", stage: "intake", repo: "" }],
      partial: false,
      notes: "",
    }),
  agentList: () => [],
}))

import { Desk } from "@/pages/Desk"

afterEach(cleanup)

// The repository checkboxes measure themselves; jsdom has no ResizeObserver.
class NoResize {
  observe(): void {}
  unobserve(): void {}
  disconnect(): void {}
}
vi.stubGlobal("ResizeObserver", NoResize)

beforeEach(() => {
  getJSON.mockReset()
  postForm.mockReset()
  getJSON.mockImplementation((url: string) => {
    if (url.startsWith("/spans")) {
      return Promise.resolve({ spans: [] })
    }
    if (url.startsWith("/agents")) {
      return Promise.resolve({ agents: [] })
    }
    if (url.startsWith("/jira?path=")) {
      return Promise.resolve({ configured: true, email: "me@example.test", baseUrl: "https://salute.atlassian.net" })
    }
    return Promise.resolve({})
  })
})

describe("Desk", () => {
  it("does not ask for Jira credentials when the folder already has a session", async () => {
    render(
      <MemoryRouter>
        <Desk />
      </MemoryRouter>,
    )
    await waitFor(() => {
      expect(getJSON.mock.calls.some(([url]) => String(url).startsWith("/jira?path="))).toBe(true)
    })
    await waitFor(() => {
      expect(screen.queryByLabelText("Token")).toBeNull()
    })
    expect(screen.getByText(/me@example\.test/)).toBeTruthy()
    expect(screen.getByRole("button", { name: "Close session" })).toBeTruthy()
  })

  it("asks for credentials when the folder has no session", async () => {
    getJSON.mockImplementation((url: string) => {
      if (url.startsWith("/jira?path=")) {
        return Promise.resolve({ configured: false })
      }
      if (url.startsWith("/spans")) {
        return Promise.resolve({ spans: [] })
      }
      if (url.startsWith("/agents")) {
        return Promise.resolve({ agents: [] })
      }
      return Promise.resolve({})
    })
    render(
      <MemoryRouter>
        <Desk />
      </MemoryRouter>,
    )
    await waitFor(() => {
      expect(screen.getByLabelText("Token")).toBeTruthy()
    })
    expect(screen.getByText("No session for this folder.")).toBeTruthy()
    expect(screen.queryByRole("button", { name: "Close session" })).toBeNull()
  })

  it("keeps the repositories when Jira fails", async () => {
    getJSON.mockImplementation((url: string) => {
      if (url.startsWith("/repos?path=")) {
        return Promise.resolve({ repos: [{ name: "api" }, { name: "web" }] })
      }
      if (url.startsWith("/jira?path=")) {
        return Promise.reject(new Error("jira is down"))
      }
      if (url.startsWith("/spans")) {
        return Promise.resolve({ spans: [] })
      }
      return Promise.resolve({ agents: [] })
    })
    render(
      <MemoryRouter>
        <Desk />
      </MemoryRouter>,
    )
    fireEvent.click(screen.getByRole("button", { name: "Find repos" }))
    await waitFor(() => {
      expect(screen.getByText("api")).toBeTruthy()
    })
    expect(screen.getByText("web")).toBeTruthy()
    expect(screen.getByText("jira is down")).toBeTruthy()
  })

  it("keeps the Jira session when the repositories fail", async () => {
    let jiraCalls = 0
    getJSON.mockImplementation((url: string) => {
      if (url.startsWith("/repos?path=")) {
        return Promise.reject(new Error("the folder is gone"))
      }
      if (url.startsWith("/jira?path=")) {
        jiraCalls += 1
        return Promise.resolve(jiraCalls === 1 ? { configured: false } : { configured: true, email: "later@example.test", baseUrl: "https://x.test" })
      }
      if (url.startsWith("/spans")) {
        return Promise.resolve({ spans: [] })
      }
      return Promise.resolve({ agents: [] })
    })
    render(
      <MemoryRouter>
        <Desk />
      </MemoryRouter>,
    )
    await waitFor(() => {
      expect(jiraCalls).toBe(1)
    })
    fireEvent.click(screen.getByRole("button", { name: "Find repos" }))
    await waitFor(() => {
      expect(screen.getByText("the folder is gone")).toBeTruthy()
    })
    await waitFor(() => {
      expect(screen.getByText(/later@example\.test/)).toBeTruthy()
    })
  })
})
