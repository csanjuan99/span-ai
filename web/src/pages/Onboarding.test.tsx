import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react"
import { MemoryRouter, Route, Routes, useParams } from "react-router"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

const getJSON = vi.fn()

const startRun = vi.fn()
const saveAgent = vi.fn()
const checkJiraAccount = vi.fn()
const saveJiraAccount = vi.fn()

vi.mock("@/lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/api")>()
  return {
    ...actual,
    getJSON: (url: string) => getJSON(url),
    startRun: (form: FormData) => startRun(form),
    saveAgent: (project: string, id: string) => saveAgent(project, id),
    checkJiraAccount: (project: string, account: unknown) =>
      checkJiraAccount(project, account),
    saveJiraAccount: (project: string, account: unknown) =>
      saveJiraAccount(project, account),
  }
})

import { readOnboarding, saveOnboarding } from "@/lib/onboarding"
import { Onboarding } from "@/pages/Onboarding"

afterEach(cleanup)

const agents = [
  {
    id: "claude",
    label: "Claude Code",
    binary: "claude",
    installed: true,
    gentle: true,
    ready: true,
  },
  {
    id: "codex",
    label: "Codex",
    binary: "codex",
    installed: true,
    gentle: true,
    ready: false,
    reason: "Codex does not have Gentle AI installed",
  },
  {
    id: "pi",
    label: "Pi",
    binary: "pi",
    installed: false,
    gentle: false,
    ready: false,
    install: "npm install -g @mariozechner/pi-coding-agent",
  },
]

function Run() {
  const { id } = useParams()
  return <p>run {id}</p>
}

function Desk() {
  return <p>desk</p>
}

const repo = {
  path: "/work/api",
  exists: true,
  git: true,
  branch: "main",
  remote: "origin",
  dirty: 2,
  reason: "",
}

function show() {
  render(
    <MemoryRouter initialEntries={["/welcome"]}>
      <Routes>
        <Route path="/welcome" element={<Onboarding />} />
        <Route path="/" element={<Desk />} />
        <Route path="/spans/:id" element={<Run />} />
      </Routes>
    </MemoryRouter>
  )
}

function answer(overrides: Record<string, unknown> = {}) {
  getJSON.mockImplementation((url: string) => {
    if (url in overrides) {
      return Promise.resolve(overrides[url])
    }
    if (url.startsWith("/onboarding/folder?")) {
      return Promise.resolve(repo)
    }
    if (url.startsWith("/agent?")) {
      return Promise.resolve({
        agents,
        selected: "any",
        chosen: "claude",
        exists: true,
        reason: "",
        models: {},
      })
    }
    if (url.startsWith("/jira")) {
      return Promise.resolve({ configured: false })
    }
    return Promise.resolve({})
  })
}

beforeEach(() => {
  for (const mock of [
    getJSON,
    startRun,
    saveAgent,
    checkJiraAccount,
    saveJiraAccount,
  ]) {
    mock.mockReset()
  }
  localStorage.clear()
  answer()
})

