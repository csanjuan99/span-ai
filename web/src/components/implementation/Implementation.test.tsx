import { render, screen } from "@testing-library/react"
import { describe, expect, it } from "vitest"

import { Implementation } from "@/components/implementation/Implementation"

describe("Implementation", () => {
  it("lists the files named in the diff stat", () => {
    render(
      <Implementation
        stat={" pay.go | 3 +\n 1 file changed, 3 insertions(+)"}
        patch="diff --git a/pay.go b/pay.go"
        truncated={false}
        transcript={["wrote the charge"]}
      />,
    )
    expect(screen.getByText("pay.go | 3 +")).toBeTruthy()
    expect(screen.queryByText(/1 file changed/)).toBeNull()
    expect(screen.getByText("wrote the charge")).toBeTruthy()
  })
})
