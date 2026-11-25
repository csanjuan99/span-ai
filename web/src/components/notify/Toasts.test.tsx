import { cleanup, fireEvent, render, screen } from "@testing-library/react"
import { afterEach, describe, expect, it, vi } from "vitest"

import { Toasts } from "@/components/notify/Toasts"
import type { Toast } from "@/hooks/use-notifications"

const toast: Toast = { id: 1, kind: "needs", runId: "r1", title: "SPAN-488 needs you", detail: "shape · 3 questions", action: "Review" }

afterEach(cleanup)

describe("Toasts", () => {
  it("shows nothing when there is nothing to say", () => {
    const { container } = render(<Toasts toasts={[]} onOpen={vi.fn()} onDismiss={vi.fn()} onHold={vi.fn()} />)
    expect(container.innerHTML).toBe("")
  })

  it("says who and what, and opens the run from the action", () => {
    const onOpen = vi.fn()
    const onDismiss = vi.fn()
    render(<Toasts toasts={[toast]} onOpen={onOpen} onDismiss={onDismiss} onHold={vi.fn()} />)
    expect(screen.getByText("SPAN-488 needs you")).toBeTruthy()
    expect(screen.getByText("shape · 3 questions")).toBeTruthy()
    fireEvent.click(screen.getByRole("button", { name: "Review" }))
    expect(onOpen).toHaveBeenCalledWith("r1")
    expect(onDismiss).toHaveBeenCalledWith(1)
  })

  it("closes with the close button", () => {
    const onDismiss = vi.fn()
    render(<Toasts toasts={[toast]} onOpen={vi.fn()} onDismiss={onDismiss} onHold={vi.fn()} />)
    fireEvent.click(screen.getByRole("button", { name: "Close notification" }))
    expect(onDismiss).toHaveBeenCalledWith(1)
  })

  it("holds a toast while the pointer is over it", () => {
    const onHold = vi.fn()
    render(<Toasts toasts={[toast]} onOpen={vi.fn()} onDismiss={vi.fn()} onHold={onHold} />)
    const item = screen.getByText("SPAN-488 needs you").closest("li") as HTMLElement
    fireEvent.mouseEnter(item)
    expect(onHold).toHaveBeenLastCalledWith(1, true)
    fireEvent.mouseLeave(item)
    expect(onHold).toHaveBeenLastCalledWith(1, false)
  })
})
