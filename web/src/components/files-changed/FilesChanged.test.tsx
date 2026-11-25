import { cleanup, render, screen } from "@testing-library/react"
import { afterEach, describe, expect, it } from "vitest"

import { FilesChanged } from "@/components/files-changed/FilesChanged"
import type { DiffFile } from "@/lib/diff"

afterEach(cleanup)

const file = (path: string, status: DiffFile["status"], added: number, removed: number, oldPath = path): DiffFile => ({
  path,
  oldPath,
  status,
  binary: false,
  added,
  removed,
  hunks: [],
})

describe("FilesChanged", () => {
  it("lists each file with its kind letter and counts, and the total", () => {
    render(<FilesChanged files={[file("internal/retry.go", "added", 84, 0), file("client.go", "modified", 24, 19)]} />)
    const region = screen.getByRole("region", { name: "Files changed" })
    expect(region.textContent).toContain("2 files")
    expect(region.textContent).toContain("+108")
    expect(region.textContent).toContain("−19")
    expect(screen.getAllByRole("listitem")).toHaveLength(2)
    expect(screen.getByTitle("added").textContent).toBe("A")
    expect(screen.getByTitle("modified").textContent).toBe("M")
  })

  it("says where a renamed file came from", () => {
    render(<FilesChanged files={[file("payload.go", "renamed", 0, 0, "event.go")]} />)
    expect(screen.getByRole("listitem").textContent).toContain("from event.go")
  })

  it("draws nothing for an empty change", () => {
    render(<FilesChanged files={[]} />)
    expect(screen.queryByRole("region", { name: "Files changed" })).toBeNull()
  })
})
