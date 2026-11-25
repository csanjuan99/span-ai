import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

const deleteRun = vi.fn()

vi.mock("@/lib/api", () => ({ deleteRun: (id: string) => deleteRun(id) }))

import { DeleteRun } from "@/components/run-list/DeleteRun"

afterEach(cleanup)
beforeEach(() => {
  // Braces matter: a hook that returns a function has it run as cleanup, and the mock would be called again.
  deleteRun.mockReset()
})

describe("DeleteRun", () => {
  it("asks before deleting, and says what stays", () => {
    render(<DeleteRun id="r1" label="AET-1" onDeleted={() => undefined} />)
    fireEvent.click(screen.getByRole("button", { name: "Delete AET-1" }))
    expect(screen.getByText(/The branch and any worktree stay/)).toBeTruthy()
    expect(deleteRun).not.toHaveBeenCalled()
  })

  it("keeping goes back without deleting", () => {
    render(<DeleteRun id="r1" label="AET-1" onDeleted={() => undefined} />)
    fireEvent.click(screen.getByRole("button", { name: "Delete AET-1" }))
    fireEvent.click(screen.getByRole("button", { name: "Keep" }))
    expect(screen.getByRole("button", { name: "Delete AET-1" })).toBeTruthy()
    expect(deleteRun).not.toHaveBeenCalled()
  })

  it("deletes after the second click and tells the list", async () => {
    deleteRun.mockResolvedValue(undefined)
    const onDeleted = vi.fn()
    render(<DeleteRun id="r1" label="AET-1" onDeleted={onDeleted} />)
    fireEvent.click(screen.getByRole("button", { name: "Delete AET-1" }))
    fireEvent.click(screen.getByRole("button", { name: "Yes, delete" }))
    await waitFor(() => expect(onDeleted).toHaveBeenCalledWith("r1"))
    expect(deleteRun).toHaveBeenCalledWith("r1")
  })

  it("shows the refusal and keeps the run", async () => {
    deleteRun.mockRejectedValue(new Error("busy: stop it before deleting it"))
    const onDeleted = vi.fn()
    render(<DeleteRun id="r1" label="AET-1" onDeleted={onDeleted} />)
    fireEvent.click(screen.getByRole("button", { name: "Delete AET-1" }))
    fireEvent.click(screen.getByRole("button", { name: "Yes, delete" }))
    expect(await screen.findByText(/stop it before deleting it/)).toBeTruthy()
    expect(onDeleted).not.toHaveBeenCalled()
  })
})
