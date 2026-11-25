import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react"
import { MemoryRouter, Route, Routes, useLocation } from "react-router"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

const getJSON = vi.fn()

vi.mock("@/lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/api")>()
  return { ...actual, getJSON: (url: string) => getJSON(url) }
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
    reason: "not logged in",
  },
  {
    id: "pi",
    label: "Pi",
    binary: "pi",
    installed: false,
    gentle: false,
    ready: false,
  },
]

function Landing() {
  const location = useLocation()
  return <p>desk {JSON.stringify(location.state)}</p>
}

function show() {
  render(
    <MemoryRouter initialEntries={["/welcome"]}>
      <Routes>
        <Route path="/welcome" element={<Onboarding />} />
        <Route path="/" element={<Landing />} />
      </Routes>
    </MemoryRouter>
  )
}

beforeEach(() => {
  getJSON.mockReset()
  localStorage.clear()
  getJSON.mockImplementation((url: string) => {
    if (url.startsWith("/branches")) {
      return Promise.resolve({ branches: ["dev", "main"], default: "dev" })
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
})

describe("Onboarding", () => {
  it("opens on the welcome step and moves to connect", () => {
    show()
    expect(
      screen.getByText("Span takes a story from idea to merged code")
    ).toBeTruthy()
    fireEvent.click(
      screen.getByRole("button", { name: "Continue: connect your project" })
    )
    expect(
      screen.getByRole("heading", { name: "Connect your project" })
    ).toBeTruthy()
    expect(readOnboarding().step).toBe(2)
  })

  it("checks the folder and each agent for real, and says what is wrong", async () => {
    saveOnboarding({ step: 2, folder: "/work/api" })
    show()
    expect(await screen.findByText(/Git repository/)).toBeTruthy()
    expect(getJSON).toHaveBeenCalledWith("/branches?path=%2Fwork%2Fapi")
    expect(
      await screen.findByText(/Claude Code is installed and ready/)
    ).toBeTruthy()
    expect(
      screen.getByText(/Codex is installed but not ready: not logged in/)
    ).toBeTruthy()
    expect(screen.getByText(/Pi was not found/)).toBeTruthy()
    expect(
      screen.getByRole("button", { name: "Continue" }).hasAttribute("disabled")
    ).toBe(false)
  })

  it("blocks Continue when the folder is not a repository", async () => {
    saveOnboarding({ step: 2, folder: "/nowhere" })
    getJSON.mockImplementation((url: string) =>
      url.startsWith("/branches")
        ? Promise.reject(new Error("not a git repository"))
        : Promise.resolve({ agents, configured: false })
    )
    show()
    expect(await screen.findByText(/not a git repository/)).toBeTruthy()
    expect(
      screen.getByRole("button", { name: "Continue" }).hasAttribute("disabled")
    ).toBe(true)
  })

  it("blocks Continue when no agent is ready", async () => {
    saveOnboarding({ step: 2, folder: "/work/api" })
    getJSON.mockImplementation((url: string) => {
      if (url.startsWith("/agent?")) {
        return Promise.resolve({
          agents: agents.slice(1),
          selected: "any",
          chosen: "",
          exists: true,
          reason: "",
          models: {},
        })
      }
      return Promise.resolve({
        branches: ["dev"],
        default: "dev",
        configured: false,
      })
    })
    show()
    await screen.findByText(/Codex is installed but not ready/)
    expect(
      screen.getByRole("button", { name: "Continue" }).hasAttribute("disabled")
    ).toBe(true)
  })

  it("starts the first run on the desk with the sample asked for", () => {
    saveOnboarding({ step: 3 })
    show()
    expect(screen.getByText("Add a /healthz endpoint")).toBeTruthy()
    expect(screen.getByRole("region", { name: "Dry look" })).toBeTruthy()
    fireEvent.click(screen.getByRole("button", { name: "Start the first run" }))
    expect(screen.getByText(/"sample":true/)).toBeTruthy()
    expect(readOnboarding().done).toBe(true)
  })

  it("skips setup from any step and remembers it", async () => {
    show()
    fireEvent.click(screen.getByRole("button", { name: "Skip setup" }))
    await waitFor(() => expect(screen.getByText(/desk/)).toBeTruthy())
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
