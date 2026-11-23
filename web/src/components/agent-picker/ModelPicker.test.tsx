import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

const getJSON = vi.fn()
const saveModel = vi.fn()

vi.mock("@/lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/api")>()
  return {
    ...actual,
    getJSON: (url: string) => getJSON(url),
    saveModel: (project: string, cli: string, model: string) => saveModel(project, cli, model),
  }
})

import { ModelPicker } from "@/components/agent-picker/ModelPicker"
import type { AgentChoice } from "@/lib/types"

afterEach(cleanup)

const choice: AgentChoice = {
  agents: [
    { id: "cursor", label: "Cursor", binary: "cursor-agent", installed: true, gentle: true, ready: true },
    { id: "pi", label: "Pi", binary: "pi", installed: false, gentle: false, ready: false, reason: "pi is not on PATH" },
  ],
  selected: "any",
  chosen: "cursor",
  exists: true,
  reason: "",
  models: { cursor: "auto" },
}

beforeEach(() => {
  getJSON.mockReset()
  saveModel.mockReset()
  getJSON.mockImplementation(() =>
    Promise.resolve({ models: [{ id: "auto", label: "Auto" }, { id: "gpt-5.3-codex", label: "Codex 5.3" }], free: true, source: "cli" }),
  )
})

describe("ModelPicker", () => {
  it("says Gentle AI is required and shows the saved model", async () => {
    render(<ModelPicker project="/p" choice={choice} onSaved={() => undefined} />)
    expect(screen.getByText(/Gentle AI and its configuration are required/)).toBeTruthy()
    const input = screen.getByLabelText("Cursor model")
    expect((input as HTMLInputElement).value).toBe("auto")
    await waitFor(() => expect(getJSON).toHaveBeenCalledWith("/agent/models?cli=cursor"))
  })

  it("asks only the CLIs that are ready and disables the rest with the reason", () => {
    render(<ModelPicker project="/p" choice={choice} onSaved={() => undefined} />)
    expect(getJSON).toHaveBeenCalledTimes(1)
    expect((screen.getByLabelText("Pi model") as HTMLInputElement).disabled).toBe(true)
    expect(screen.getByText(/pi is not on PATH/)).toBeTruthy()
  })

  it("saves the typed model and hands the new view to the page", async () => {
    saveModel.mockImplementation(() => Promise.resolve({ ...choice, models: { cursor: "gpt-5.3-codex" } }))
    const onSaved = vi.fn()
    render(<ModelPicker project="/p" choice={choice} onSaved={onSaved} />)
    fireEvent.change(screen.getByLabelText("Cursor model"), { target: { value: "gpt-5.3-codex" } })
    fireEvent.click(screen.getAllByRole("button", { name: "Save model" })[0] as HTMLElement)
    await waitFor(() => expect(onSaved).toHaveBeenCalled())
    expect(saveModel).toHaveBeenCalledWith("/p", "cursor", "gpt-5.3-codex")
    expect(await screen.findByText("Saved. The next run uses this model.")).toBeTruthy()
  })

  it("shows the server's refusal and still lets the person type when the list failed", async () => {
    getJSON.mockImplementation(() => Promise.resolve({ models: [], free: true, source: "none", error: "not logged in" }))
    saveModel.mockImplementation(() => Promise.reject(new Error("models.cursor: \"--x\" is not a model name")))
    render(<ModelPicker project="/p" choice={choice} onSaved={() => undefined} />)
    expect(await screen.findByText(/The list is not available \(not logged in\)/)).toBeTruthy()
    fireEvent.change(screen.getByLabelText("Cursor model"), { target: { value: "--x" } })
    fireEvent.click(screen.getAllByRole("button", { name: "Save model" })[0] as HTMLElement)
    expect(await screen.findByText(/is not a model name/)).toBeTruthy()
  })

  it("lets a model be cleared on a CLI that is no longer ready, but not changed", () => {
    const stale: AgentChoice = { ...choice, models: { cursor: "auto", pi: "xai/grok-4.7" } }
    render(<ModelPicker project="/p" choice={stale} onSaved={() => undefined} />)
    const pi = screen.getByLabelText("Pi model") as HTMLInputElement
    expect(pi.disabled).toBe(false)
    const save = screen.getAllByRole("button", { name: "Save model" })[1] as HTMLButtonElement
    fireEvent.change(pi, { target: { value: "xai/grok-4.6" } })
    expect(save.disabled).toBe(true)
    fireEvent.change(pi, { target: { value: "" } })
    expect(save.disabled).toBe(false)
  })
})
