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

vi.mock("@/lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/api")>()
  return { ...actual, getJSON: (url: string) => getJSON(url) }
})
vi.mock("@/components/agent-picker/AgentPicker", () => ({
  AgentPicker: () => <p>agent picker</p>,
}))

import { ThemeProvider } from "@/components/theme-provider"
import { ProjectSettings } from "@/pages/ProjectSettings"

afterEach(cleanup)

beforeEach(() => {
  getJSON.mockReset()
  localStorage.clear()
  window.matchMedia = ((query: string) => ({
    matches: false,
    media: query,
    addEventListener: () => undefined,
    removeEventListener: () => undefined,
  })) as unknown as typeof window.matchMedia
})

function show() {
  render(
    <MemoryRouter>
      <ThemeProvider>
        <ProjectSettings />
      </ThemeProvider>
    </MemoryRouter>
  )
}

describe("ProjectSettings", () => {
  it("shows what span.yaml says and marks what differs from the default", async () => {
    getJSON.mockResolvedValue({
      exists: true,
      base: "feat/SAL-3",
      stages: { merge: "hold" },
      prove: { fast: ["go test ./..."] },
      qa: "npm test",
    })
    show()
    expect(await screen.findByText("feat/SAL-3")).toBeTruthy()
    expect(screen.getByText("go test ./...")).toBeTruthy()
    expect(screen.getAllByText("Changed").length).toBeGreaterThanOrEqual(3)
    expect(screen.getAllByText("Asks me").length).toBeGreaterThanOrEqual(1)
  })

  it("says so when the project has no span.yaml", async () => {
    getJSON.mockResolvedValue({
      exists: false,
      reason: "There is no span.yaml, so runs use the defaults.",
    })
    show()
    expect(await screen.findByText(/There is no span.yaml/)).toBeTruthy()
    expect(screen.queryByText("Changed")).toBeNull()
  })

  it("changes the theme from the Appearance section", async () => {
    getJSON.mockResolvedValue({})
    show()
    await waitFor(() => expect(getJSON).toHaveBeenCalled())
    fireEvent.click(screen.getByRole("button", { name: "Dark" }))
    expect(localStorage.getItem("theme")).toBe("dark")
  })
})
