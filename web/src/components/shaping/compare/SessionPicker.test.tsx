import { cleanup, fireEvent, render, screen } from "@testing-library/react"
import { afterEach, describe, expect, it, vi } from "vitest"

import { SessionPicker } from "@/components/shaping/compare/SessionPicker"
import type { ShapeSession } from "@/lib/types"

afterEach(cleanup)

function session(
  patch: Partial<ShapeSession> & Pick<ShapeSession, "id" | "repo" | "status">
): ShapeSession {
  return {
    workspace: ".",
    idea: patch.title ?? patch.id,
    rules: [],
    examples: [],
    questions: [],
    split: [],
    proposals: [],
    story: "",
    digest: "",
    jira: null,
    ref: patch.ref ?? patch.id,
    title: patch.title ?? patch.id,
    ...patch,
  }
}

const api = session({
  id: "a",
  ref: "SES-A",
  title: "Retry webhooks",
  repo: "api",
  status: "ready",
})
const web = session({
  id: "b",
  ref: "SES-B",
  title: "Show the log",
  repo: "api",
  status: "interviewing",
})
const other = session({
  id: "c",
  ref: "SES-C",
  title: "Tax lines",
  repo: "billing",
  status: "sealed",
})

describe("sessions to compare", () => {
  it("checks a session and refuses one from another repository", () => {
    const onChange = vi.fn()
    render(
      <SessionPicker
        sessions={[api, web, other]}
        chosen={["a"]}
        onChange={onChange}
      />
    )
    expect(screen.getByText(/Same repo only: api/)).toBeTruthy()
    const foreign = screen.getByRole("checkbox", { name: "Compare SES-C" })
    expect(foreign).toHaveProperty("disabled", true)
    fireEvent.click(foreign)
    expect(onChange).not.toHaveBeenCalled()
    fireEvent.click(screen.getByRole("checkbox", { name: "Compare SES-B" }))
    expect(onChange).toHaveBeenCalledWith(["a", "b"])
  })

  it("adds a session from the menu and stops at five", () => {
    const onChange = vi.fn()
    const many = ["a", "b", "d", "e", "f"].map((id) =>
      session({ id, ref: `SES-${id}`, repo: "api", status: "ready" })
    )
    const { rerender } = render(
      <SessionPicker sessions={many} chosen={[]} onChange={onChange} />
    )
    fireEvent.change(screen.getByLabelText("Add session"), {
      target: { value: "b" },
    })
    expect(onChange).toHaveBeenCalledWith(["b"])
    rerender(
      <SessionPicker
        sessions={many}
        chosen={["a", "b", "d", "e", "f"]}
        onChange={onChange}
      />
    )
    expect(screen.getByLabelText("Add session")).toHaveProperty("disabled", true)
    fireEvent.click(screen.getByRole("checkbox", { name: "Compare SES-a" }))
    expect(onChange).toHaveBeenLastCalledWith(["b", "d", "e", "f"])
  })
})
