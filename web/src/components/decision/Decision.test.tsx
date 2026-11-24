import { cleanup, fireEvent, render, screen } from "@testing-library/react"
import { afterEach, describe, expect, it, vi } from "vitest"

import { Decision } from "@/components/decision/Decision"
import { decisionOf } from "@/lib/api"
import type { DecisionRequest } from "@/lib/types"

const request: DecisionRequest = {
  id: "d1",
  class: "spec-conflict",
  stage: "build",
  title: "AC 400 contradicts AGENTS.md",
  happened: "The brief wants the button enabled.",
  evidence: [{ ref: "AGENTS.md:12", excerpt: "disable submit while invalid" }],
  options: [
    { id: "follow-rule", label: "Follow AGENTS.md", consequence: "The button is disabled.", reversible: true },
    { id: "stop", label: "Stop", consequence: "The run ends.", reversible: false },
  ],
  recommendation: "follow-rule",
}

describe("Decision", () => {
  afterEach(() => {
    cleanup()
  })

  it("preselects the recommendation and sends the chosen option with the words", async () => {
    const onDecide = vi.fn(() => Promise.resolve())
    render(<Decision request={request} note="" onDecide={onDecide} />)
    expect(screen.getByText("disable submit while invalid")).toBeTruthy()
    expect((screen.getByRole("radio", { name: /Follow AGENTS.md/ }) as HTMLInputElement).checked).toBe(true)
    fireEvent.change(screen.getByLabelText("Your words for the agent (optional)"), { target: { value: "keep it disabled" } })
    fireEvent.click(screen.getByRole("button", { name: "Choose and continue" }))
    await screen.findByRole("button", { name: "Choose and continue" })
    expect(onDecide).toHaveBeenCalledWith("follow-rule", "keep it disabled")
  })

  it("lets the person pick another option and warns when it cannot be undone", async () => {
    const onDecide = vi.fn(() => Promise.resolve())
    render(<Decision request={request} note="" onDecide={onDecide} />)
    fireEvent.click(screen.getByRole("radio", { name: /Stop/ }))
    expect(screen.getByText("This cannot be undone from here.")).toBeTruthy()
    fireEvent.click(screen.getByRole("button", { name: "Choose and continue" }))
    await screen.findByRole("button", { name: "Choose and continue" })
    expect(onDecide).toHaveBeenCalledWith("stop", "")
  })
})

describe("decisionOf", () => {
  it("reads a request and drops one nobody can answer", () => {
    expect(decisionOf(request)?.options).toHaveLength(2)
    expect(decisionOf({ ...request, options: [request.options[0]] })).toBeUndefined()
    expect(decisionOf(null)).toBeUndefined()
    expect(decisionOf({ ...request, options: [{ id: "", label: "" }, request.options[0]] })).toBeUndefined()
  })
})
