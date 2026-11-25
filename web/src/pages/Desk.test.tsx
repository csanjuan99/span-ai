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
    getPrecheck: (url: string) => getJSON(url),
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

import { UnauthorizedError } from "@/lib/api"
import { Desk } from "@/pages/Desk"

afterEach(cleanup)

// The repository checkboxes measure themselves; jsdom has no ResizeObserver.
class NoResize {
  observe(): void {}
  unobserve(): void {}
  disconnect(): void {}
}
vi.stubGlobal("ResizeObserver", NoResize)
// jsdom has no layout, so a refusal asking to be scrolled into view has nothing to do.
Element.prototype.scrollIntoView = vi.fn()

/** What starting one run answers: the first run, intake in flight. */
const oneRunStarted = {
  id: "s",
  stage: "intake",
  runs: [{ id: "s", stage: "intake", repo: "" }],
  partial: false,
  notes: "",
}

beforeEach(() => {
  getJSON.mockReset()
  postForm.mockReset()
  startRun.mockReset()
  startRun.mockImplementation(() => Promise.resolve(oneRunStarted))
  getJSON.mockImplementation((url: string) => {
    if (url.startsWith("/start/precheck")) {
      return Promise.resolve({ ok: true })
    }
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

/** The empty desk shows the first-run card; the start form opens behind "Paste a Jira story". */
async function openForm() {
  fireEvent.click(
    await screen.findByRole("button", { name: "Paste a Jira story" })
  )
}

/** The folder shows as a chip; the repository search lives behind "Change folder". */
async function openFolder() {
  fireEvent.click(await screen.findByRole("button", { name: "Change folder" }))
}

/** Routes the desk's reads by URL prefix; an unlisted `/spans` is empty and the rest read as no agents. */
function mockRoutes(routes: Record<string, () => Promise<unknown>>) {
  getJSON.mockImplementation((url: string) => {
    const hit = Object.keys(routes).find((prefix) => url.startsWith(prefix))
    if (hit !== undefined) {
      return routes[hit]!()
    }
    return Promise.resolve(
      url.startsWith("/spans") ? { spans: [] } : { agents: [] }
    )
  })
}

function renderDesk() {
  render(
    <MemoryRouter>
      <Desk />
    </MemoryRouter>
  )
}

describe("Desk", () => {
  it("does not call a desk empty when the list never loaded", async () => {
    mockRoutes({
      "/spans": () => Promise.reject(new Error("the desk could not load")),
      "/jira?path=": () => Promise.resolve({ configured: false }),
    })
    renderDesk()
    expect(await screen.findByText("the desk could not load")).toBeTruthy()
    expect(screen.queryByText("No runs yet")).toBeNull()
    expect(
      screen.queryByRole("button", { name: /Start with the sample brief/ })
    ).toBeNull()
  })

  it("does not ask for Jira credentials when the folder already has a session", async () => {
    renderDesk()
    await openForm()
    expect(await screen.findByText(/me@example\.test/)).toBeTruthy()
    expect(screen.queryByLabelText("API token")).toBeNull()
    // The session is closed from Settings, which stay out of the way of the start form.
    expect(screen.queryByRole("button", { name: "Close session" })).toBeNull()
  })

  it("asks for credentials when the folder has no session", async () => {
    mockRoutes({ "/jira?path=": () => Promise.resolve({ configured: false }) })
    renderDesk()
    await openForm()
    fireEvent.click(await screen.findByRole("button", { name: /^Connect$/ }))
    await waitFor(() => {
      expect(screen.getByLabelText("API token")).toBeTruthy()
    })
    expect(screen.queryByRole("button", { name: "Close session" })).toBeNull()
  })

  it("keeps the repositories when Jira fails", async () => {
    mockRoutes({
      "/repos?path=": () =>
        Promise.resolve({ repos: [{ name: "api" }, { name: "web" }] }),
      "/jira?path=": () => Promise.reject(new Error("jira is down")),
      "/spans": () => Promise.resolve({ spans: [] }),
    })
    renderDesk()
    await openForm()
    await openFolder()
    fireEvent.click(await screen.findByRole("button", { name: "Find repos" }))
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
    renderDesk()
    await openForm()
    await waitFor(() => {
      expect(jiraCalls).toBe(1)
    })
    await openFolder()
    fireEvent.click(await screen.findByRole("button", { name: "Find repos" }))
    await waitFor(() => {
      expect(screen.getByText("the folder is gone")).toBeTruthy()
    })
    await waitFor(() => {
      expect(screen.getByText(/later@example\.test/)).toBeTruthy()
    })
  })

  it("offers the repository's branches as the base and posts the one chosen", async () => {
    renderDesk()
    await openForm()
    const field = await screen.findByLabelText("Base branch")
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
    fireEvent.click(await screen.findByRole("button", { name: "Start run" }))
    await waitFor(() => {
      expect(startRun).toHaveBeenCalled()
    })
    const form: unknown = startRun.mock.calls[0]?.[0]
    expect(form instanceof FormData ? form.get("base") : null).toBe(
      "feat/SAL-3"
    )
  })

  it("posts no base when the field is left empty, so the default applies", async () => {
    renderDesk()
    await openForm()
    fireEvent.click(await screen.findByRole("button", { name: "Start run" }))
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
    renderDesk()
    expect(
      await screen.findByRole("region", { name: "Waiting for me" })
    ).toBeTruthy()
    expect(
      screen.queryByRole("form", { name: "Start a run", hidden: true })
    ).toBeNull()
    fireEvent.click(screen.getByRole("button", { name: "New run" }))
    expect(
      await screen.findByRole("form", { name: "Start a run" })
    ).toBeTruthy()
  })

  it("starts the sample brief in one click when there are no runs", async () => {
    renderDesk()
    fireEvent.click(
      await screen.findByRole("button", { name: "Start with the sample brief" })
    )
    await waitFor(() => expect(startRun).toHaveBeenCalledTimes(1))
    const sent = startRun.mock.calls[0]?.[0] as FormData
    expect(sent.get("story")).toContain("Add a /healthz endpoint")
    expect(sent.get("workspace")).toBe(".")
  })

  it("shows only the first-run card on an empty desk until a story is pasted", async () => {
    renderDesk()
    await screen.findByRole("button", { name: "Start with the sample brief" })
    const form = () =>
      screen.getByRole("form", { name: "Start a run", hidden: true })
    expect(form().closest("[hidden]")).not.toBeNull()
    fireEvent.click(screen.getByRole("button", { name: "Paste a Jira story" }))
    expect(form().closest("[hidden]")).toBeNull()
  })

  it("shows what the saved settings will do before Start is pressed", async () => {
    getJSON.mockImplementation((url: string) => {
      if (url.startsWith("/start/what-will-happen")) {
        return Promise.resolve({
          base: "main",
          chosenAgent: "claude",
          stages: { merge: "hold", open: "auto" },
          checks: ["go test ./..."],
        })
      }
      if (url.startsWith("/spans")) {
        return Promise.resolve({ spans: [] })
      }
      return Promise.resolve({ agents: [] })
    })
    renderDesk()
    await openForm()
    expect(await screen.findByText("claude writes the stages.")).toBeTruthy()
    expect(screen.getByText(/merge wait for your yes/)).toBeTruthy()
    expect(screen.getByText("go test ./...")).toBeTruthy()
    expect(
      screen.getByText("Nothing on main and none of your files.")
    ).toBeTruthy()
  })

  it("offers to shape a short pasted story first", async () => {
    renderDesk()
    await openForm()
    expect(screen.queryByText("This story is short")).toBeNull()
    fireEvent.click(await screen.findByRole("button", { name: "Pasted text" }))
    fireEvent.change(screen.getByLabelText("Story text"), {
      target: { value: "Refunds should not be larger than the charge." },
    })
    expect(await screen.findByText("This story is short")).toBeTruthy()
    expect(screen.getByRole("button", { name: /Shape it first/ })).toBeTruthy()
  })

  it("locks the source when a sealed story arrives from Shape a story", async () => {
    render(
      <MemoryRouter
        initialEntries={[
          {
            pathname: "/",
            state: {
              shape: {
                id: "sh-1",
                title: "Retry a failed delivery",
                workspace: "/w",
                story: "# Retry",
                issue: "",
                repo: "api",
              },
            },
          },
        ]}
      >
        <Desk />
      </MemoryRouter>
    )
    expect(await screen.findByText("Came from Shape a story")).toBeTruthy()
    expect(screen.getByText("Retry a failed delivery")).toBeTruthy()
    expect(screen.queryByRole("button", { name: "Pasted text" })).toBeNull()
    fireEvent.click(await screen.findByRole("button", { name: "Start run" }))
    await waitFor(() => expect(startRun).toHaveBeenCalled())
    const sent = startRun.mock.calls[0]?.[0] as FormData
    expect(sent.get("story")).toBe("# Retry")
    expect(sent.get("workspace")).toBe("/w")
    expect(sent.get("repo")).toBe("api")
  })

  it("names the missing base branch and starts nothing when the precheck refuses it", async () => {
    getJSON.mockImplementation((url: string) =>
      url.startsWith("/start/precheck")
        ? Promise.resolve({
            number: 15,
            code: "base_missing",
            message: 'branch "develop" is not in ws',
            base: "develop",
          })
        : Promise.resolve({ spans: [], agents: [] })
    )
    renderDesk()
    await openForm()
    fireEvent.click(await screen.findByRole("button", { name: "Start run" }))
    expect(
      await screen.findByText("Branch develop does not exist")
    ).toBeTruthy()
    expect(startRun).not.toHaveBeenCalled()
  })

  it("shows the token refusal when the precheck is refused for its token", async () => {
    mockRoutes({
      "/start/precheck": () => Promise.reject(new UnauthorizedError()),
      "/spans": () => Promise.resolve({ spans: [] }),
    })
    renderDesk()
    await openForm()
    fireEvent.click(await screen.findByRole("button", { name: "Start run" }))
    expect(await screen.findByText(/needs its access token/)).toBeTruthy()
    expect(startRun).not.toHaveBeenCalled()
  })

  it("still starts the run when the precheck route is missing, and says why", async () => {
    mockRoutes({
      "/start/precheck": () => Promise.reject(new Error("no precheck here")),
      "/spans": () => Promise.resolve({ spans: [] }),
    })
    renderDesk()
    await openForm()
    fireEvent.click(await screen.findByRole("button", { name: "Start run" }))
    await waitFor(() => expect(startRun).toHaveBeenCalledTimes(1))
  })

  it("says when the agent list cannot be read", async () => {
    mockRoutes({
      "/agents": () => Promise.reject(new Error("the agents are away")),
    })
    renderDesk()
    expect(await screen.findByText("the agents are away")).toBeTruthy()
  })

  it("tells what Start has reached while the run starts, then goes to the run", async () => {
    let finish: (value: unknown) => void = () => undefined
    startRun.mockImplementation(
      () => new Promise((resolve) => (finish = resolve))
    )
    renderDesk()
    await openForm()
    fireEvent.click(await screen.findByRole("button", { name: "Start run" }))
    expect(await screen.findByText("Starting intake")).toBeTruthy()
    expect(screen.getByText("Checked the folder")).toBeTruthy()
    finish(oneRunStarted)
    await waitFor(() => {
      expect(screen.queryByText("Starting intake")).toBeNull()
    })
  })
})
