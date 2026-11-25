import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

import { ConflictPanel } from "@/components/conflict/ConflictPanel"
import type { Conflict, ConflictGroup, ConflictOption } from "@/lib/types"

function option(over: Partial<ConflictOption> = {}): ConflictOption {
  return {
    id: "ours",
    title: "Keep the branch total",
    summary: "The branch rounding wins.",
    risk: "low",
    action: "ours",
    command: "",
    diff: "@@ -1,2 +1,2 @@\n same\n-old total\n+new total\n",
    ...over,
  }
}

function group(over: Partial<ConflictGroup> = {}): ConflictGroup {
  return {
    id: "g1",
    kind: "content",
    files: ["src/cart.ts"],
    summary: "Both sides changed the total.",
    options: [
      option(),
      option({
        id: "theirs",
        title: "Take the base total",
        action: "theirs",
        risk: "high",
      }),
    ],
    chosen: "",
    ...over,
  }
}

function conflict(over: Partial<Conflict> = {}): Conflict {
  return {
    base: "main",
    baseSHA: "abc",
    state: "choosing",
    note: "Two groups conflict.",
    groups: [
      group(),
      group({
        id: "g2",
        kind: "lockfile",
        files: ["package-lock.json"],
        summary: "Lockfile.",
      }),
    ],
    ...over,
  }
}

type Call = [string, RequestInit]
const calls: Call[] = []
let status = 202
let reply = ""

beforeEach(() => {
  calls.length = 0
  status = 202
  reply = ""
  vi.stubGlobal(
    "fetch",
    vi.fn((url: string, init: RequestInit) => {
      calls.push([url, init])
      return Promise.resolve(new Response(reply, { status }))
    })
  )
})

afterEach(() => {
  cleanup()
  vi.unstubAllGlobals()
})

function section(name: string): HTMLElement {
  return screen.getByRole("group", { name: new RegExp(name) })
}

