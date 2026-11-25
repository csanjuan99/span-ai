import { cleanup, fireEvent, render, screen } from "@testing-library/react"
import { afterEach, describe, expect, it } from "vitest"
import { MemoryRouter } from "react-router"

import { BoardColumns } from "@/components/shaping/board/BoardColumns"
import type { BoardColumn } from "@/lib/shape-flows"

afterEach(() => {
  cleanup()
  Reflect.deleteProperty(window, "matchMedia")
})

const columns: BoardColumn[] = [
  {
    key: "interviewing",
    name: "Interviewing",
    count: 1,
    hidden: 0,
    cards: [
      {
        id: "a1",
        ref: "SES-A",
        title: "Retry webhooks",
        status: "interviewing",
        repo: "api",
        agent: "claude",
        model: "",
        openQuestions: 1,
        assumed: 0,
        stories: 0,
        jiraKey: "",
        sealedAt: "",
        doing: "asking",
      },
    ],
  },
  {
    key: "ready",
    name: "Ready",
    count: 0,
    hidden: 0,
    cards: [],
  },
]

function narrow() {
  window.matchMedia = (query: string) =>
    ({
      matches: query.includes("max-width"),
      media: query,
      addEventListener: () => undefined,
      removeEventListener: () => undefined,
      dispatchEvent: () => false,
      onchange: null,
      addListener: () => undefined,
      removeListener: () => undefined,
    }) as MediaQueryList
}

describe("board columns", () => {
  it("shows one status at a time on a narrow window", async () => {
    narrow()
    render(
      <MemoryRouter>
        <BoardColumns
          columns={columns}
          workspace="."
          chosen={new Set()}
          onToggle={() => undefined}
        />
      </MemoryRouter>
    )
    expect(await screen.findByRole("group", { name: "Status" })).toBeTruthy()
    expect(screen.getByLabelText("Select SES-A")).toBeTruthy()
    fireEvent.click(screen.getByRole("button", { name: /Ready/ }))
    expect(screen.getByText("Nothing here.")).toBeTruthy()
    expect(screen.queryByLabelText("Select SES-A")).toBeNull()
  })
})
