import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

const postForm = vi.fn()
const readJiraSession = vi.fn()

vi.mock("@/lib/api", () => ({
  postForm: (url: string, body: URLSearchParams) => postForm(url, body),
}))
vi.mock("@/lib/jira-session", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/jira-session")>()
  return {
    ...actual,
    readJiraSession: (folder: string) => readJiraSession(folder),
  }
})
vi.mock("@/components/settings-edit/fields", () => ({
  TextField: () => <p>done status field</p>,
}))

import { JiraSection } from "@/components/settings-edit/JiraSection"
import type { EditProps } from "@/components/settings-edit/fields"

afterEach(cleanup)
beforeEach(() => {
  postForm.mockReset()
  readJiraSession.mockReset()
  readJiraSession.mockResolvedValue({
    configured: true,
    email: "carlos@acme.com",
    baseUrl: "https://acme.atlassian.net",
  })
  postForm.mockResolvedValue(undefined)
})

function show() {
  render(<JiraSection folder="~/work/billing-api" props={{} as EditProps} />)
}

describe("JiraSection", () => {
  it("shows the account that is connected for the folder", async () => {
    show()
    expect(await screen.findByText("carlos@acme.com")).toBeTruthy()
    expect(screen.getByText("Connected")).toBeTruthy()
    expect(screen.getByText("The session belongs to this folder.")).toBeTruthy()
  })

  it("explains what closing does before it closes anything", async () => {
    show()
    fireEvent.click(
      await screen.findByRole("button", { name: "Close session" })
    )
    expect(postForm).not.toHaveBeenCalled()
    expect(
      screen.getByText("Close the Jira session for ~/work/billing-api?")
    ).toBeTruthy()
    expect(screen.getByText(/Nothing is deleted from Jira/)).toBeTruthy()
    fireEvent.click(screen.getByRole("button", { name: "Keep connected" }))
    expect(postForm).not.toHaveBeenCalled()
    expect(screen.getByRole("button", { name: "Close session" })).toBeTruthy()
  })

  it("closes the session of the folder once confirmed", async () => {
    show()
    fireEvent.click(
      await screen.findByRole("button", { name: "Close session" })
    )
    fireEvent.click(screen.getByRole("button", { name: "Close session" }))
    await waitFor(() => {
      expect(postForm).toHaveBeenCalledTimes(1)
    })
    const [url, body] = postForm.mock.calls[0] as [string, URLSearchParams]
    expect(url).toBe("/jira/logout")
    expect(body.get("project")).toBe("~/work/billing-api")
    expect(
      await screen.findByText("Session closed for this folder.")
    ).toBeTruthy()
    expect(screen.queryByRole("button", { name: "Close session" })).toBeNull()
  })

  it("says why a session was not closed and keeps it", async () => {
    postForm.mockRejectedValue(new Error("the server said no"))
    show()
    fireEvent.click(
      await screen.findByRole("button", { name: "Close session" })
    )
    fireEvent.click(screen.getByRole("button", { name: "Close session" }))
    expect(await screen.findByText("the server said no")).toBeTruthy()
    expect(screen.getByText("carlos@acme.com")).toBeTruthy()
  })
})
