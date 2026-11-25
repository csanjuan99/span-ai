import { cleanup, fireEvent, render, screen } from "@testing-library/react"
import { afterEach, describe, expect, it, vi } from "vitest"

import { FilesChanged } from "@/components/files-changed/FilesChanged"
import type { DiffFile } from "@/lib/diff"

afterEach(cleanup)

const file = (
  path: string,
  status: DiffFile["status"],
  added: number,
  removed: number,
  oldPath = path
): DiffFile => ({
  path,
  oldPath,
  status,
  binary: false,
  added,
  removed,
  hunks: [],
})

describe("FilesChanged", () => {
  it("lists each file with its kind letter and counts", () => {
    render(
      <FilesChanged
        files={[
          file("internal/retry.go", "added", 84, 0),
          file("client.go", "modified", 24, 19),
        ]}
      />
    )
    const region = screen.getByRole("region", { name: "Files changed" })
    expect(region.textContent).toContain("+84")
    expect(region.textContent).toContain("−19")
    expect(screen.getAllByRole("listitem")).toHaveLength(2)
    expect(screen.getByTitle("added").textContent).toContain("A")
    expect(screen.getByTitle("modified").textContent).toContain("M")
  })

  it("says where a renamed file came from instead of counts", () => {
    render(
      <FilesChanged files={[file("payload.go", "renamed", 0, 0, "event.go")]} />
    )
    expect(screen.getByRole("listitem").textContent).toContain(
      "renamed from event.go"
    )
  })

  it("offers the full diff only when asked to", () => {
    const open = vi.fn()
    const { rerender } = render(
      <FilesChanged files={[file("a.go", "added", 1, 0)]} />
    )
    expect(screen.queryByRole("button", { name: /open full diff/i })).toBeNull()
    rerender(
      <FilesChanged files={[file("a.go", "added", 1, 0)]} onOpenDiff={open} />
    )
    fireEvent.click(screen.getByRole("button", { name: /open full diff/i }))
    expect(open).toHaveBeenCalledTimes(1)
  })

  it("draws nothing for an empty change", () => {
    render(<FilesChanged files={[]} />)
    expect(screen.queryByRole("region", { name: "Files changed" })).toBeNull()
  })
})
