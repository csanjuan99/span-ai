import { cleanup, render, screen } from "@testing-library/react"
import { afterEach, describe, expect, it } from "vitest"

import { SafeAndKept } from "@/components/safe-and-kept/SafeAndKept"
import type { Kept } from "@/lib/kept"

afterEach(cleanup)

const kept: Kept = { sealed: 4, branch: "span/SPAN-482", head: "c91f3ab", commits: 3, files: { files: 7, plus: 184, minus: 32 } }

describe("SafeAndKept", () => {
  it("counts the commits on the branch beside the head", () => {
    render(<SafeAndKept kept={kept} />)
    const text = screen.getByRole("region", { name: "Safe and kept" }).textContent
    expect(text).toContain("3 commits on span/SPAN-482 at c91f3ab")
    expect(text).toContain("7 files changed, +184 −32")
  })

  it("uses the singular for one commit", () => {
    render(<SafeAndKept kept={{ ...kept, commits: 1 }} />)
    expect(screen.getByRole("region", { name: "Safe and kept" }).textContent).toContain("1 commit on span/SPAN-482")
  })

  it("names the branch alone when the server did not count commits", () => {
    render(<SafeAndKept kept={{ ...kept, commits: 0 }} />)
    const text = screen.getByRole("region", { name: "Safe and kept" }).textContent
    expect(text).toContain("branch span/SPAN-482 at c91f3ab")
    expect(text).not.toContain("commits on")
  })
})