describe("ConflictPanel choosing", () => {
  it("preselects nothing and keeps apply disabled until every group has a choice", () => {
    render(<ConflictPanel id="s" conflict={conflict()} />)
    const apply = screen.getByRole("button", { name: "Apply resolution" })
    expect(
      screen
        .getAllByRole("radio")
        .every((radio) => radio instanceof HTMLInputElement && !radio.checked)
    ).toBe(true)
    expect(apply).toHaveProperty("disabled", true)
    expect(screen.getByText("2 groups left to choose")).toBeTruthy()
    fireEvent.click(
      within(section("Both sides changed")).getByRole("radio", {
        name: /Take the base total/,
      })
    )
    expect(screen.getByText("1 group left to choose")).toBeTruthy()
    expect(apply).toHaveProperty("disabled", true)
    fireEvent.click(
      within(section("Lockfile")).getByRole("radio", {
        name: /Keep the branch total/,
      })
    )
    expect(apply).toHaveProperty("disabled", false)
  })

  it("posts one pair per group and waits for the socket", async () => {
    const { rerender } = render(<ConflictPanel id="s" conflict={conflict()} />)
    fireEvent.click(
      within(section("Both sides changed")).getByRole("radio", {
        name: /Take the base total/,
      })
    )
    fireEvent.click(
      within(section("Lockfile")).getByRole("radio", {
        name: /Keep the branch total/,
      })
    )
    fireEvent.click(screen.getByRole("button", { name: "Apply resolution" }))
    await screen.findByText("Applying your choice…")
    expect(calls[0]?.[0]).toBe("/spans/s/conflict/choose")
    expect(String(calls[0]?.[1].body)).toBe(
      "group=g1&option=theirs&group=g2&option=ours"
    )
    expect(
      screen.queryByRole("button", { name: "Apply resolution" })
    ).toBeNull()
    rerender(
      <ConflictPanel
        id="s"
        conflict={conflict({ state: "failed", note: "push was rejected" })}
      />
    )
    expect(screen.getByText("push was rejected")).toBeTruthy()
    expect(screen.queryByText("Applying your choice…")).toBeNull()
  })

  it("shows the server's message inline when the choice is rejected", async () => {
    status = 409
    reply = JSON.stringify({ message: "every group needs an option" })
    render(<ConflictPanel id="s" conflict={conflict({ groups: [group()] })} />)
    fireEvent.click(screen.getAllByRole("radio")[0] ?? document.body)
    fireEvent.click(screen.getByRole("button", { name: "Apply resolution" }))
    expect((await screen.findByRole("alert")).textContent).toBe(
      "every group needs an option"
    )
    expect(
      screen.getByRole("button", { name: "Apply resolution" })
    ).toBeTruthy()
  })

  it("asks the agent again only with a hint", async () => {
    render(<ConflictPanel id="s" conflict={conflict()} />)
    fireEvent.click(
      screen.getByRole("button", { name: "Leave a hint for the agent" })
    )
    const ask = screen.getByRole("button", { name: "Ask the agent again" })
    expect(ask).toHaveProperty("disabled", true)
    fireEvent.change(screen.getByLabelText("Hint for the agent"), {
      target: { value: "   " },
    })
    expect(ask).toHaveProperty("disabled", true)
    fireEvent.change(screen.getByLabelText("Hint for the agent"), {
      target: { value: "keep the new rounding" },
    })
    fireEvent.click(ask)
    await screen.findByText("The agent is reading both sides…")
    expect(calls[0]?.[0]).toBe("/spans/s/conflict/hint")
    expect(String(calls[0]?.[1].body)).toBe("text=keep+the+new+rounding")
  })

  it("aborts only after the confirmation", async () => {
    render(<ConflictPanel id="s" conflict={conflict()} />)
    fireEvent.click(screen.getByRole("button", { name: "Abort merge" }))
    expect(calls).toHaveLength(0)
    const dialog = await screen.findByRole("alertdialog")
    fireEvent.click(
      within(dialog).getByRole("button", { name: "Keep resolving" })
    )
    expect(calls).toHaveLength(0)
    fireEvent.click(screen.getByRole("button", { name: "Abort merge" }))
    fireEvent.click(
      within(await screen.findByRole("alertdialog")).getByRole("button", {
        name: "Abort the merge",
      })
    )
    await waitFor(() => {
      expect(calls[0]?.[0]).toBe("/spans/s/conflict/abort")
    })
  })

  it("shows the regenerate command and when it runs", () => {
    const regen = option({
      id: "regen",
      title: "Regenerate the lockfile",
      action: "regenerate",
      command: "npm install --package-lock-only",
      diff: "",
    })
    render(
      <ConflictPanel
        id="s"
        conflict={conflict({ groups: [group({ options: [option(), regen] })] })}
      />
    )
    expect(screen.getByText("npm install --package-lock-only")).toBeTruthy()
    expect(screen.getByText("Runs only after you apply")).toBeTruthy()
    expect(screen.getByText("Regenerate")).toBeTruthy()
    expect(screen.getAllByText("Keep this branch's version")).toHaveLength(1)
  })

  it("names the risk in words and styles it differently", () => {
    render(<ConflictPanel id="s" conflict={conflict({ groups: [group()] })} />)
    const low = screen.getByText("low risk")
    const high = screen.getByText("high risk")
    expect(low.className).not.toBe(high.className)
  })

  it("shows the result of the chosen option only, marked and numbered, capped behind Show more", () => {
    const long = [
      "@@ -1,60 +1,60 @@",
      ...Array.from({ length: 60 }, (_, at) => `+line ${String(at)}`),
    ].join("\n")
    render(
      <ConflictPanel
        id="s"
        conflict={conflict({
          groups: [
            group({
              options: [
                option(),
                option({ id: "x", title: "Long", diff: long }),
              ],
            }),
          ],
        })}
      />
    )
    expect(screen.queryByText("-old total")).toBeNull()
    fireEvent.click(
      screen.getByRole("radio", { name: /Keep the branch total/ })
    )
    expect(screen.getByText("Result if you keep the branch total")).toBeTruthy()
    const del = screen.getByText("-old total")
    expect(del.closest("[role=region]")?.className).toContain("font-mono")
    expect(del.getAttribute("data-kind")).toBe("del")
    expect(screen.getByText("+new total").getAttribute("data-kind")).toBe("add")
    expect(screen.getByText("Chosen")).toBeTruthy()
    fireEvent.click(screen.getByRole("radio", { name: /Long/ }))
    expect(screen.queryByText("-old total")).toBeNull()
    expect(screen.queryByText("+line 59")).toBeNull()
    fireEvent.click(screen.getByRole("button", { name: "Show more" }))
    expect(screen.getByText("+line 59")).toBeTruthy()
  })

  it("collapses a long file list after five", () => {
    const files = ["a", "b", "c", "d", "e", "f", "g"].map(
      (name) => `src/${name}.ts`
    )
    render(
      <ConflictPanel
        id="s"
        conflict={conflict({ groups: [group({ files })] })}
      />
    )
    expect(screen.queryByText("src/f.ts")).toBeNull()
    fireEvent.click(screen.getByRole("button", { name: "+2 more" }))
    expect(screen.getByText("src/g.ts")).toBeTruthy()
  })
})

