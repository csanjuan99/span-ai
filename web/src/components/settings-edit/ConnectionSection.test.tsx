import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

const getJSON = vi.fn()
const rememberServeToken = vi.fn()

vi.mock("@/lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/api")>()
  return {
    UnauthorizedError: actual.UnauthorizedError,
    getJSON: (url: string) => getJSON(url),
    rememberServeToken: (value: string) => rememberServeToken(value),
    serveToken: () => "kept",
  }
})

import { UnauthorizedError } from "@/lib/api"
import { ConnectionSection } from "@/components/settings-edit/ConnectionSection"

afterEach(cleanup)
beforeEach(() => {
  getJSON.mockReset()
  rememberServeToken.mockReset()
})

describe("ConnectionSection", () => {
  it("starts unchecked and shows the address serving the page", () => {
    render(<ConnectionSection />)
    expect(screen.getByText("Not checked")).toBeTruthy()
    const server = screen.getByLabelText("Server address")
    expect((server as HTMLInputElement).value).toBe(window.location.origin)
    expect((server as HTMLInputElement).readOnly).toBe(true)
  })

  it("reports a connected server with its version", async () => {
    getJSON.mockResolvedValue({ status: "connected", version: "0.9.2" })
    render(<ConnectionSection />)
    fireEvent.click(screen.getByRole("button", { name: "Test" }))
    expect(await screen.findByText("Connected")).toBeTruthy()
    expect(screen.getByText(/Span 0\.9\.2/)).toBeTruthy()
    expect(getJSON).toHaveBeenCalledWith("/health")
  })

  it("says a refused token is refused, not unreachable", async () => {
    getJSON.mockRejectedValue(new UnauthorizedError())
    render(<ConnectionSection />)
    fireEvent.click(screen.getByRole("button", { name: "Test" }))
    expect(
      await screen.findByText(/Refused · the server wants an access token/)
    ).toBeTruthy()
    expect(
      screen.queryByText("Unreachable", { selector: "span.rounded-full" })
    ).toBeNull()
  })

  it("says an unreachable server is unreachable and why", async () => {
    getJSON.mockRejectedValue(new Error("connection refused"))
    render(<ConnectionSection />)
    fireEvent.click(screen.getByRole("button", { name: "Test" }))
    await waitFor(() => {
      expect(screen.getByText("connection refused")).toBeTruthy()
    })
  })

  it("keeps the token in this tab only and says when it is cleared", () => {
    render(<ConnectionSection />)
    const field = screen.getByLabelText("Access token")
    fireEvent.change(field, { target: { value: " s3cret " } })
    fireEvent.click(screen.getByRole("button", { name: "Save token" }))
    expect(rememberServeToken).toHaveBeenCalledWith("s3cret")
    expect(screen.getByText("Access token saved for this tab.")).toBeTruthy()
    fireEvent.change(field, { target: { value: "" } })
    fireEvent.click(screen.getByRole("button", { name: "Save token" }))
    expect(rememberServeToken).toHaveBeenLastCalledWith("")
    expect(screen.getByText("Access token cleared.")).toBeTruthy()
  })
})
