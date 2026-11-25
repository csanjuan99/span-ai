import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react"
import { MemoryRouter } from "react-router"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

const getJSON = vi.fn()
const postForm = vi.fn()
const startRun = vi.fn()

vi.mock("@/lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/api")>()
  return {
    getJSON: (url: string) => getJSON(url),
    postForm: (url: string, body: unknown) => postForm(url, body),
    rememberServeToken: () => undefined,
    UnauthorizedError: actual.UnauthorizedError,
    serveToken: () => "",
    startRun: (form: FormData) => startRun(form),
    agentList: () => [],
    branchChoice: actual.branchChoice,
    basePlaceholder: actual.basePlaceholder,
    spanHeadsOf: actual.spanHeadsOf,
  }
})

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
  startRun.mockReset()
  startRun.mockImplementation(() =>
    Promise.resolve({
      id: "s",
      stage: "intake",
      runs: [{ id: "s", stage: "intake", repo: "" }],
      partial: false,
      notes: "",
    })
  )
  getJSON.mockImplementation((url: string) => {
    if (url.startsWith("/branches?path=")) {
      return Promise.resolve({
        branches: ["dev", "feat/SAL-3"],
        default: "dev",
      })
    }
    if (url.startsWith("/spans")) {
      return Promise.resolve({ spans: [] })
    }
    if (url.startsWith("/agents")) {
      return Promise.resolve({ agents: [] })
    }
    if (url.startsWith("/jira?path=")) {
      return Promise.resolve({
        configured: true,
        email: "me@example.test",
        baseUrl: "https://salute.atlassian.net",
      })
    }
    return Promise.resolve({})
  })
})

describe("Desk", () => {
  it("does not ask for Jira credentials when the folder already has a session", async () => {
    render(
      <MemoryRouter>
        <Desk />
      </MemoryRouter>
    )
    await waitFor(() => {
      expect(
        getJSON.mock.calls.some(([url]) =>
          String(url).startsWith("/jira?path=")
        )
      ).toBe(true)
    })
    await waitFor(() => {
      expect(screen.queryByLabelText("Token")).toBeNull()
    })
    expect(screen.getByText(/me@example\.test/)).toBeTruthy()
    // The session is closed from settings, which stay out of the way until asked for.
    expect(screen.queryByRole("button", { name: "Close session" })).toBeNull()
    fireEvent.click(screen.getByRole("button", { name: "Show" }))
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
      </MemoryRouter>
    )
    fireEvent.click(await screen.findByRole("button", { name: /Connect Jira/ }))
    await waitFor(() => {
      expect(screen.getByLabelText("Token")).toBeTruthy()
    })
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
      </MemoryRouter>
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
        return Promise.resolve(
          jiraCalls === 1
            ? { configured: false }
            : {
                configured: true,
                email: "later@example.test",
                baseUrl: "https://x.test",
              }
        )
      }
      if (url.startsWith("/spans")) {
        return Promise.resolve({ spans: [] })
      }
      return Promise.resolve({ agents: [] })
    })
    render(
      <MemoryRouter>
        <Desk />
      </MemoryRouter>
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

  it("offers the repository's branches as the base and posts the one chosen", async () => {
    render(
      <MemoryRouter>
        <Desk />
      </MemoryRouter>
    )
    const field = screen.getByLabelText("Base branch")
    await waitFor(() => {
      expect(field.getAttribute("placeholder")).toBe("dev")
    })
    await waitFor(() => {
      expect(document.querySelectorAll("#base-branches option").length).toBe(2)
    })
    expect(
      getJSON.mock.calls.some(([url]) => String(url) === "/branches?path=.")
    ).toBe(true)
    fireEvent.change(field, { target: { value: "  feat/SAL-3 " } })
    fireEvent.click(screen.getByRole("button", { name: "Configure and run" }))
    await waitFor(() => {
      expect(startRun).toHaveBeenCalled()
    })
    const form: unknown = startRun.mock.calls[0]?.[0]
    expect(form instanceof FormData ? form.get("base") : null).toBe(
      "feat/SAL-3"
    )
  })

  it("posts no base when the field is left empty, so the default applies", async () => {
    render(
      <MemoryRouter>
        <Desk />
      </MemoryRouter>
    )
    fireEvent.click(screen.getByRole("button", { name: "Configure and run" }))
    await waitFor(() => {
      expect(startRun).toHaveBeenCalled()
    })
    const form: unknown = startRun.mock.calls[0]?.[0]
    expect(form instanceof FormData ? form.has("base") : true).toBe(false)
  })

  it("opens on the runs, and the start form waits behind New run", async () => {
    getJSON.mockImplementation((url: string) =>
      url.startsWith("/spans")
        ? Promise.resolve({
            spans: [
              {
                id: "a",
                verdict: "hold",
                stage: "merge",
                issue: "SAL-1",
                title: "Pay",
                reason: "conflict",
              },
            ],
          })
        : Promise.resolve({})
    )
    render(
      <MemoryRouter>
        <Desk />
      </MemoryRouter>
    )
    expect(
      await screen.findByRole("region", { name: "Waiting for me" })
    ).toBeTruthy()
    expect(screen.queryByLabelText("Story")).toBeNull()
    fireEvent.click(screen.getByRole("button", { name: "New run" }))
    expect(await screen.findByLabelText("Story")).toBeTruthy()
  })

  it("starts from the sample brief when there are no runs", async () => {
    render(
      <MemoryRouter>
        <Desk />
      </MemoryRouter>
    )
    fireEvent.click(
      await screen.findByRole("button", { name: "Start with the sample brief" })
    )
    const story = (await screen.findByLabelText("Story")) as HTMLTextAreaElement
    expect(story.value).toContain("Add a /healthz endpoint")
    expect(startRun).not.toHaveBeenCalled()
  })
})
