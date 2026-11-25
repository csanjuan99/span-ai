import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react"
import { MemoryRouter } from "react-router"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

const getJSON = vi.fn()
const previewConfig = vi.fn()
const saveConfig = vi.fn()

vi.mock("@/lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/api")>()
  return {
    ...actual,
    getJSON: (url: string) => getJSON(url),
    previewConfig: (project: string, edits: unknown) =>
      previewConfig(project, edits),
    saveConfig: (project: string, edits: unknown) => saveConfig(project, edits),
  }
})
vi.mock("@/components/agent-picker/AgentPicker", async () => {
  const { useEffect } = await import("react")
  const ready = {
    id: "codex",
    label: "Codex",
    binary: "codex",
    installed: true,
    gentle: true,
    ready: true,
  }
  return {
    AgentPicker: ({ onLoaded }: { onLoaded?: (choice: unknown) => void }) => {
      useEffect(() => {
        onLoaded?.({
          agents: [ready],
          selected: "any",
          chosen: "",
          exists: false,
          reason: "",
          models: {},
        })
      }, [onLoaded])
      return <p>agent picker</p>
    },
  }
})

import { projectConfig } from "@/lib/api"
import { ThemeProvider } from "@/components/theme-provider"
import { ProjectSettings } from "@/pages/ProjectSettings"

afterEach(cleanup)

beforeEach(() => {
  getJSON.mockReset()
  previewConfig.mockReset()
  saveConfig.mockReset()
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
    expect(await screen.findByDisplayValue("feat/SAL-3")).toBeTruthy()
    expect(screen.getByDisplayValue("go test ./...")).toBeTruthy()
    expect(screen.getByDisplayValue("npm test")).toBeTruthy()
    expect(screen.getAllByText("Changed").length).toBeGreaterThanOrEqual(3)
    const merge = screen.getByLabelText<HTMLSelectElement>("merge approval")
    expect(merge.value).toBe("hold")
    expect(screen.queryByText(/\d+ unsaved/)).toBeNull()
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

  describe("editing span.yaml", () => {
    const saved = {
      exists: true,
      stages: { merge: "auto" },
      prove: { fast: ["go test ./..."] },
    }

    async function edited() {
      getJSON.mockResolvedValue(saved)
      previewConfig.mockResolvedValue({
        lines: [{ key: "stages.merge", before: "auto", after: "hold" }],
        effects: [
          {
            label: "Next stage",
            text: "A run that reaches merge will stop and ask you.",
          },
        ],
      })
      show()
      await screen.findByDisplayValue("go test ./...")
      fireEvent.change(screen.getByLabelText("merge approval"), {
        target: { value: "hold" },
      })
    }

    it("shows what a change would write before anything is saved", async () => {
      await edited()
      const panel = await screen.findByRole("region", { name: "What changes" })
      expect(await within(panel).findByText(/merge hold/)).toBeTruthy()
      expect(within(panel).getByText(/merge auto/)).toBeTruthy()
      expect(within(panel).getByText("1 unsaved")).toBeTruthy()
      expect(
        within(panel).getByText(
          "A run that reaches merge will stop and ask you."
        )
      ).toBeTruthy()
      expect(previewConfig).toHaveBeenCalledWith(".", { merge: "hold" })
      expect(saveConfig).not.toHaveBeenCalled()
    })

    it("saves only what changed and shows the settings the server returns", async () => {
      await edited()
      saveConfig.mockResolvedValue({
        lines: [{ key: "stages.merge", before: "auto", after: "hold" }],
        effects: [],
        config: projectConfig({ ...saved, stages: { merge: "hold" } }),
      })
      const panel = await screen.findByRole("region", { name: "What changes" })
      await within(panel).findByText(/merge hold/)
      fireEvent.click(screen.getByRole("button", { name: "Save 1 change" }))
      expect(
        await screen.findByText("Saved 1 change to span.yaml.")
      ).toBeTruthy()
      expect(saveConfig).toHaveBeenCalledWith(".", { merge: "hold" })
      expect(screen.queryByText(/\d+ unsaved/)).toBeNull()
      expect(
        screen.getByLabelText<HTMLSelectElement>("merge approval").value
      ).toBe("hold")
    })

    it("drops the changes on Discard and writes nothing", async () => {
      await edited()
      fireEvent.click(
        await screen.findByRole("button", { name: "Discard changes" })
      )
      expect(screen.queryByText(/\d+ unsaved/)).toBeNull()
      expect(
        screen.getByLabelText<HTMLSelectElement>("merge approval").value
      ).toBe("auto")
      expect(saveConfig).not.toHaveBeenCalled()
    })

    it("does not count a value typed back to what is saved", async () => {
      await edited()
      fireEvent.change(screen.getByLabelText("merge approval"), {
        target: { value: "auto" },
      })
      expect(screen.queryByText(/\d+ unsaved/)).toBeNull()
    })

    it("says why a value is refused and keeps Save off", async () => {
      getJSON.mockResolvedValue(saved)
      previewConfig.mockRejectedValue(
        new Error("repair.maxAttempts is 12, allowed 1 to 8")
      )
      show()
      await screen.findByDisplayValue("go test ./...")
      fireEvent.change(screen.getByLabelText("Repair attempts"), {
        target: { value: "12" },
      })
      expect(await screen.findByText(/allowed 1 to 8/)).toBeTruthy()
      const button = screen.getByRole("button", { name: "Save 1 change" })
      expect((button as HTMLButtonElement).disabled).toBe(true)
    })

    it("sends one command per line for a list of checks", async () => {
      getJSON.mockResolvedValue(saved)
      previewConfig.mockResolvedValue({
        lines: [
          {
            key: "prove.fast",
            before: "go test ./...",
            after: "go vet ./..., go test ./...",
          },
        ],
        effects: [],
      })
      show()
      await screen.findByDisplayValue("go test ./...")
      fireEvent.change(screen.getByLabelText("Fast"), {
        target: { value: "go vet ./...\n\ngo test ./...\n" },
      })
      await waitFor(() =>
        expect(previewConfig).toHaveBeenCalledWith(".", {
          fast: ["go vet ./...", "go test ./..."],
        })
      )
    })

    it("sets the CLI of one stage and the Jira status without touching the rest", async () => {
      getJSON.mockResolvedValue(saved)
      previewConfig.mockResolvedValue({ lines: [], effects: [] })
      show()
      await screen.findByDisplayValue("go test ./...")
      fireEvent.change(screen.getByLabelText("build agent"), {
        target: { value: "codex" },
      })
      fireEvent.change(screen.getByLabelText("Move story on done"), {
        target: { value: "Done" },
      })
      await waitFor(() =>
        expect(previewConfig).toHaveBeenCalledWith(".", {
          "agents.build": "codex",
          doneStatus: "Done",
        })
      )
    })
  })
})