describe("Onboarding", () => {
  it("opens on the welcome step with the ten stages and moves to connect", () => {
    show()
    expect(
      screen.getByText("Span takes a story from idea to merged code")
    ).toBeTruthy()
    expect(screen.getByRole("region", { name: "The ten stages" })).toBeTruthy()
    fireEvent.click(
      screen.getByRole("button", { name: "Continue: connect your project" })
    )
    expect(
      screen.getByRole("heading", { name: "Connect your project" })
    ).toBeTruthy()
    expect(readOnboarding().step).toBe(2)
  })

  it("marks the step you are on in the bar", () => {
    show()
    expect(
      screen
        .getByRole("list", { name: "Setup steps" })
        .querySelector("[aria-current=step]")?.textContent
    ).toContain("What Span does")
  })

  it("checks the folder and each agent for real, and says what is wrong", async () => {
    saveOnboarding({ step: 2, folder: "/work/api" })
    show()
    expect(await screen.findByText(/Git repository\./)).toBeTruthy()
    expect(getJSON).toHaveBeenCalledWith(
      "/onboarding/folder?path=%2Fwork%2Fapi"
    )
    expect(screen.getByText(/2 uncommitted files/)).toBeTruthy()
    expect(
      await screen.findByText(/Codex does not have Gentle AI installed/)
    ).toBeTruthy()
    expect(screen.getByText("Not installed")).toBeTruthy()
    expect(
      screen.getByRole("button", {
        name: "Copy npm install -g @mariozechner/pi-coding-agent",
      })
    ).toBeTruthy()
    expect(
      screen.getByRole("button", { name: "Continue" }).hasAttribute("disabled")
    ).toBe(false)
  })

  it("welcomes a returning visitor back with the folder filled in", async () => {
    saveOnboarding({ step: 2, folder: "/work/api" })
    show()
    expect(await screen.findByText(/Welcome back/)).toBeTruthy()
    expect(screen.getByText(/Saved just now/)).toBeTruthy()
  })

  it("blocks Continue when the folder is not a repository", async () => {
    saveOnboarding({ step: 2, folder: "/nowhere" })
    answer({
      "/onboarding/folder?path=%2Fnowhere": {
        ...repo,
        git: false,
        branch: "",
        remote: "",
        dirty: 0,
      },
    })
    show()
    expect(await screen.findByText(/Not a git repository/)).toBeTruthy()
    expect(screen.getByRole("button", { name: "Copy git init" })).toBeTruthy()
    expect(
      screen.getByRole("button", { name: "Continue" }).hasAttribute("disabled")
    ).toBe(true)
  })

  it("blocks Continue when no agent is ready", async () => {
    saveOnboarding({ step: 2, folder: "/work/api" })
    answer({
      "/agent?path=%2Fwork%2Fapi": {
        agents: agents.slice(1),
        selected: "any",
        chosen: "",
        exists: true,
        reason: "",
        models: {},
      },
    })
    show()
    await screen.findByText(/Codex does not have Gentle AI installed/)
    expect(
      screen.getByRole("button", { name: "Continue" }).hasAttribute("disabled")
    ).toBe(true)
  })

  it("remembers the agent that was picked", async () => {
    saveOnboarding({ step: 2, folder: "/work/api" })
    answer({
      "/agent?path=%2Fwork%2Fapi": {
        agents: [agents[0], { ...agents[1], ready: true }],
        selected: "any",
        chosen: "claude",
        exists: true,
        reason: "",
        models: {},
      },
    })
    show()
    fireEvent.click(await screen.findByRole("radio", { name: /Codex/ }))
    expect(readOnboarding().agent).toBe("codex")
  })

  it("tests the Jira account and saves it only when it is accepted", async () => {
    saveOnboarding({ step: 2, folder: "/work/api" })
    checkJiraAccount.mockResolvedValueOnce({
      status: "unauthorized",
      message: "Jira did not accept this token.",
    })
    show()
    fireEvent.change(await screen.findByLabelText("Jira site"), {
      target: { value: "https://acme.atlassian.net" },
    })
    fireEvent.change(screen.getByLabelText("API token"), {
      target: { value: "bad" },
    })
    fireEvent.click(screen.getByRole("button", { name: "Test and save" }))
    expect(await screen.findByText(/Rejected\./)).toBeTruthy()
    expect(saveJiraAccount).not.toHaveBeenCalled()
    checkJiraAccount.mockResolvedValueOnce({ status: "ok", message: "" })
    saveJiraAccount.mockResolvedValueOnce({})
    fireEvent.click(screen.getByRole("button", { name: "Test and save" }))
    expect(await screen.findByText(/Accepted and saved/)).toBeTruthy()
    expect(saveJiraAccount).toHaveBeenCalledWith("/work/api", {
      url: "https://acme.atlassian.net",
      email: "",
      token: "bad",
    })
  })

  it("starts the first run for real and opens it", async () => {
    saveOnboarding({ step: 3, folder: "/work/api", agent: "claude" })
    startRun.mockResolvedValueOnce({
      id: "run-7",
      stage: "intake",
      runs: [],
      partial: false,
      notes: "",
    })
    show()
    expect(screen.getByDisplayValue("Add a /healthz endpoint")).toBeTruthy()
    expect(screen.getByRole("region", { name: "Dry look" })).toBeTruthy()
    fireEvent.click(screen.getByRole("button", { name: "Start the first run" }))
    expect(await screen.findByText("run run-7")).toBeTruthy()
    expect(saveAgent).toHaveBeenCalledWith("/work/api", "claude")
    const form = startRun.mock.calls[0]?.[0] as FormData
    expect(form.get("workspace")).toBe("/work/api")
    expect(String(form.get("story"))).toContain("Add a /healthz endpoint")
    expect(readOnboarding().done).toBe(true)
  })

  it("says why the run did not start and stays on the step", async () => {
    saveOnboarding({ step: 3, folder: "/work/api" })
    startRun.mockRejectedValueOnce(new Error("the folder is busy"))
    show()
    fireEvent.click(screen.getByRole("button", { name: "Start the first run" }))
    expect(await screen.findByText("the folder is busy")).toBeTruthy()
    expect(readOnboarding().done).toBe(false)
    expect(saveAgent).not.toHaveBeenCalled()
  })

  it("skips setup from the bar and remembers it", async () => {
    show()
    fireEvent.click(screen.getAllByRole("button", { name: "Skip setup" })[0]!)
    await waitFor(() => expect(screen.getByText("desk")).toBeTruthy())
    expect(readOnboarding().skipped).toBe(true)
  })

  it("goes back a step", () => {
    saveOnboarding({ step: 3 })
    show()
    fireEvent.click(screen.getByRole("button", { name: "Back" }))
    expect(
      screen.getByRole("heading", { name: "Connect your project" })
    ).toBeTruthy()
  })
})
