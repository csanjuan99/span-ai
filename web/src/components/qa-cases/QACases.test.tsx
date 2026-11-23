import { render, screen } from "@testing-library/react"
import { describe, expect, it } from "vitest"

import { QACases } from "@/components/qa-cases/QACases"

describe("QACases", () => {
  it("shows a covered declaration and leaves a missing one unverified", () => {
    render(
      <QACases
        skipped={false}
        command="go test ./..."
        exitCode={0}
        lines={[
          { id: "AC-1", text: "A charge records the amount.", result: "pass", file: "pay_test.go", line: 3 },
          { id: "AC-2", text: "A refund records the reason.", result: "unverified", file: "", line: 0 },
        ]}
      />,
    )
    expect(screen.getByText("AC-1")).toBeTruthy()
    expect(screen.getByText("pay_test.go:3")).toBeTruthy()
    expect(screen.getByText("AC-2")).toBeTruthy()
    expect(screen.getByText("unverified")).toBeTruthy()
    expect(screen.queryByText(":0")).toBeNull()
    expect(screen.getByText("go test ./... · exit 0")).toBeTruthy()
  })
})
