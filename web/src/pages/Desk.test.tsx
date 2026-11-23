import { cleanup, render, screen, waitFor } from "@testing-library/react"
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
})
