import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

import type { ShapeSession } from "@/lib/types"

const resumeShape = vi.fn()
const getJSON = vi.fn()

vi.mock("@/lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/api")>()
  return {
    ...actual,
    resumeShape: (...args: unknown[]) => resumeShape(...args),
    getJSON: (url: string) => getJSON(url),
  }
})

import { InterviewStopped } from "@/components/shape-entry/InterviewStopped"

afterEach(cleanup)

const source = { kind: "prompt" as const, ref: "idea" }
const stopped: ShapeSession = {
  id: "sh-1",
  workspace: "/w",
  repo: "api",
  idea: "x",
  status: "interviewing",
  rules: [
    { id: "R1", text: "a", source },
    { id: "R2", text: "b", source },
  ],
  examples: [{ id: "E1", rule: "R1", text: "c", source }],
  questions: [
    {
      id: "Q1",
      text: "",
      blocks: "",
      assume: "",
      status: "answered",
      answer: "a",
    },
  ],
  split: [],
  proposals: [],
  story: "",
  digest: "",
  jira: null,
  agent: "claude-code",
  ref: "SES-0142",
  failure: {
    at: "2026-11-25T09:36:00",
    agent: "claude-code",
    kind: "usage_limit",
    message: "usage limit reached, resets at 18:00",
  },
}

beforeEach(() => {
  resumeShape.mockReset()
  getJSON.mockReset()
  getJSON.mockImplementation((url: string) =>
    Promise.resolve(
      url.startsWith("/agent?")
        ? {
            agents: [
              {
                id: "claude-code",
                label: "Claude Code",
                binary: "claude",
                installed: true,
                gentle: true,
                ready: true,
              },
              {
                id: "codex",
                label: "Codex",
                binary: "codex",
                installed: true,
                gentle: true,
                ready: true,
              },
            ],
            selected: "claude-code",
            chosen: "claude-code",
            exists: true,
            reason: "",
            models: {},
          }
        : {}
    )
  )
})

describe("interview stopped", () => {
  it("says why it stopped and what was kept", () => {
    render(
      <InterviewStopped
        session={stopped}
        onResumed={vi.fn()}
        onLater={vi.fn()}
      />
    )
    expect(
      screen.getByText(
        /claude-code reported a usage limit at 09:36\. It resets at 18:00\./
      )
    ).toBeTruthy()
    expect(
      screen.getByText("2 rules and 1 example, each with its source")
    ).toBeTruthy()
    expect(screen.getByText("1 question: 1 answered, 0 assumed")).toBeTruthy()
    expect(
      screen.getByRole("button", { name: "Wait until 18:00" })
    ).toBeTruthy()
  })

  it("resumes on the agent picked for the rest of the session", async () => {
    const next = { ...stopped, failure: null, agent: "codex" }
    resumeShape.mockResolvedValue(next)
    const onResumed = vi.fn()
    render(
      <InterviewStopped
        session={stopped}
        onResumed={onResumed}
        onLater={vi.fn()}
      />
    )
    await screen.findByRole("option", { name: "codex" })
    fireEvent.change(screen.getByLabelText("Agent"), {
      target: { value: "codex" },
    })
    fireEvent.click(screen.getByRole("button", { name: "Resume with codex" }))
    await waitFor(() => expect(onResumed).toHaveBeenCalledWith(next))
    expect(resumeShape).toHaveBeenCalledWith("sh-1", "/w", "codex", "")
  })

  it("tells when the resumed turn fails again", async () => {
    resumeShape.mockRejectedValue(
      new Error("claude-code is still over its limit")
    )
    render(
      <InterviewStopped
        session={stopped}
        onResumed={vi.fn()}
        onLater={vi.fn()}
      />
    )
    fireEvent.click(
      screen.getByRole("button", { name: "Resume with claude-code" })
    )
    expect((await screen.findByRole("alert")).textContent).toContain(
      "still over its limit"
    )
  })

  it("shows nothing for a session that did not stop", () => {
    const { container } = render(
      <InterviewStopped
        session={{ ...stopped, failure: null }}
        onResumed={vi.fn()}
        onLater={vi.fn()}
      />
    )
    expect(container.textContent).toBe("")
  })
})
