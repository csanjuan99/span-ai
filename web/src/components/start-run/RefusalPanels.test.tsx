import { cleanup, fireEvent, render, screen } from "@testing-library/react"
import { MemoryRouter } from "react-router"
import { afterEach, describe, expect, it, vi } from "vitest"

import { RefusalPanels } from "@/components/start-run/RefusalPanels"
import { suggestBranch, type StartRefusal } from "@/components/start-run/types"

afterEach(cleanup)

// jsdom has no layout, so the scroll the panel asks for is a no-op to observe.
Element.prototype.scrollIntoView = vi.fn()

function show(refusal: StartRefusal, extra?: { suggestion?: string }) {
  const onDismiss = vi.fn()
  const onUseBranch = vi.fn()
  render(
    <MemoryRouter>
      <RefusalPanels
        refusal={refusal}
        onDismiss={onDismiss}
        suggestion={extra?.suggestion}
        onUseBranch={onUseBranch}
      />
    </MemoryRouter>
  )
  return { onDismiss, onUseBranch }
}

describe("RefusalPanels", () => {
  it("names the run a refusal is about in its title and its button", () => {
    show({
      number: 6,
      code: "run_exists",
      message: "a run for SPAN-479 already exists",
      runId: "SPAN-479",
    })
    expect(
      screen.getByRole("heading", { name: "A run for SPAN-479 already exists" })
    ).toBeTruthy()
    const open = screen.getByRole("link", { name: "Open SPAN-479" })
    expect(open.getAttribute("href")).toBe("/spans/SPAN-479")
  })

  it("says which run holds a busy checkout", () => {
    show({
      number: 16,
      code: "checkout_busy",
      message: "billing-api is busy with SPAN-495",
      runId: "SPAN-495",
    })
    expect(
      screen.getByRole("heading", {
        name: "This checkout is busy with SPAN-495",
      })
    ).toBeTruthy()
    expect(screen.getByRole("link", { name: "See SPAN-495" })).toBeTruthy()
  })

  it("leaves the green plus off what is kept, as the refusal boards do", () => {
    show({ number: 14, code: "not_git", message: "not a repo" })
    expect(
      screen.getByText("Nothing was created. Your files are untouched.")
        .textContent
    ).not.toContain("+")
    expect(document.querySelector(".text-pass")).toBeNull()
  })

  it("offers a branch that exists when the base is missing", () => {
    const { onDismiss, onUseBranch } = show(
      {
        number: 15,
        code: "base_missing",
        message: "no develop",
        base: "develop",
      },
      { suggestion: "main" }
    )
    fireEvent.click(screen.getByRole("button", { name: "Use main" }))
    expect(onUseBranch).toHaveBeenCalledWith("main")
    expect(onDismiss).toHaveBeenCalled()
  })

  it("hands the form back when the base is missing and no other branch exists", () => {
    const { onDismiss, onUseBranch } = show({
      number: 15,
      code: "base_missing",
      message: "no develop",
      base: "develop",
    })
    fireEvent.click(screen.getByRole("button", { name: "Back to the form" }))
    expect(onDismiss).toHaveBeenCalled()
    expect(onUseBranch).not.toHaveBeenCalled()
  })
})

describe("suggestBranch", () => {
  it("prefers main, then master, never the missing one", () => {
    expect(suggestBranch(["release", "master", "main"], "develop")).toBe("main")
    expect(suggestBranch(["release", "master"], "develop")).toBe("master")
    expect(suggestBranch(["develop", "release"], "develop")).toBe("release")
    expect(suggestBranch(["develop"], "develop")).toBe("")
  })
})