describe("ConflictPanel states", () => {
  it("is calm and offers nothing while the agent proposes", () => {
    render(
      <ConflictPanel
        id="s"
        conflict={conflict({ state: "proposing", groups: [] })}
      />
    )
    expect(screen.getByText("The agent is reading both sides…")).toBeTruthy()
    expect(screen.queryAllByRole("button")).toHaveLength(0)
  })

  it("offers nothing while the choice is applied", () => {
    render(<ConflictPanel id="s" conflict={conflict({ state: "applying" })} />)
    expect(screen.getByText("Applying your choice…")).toBeTruthy()
    expect(screen.queryAllByRole("button")).toHaveLength(0)
  })

  it("shows the note and only Abort when it failed", () => {
    render(
      <ConflictPanel
        id="s"
        conflict={conflict({
          state: "failed",
          note: "the agent could not read the diff",
        })}
      />
    )
    expect(screen.getByText("the agent could not read the diff")).toBeTruthy()
    expect(
      screen.getAllByRole("button").map((button) => button.textContent)
    ).toEqual(["Abort merge"])
  })

  it("names the base branch in the header", () => {
    render(<ConflictPanel id="s" conflict={conflict()} />)
    expect(
      screen.getByRole("heading", {
        name: "Merging main into this branch hits a conflict in 2 files",
      })
    ).toBeTruthy()
    expect(screen.getByText("If you apply 0 choices")).toBeTruthy()
  })
})

describe("ConflictPanel compare options", () => {
  afterEach(() => {
    cleanup()
  })

  const lone = () => conflict({ groups: [group()] })

  it("is off until asked, then shows every option's diff and marks the chosen one", () => {
    render(<ConflictPanel id="s" conflict={lone()} />)
    expect(screen.queryByRole("group", { name: "Compare options" })).toBeNull()
    const toggle = screen.getByRole("button", { name: "Compare options" })
    expect(toggle.getAttribute("aria-pressed")).toBe("false")
    fireEvent.click(toggle)
    expect(toggle.getAttribute("aria-pressed")).toBe("true")
    const compare = screen.getByRole("group", { name: "Compare options" })
    expect(
      within(compare).getAllByRole("button", { name: "Choose" })
    ).toHaveLength(2)
    fireEvent.click(
      within(compare).getAllByRole("button", { name: "Choose" })[1] ?? compare
    )
    expect(
      within(compare)
        .getByRole("button", { name: "Chosen" })
        .getAttribute("aria-pressed")
    ).toBe("true")
    expect(
      (
        screen.getByRole("radio", {
          name: /Take the base total/,
        }) as HTMLInputElement
      ).checked
    ).toBe(true)
    expect(within(compare).getByRole("status").textContent).toContain(
      "Chosen: option B"
    )
  })

  it("highlights the lines the options do not share", () => {
    const own = group({
      options: [
        option({ diff: "@@ -1 +1 @@\n-x\n+y\n+shared\n" }),
        option({
          id: "theirs",
          title: "Take the base total",
          diff: "@@ -1 +1 @@\n-x\n+z\n+shared\n",
        }),
      ],
    })
    render(<ConflictPanel id="s" conflict={conflict({ groups: [own] })} />)
    fireEvent.click(screen.getByRole("button", { name: "Compare options" }))
    expect(screen.getByText("+y").closest("[data-differs]")).not.toBeNull()
    expect(screen.getByText("+z").closest("[data-differs]")).not.toBeNull()
    expect(
      screen.getAllByText("+shared")[0]?.closest("[data-differs]")
    ).toBeNull()
  })

  it("moves between options with the arrow keys and says where it is", () => {
    render(<ConflictPanel id="s" conflict={lone()} />)
    fireEvent.click(screen.getByRole("button", { name: "Compare options" }))
    const compare = screen.getByRole("group", { name: "Compare options" })
    const [first, second] = within(compare).getAllByRole("button", {
      name: "Choose",
    })
    first?.focus()
    fireEvent.keyDown(first ?? compare, { key: "ArrowRight" })
    expect(document.activeElement).toBe(second)
    expect(within(compare).getByRole("status").textContent).toContain(
      "Option B, Take the base total, high risk"
    )
    fireEvent.keyDown(second ?? compare, { key: "ArrowRight" })
    expect(document.activeElement).toBe(first)
  })
})
