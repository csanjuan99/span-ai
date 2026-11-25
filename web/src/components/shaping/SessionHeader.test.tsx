import { cleanup, render, screen } from "@testing-library/react"
import { MemoryRouter } from "react-router"
import { afterEach, describe, expect, it } from "vitest"

import { SessionHeader } from "@/components/shaping/SessionHeader"
import type { ShapeSession } from "@/lib/types"

afterEach(cleanup)

const session: ShapeSession = {
  id: "ses-1",
  workspace: ".",
  repo: "billing",
  idea: "Retry deliveries",
  status: "ready",
  rules: [],
  examples: [],
  questions: [],
  split: [],
  proposals: [],
  story: "",
  digest: "",
  jira: null,
  ref: "SES-1",
}

describe("SessionHeader", () => {
  it("links the step history, the version compare and the export", () => {
    render(
      <MemoryRouter>
        <SessionHeader session={session} />
      </MemoryRouter>
    )

    expect(screen.getByRole("link", { name: "History" }).getAttribute("href")).toBe(
      "/shaping/ses-1/history?workspace=."
    )
    expect(screen.getByRole("link", { name: "Compare" }).getAttribute("href")).toBe(
      "/shaping/ses-1/versions?workspace=."
    )
    expect(screen.getByRole("link", { name: "Export" }).getAttribute("href")).toBe(
      "/shaping/ses-1/transfer?workspace=."
    )
    expect(screen.queryByRole("link", { name: "Steps" })).toBeNull()
  })
})
