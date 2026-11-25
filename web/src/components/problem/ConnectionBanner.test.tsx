import { cleanup, render, screen } from "@testing-library/react"
import { afterEach, describe, expect, it } from "vitest"

import { ConnectionBanner } from "@/components/problem/ConnectionBanner"

afterEach(cleanup)

describe("ConnectionBanner", () => {
  it("says nothing while idle", () => {
    const { container } = render(<ConnectionBanner notice="idle" lastSeen="" />)
    expect(container.firstChild).toBeNull()
  })

  it("stays calm while it reconnects and offers a retry", () => {
    render(<ConnectionBanner notice="reconnecting" lastSeen="14:52" />)
    expect(screen.getByText("Reconnecting to Span…")).toBeTruthy()
    expect(screen.getByRole("button", { name: "Retry now" })).toBeTruthy()
  })

  it("says it closes by itself when it is back", () => {
    render(<ConnectionBanner notice="back" lastSeen="" />)
    expect(screen.getByText(/This message closes by itself/)).toBeTruthy()
    expect(screen.queryByRole("button")).toBeNull()
  })

  it("says nothing was lost when the server stops answering", () => {
    render(<ConnectionBanner notice="lost" lastSeen="14:52" />)
    expect(screen.getByText("Span is not answering.")).toBeTruthy()
    expect(
      screen.getByText(/Last seen 14:52\. Nothing was lost\./)
    ).toBeTruthy()
    expect(screen.getByRole("button", { name: "Retry" })).toBeTruthy()
  })
})
