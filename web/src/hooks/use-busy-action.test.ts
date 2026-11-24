import { act, renderHook } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"

import { useBusyAction } from "@/hooks/use-busy-action"

describe("useBusyAction", () => {
  it("is busy while the task runs, calls onStart first, and is free after", async () => {
    const onStart = vi.fn()
    const onError = vi.fn()
    const { result } = renderHook(() => useBusyAction({ onStart, onError }))
    let release: () => void = () => undefined
    let pending: Promise<void> = Promise.resolve()
    act(() => {
      pending = result.current.run(
        () => new Promise<void>((resolve) => (release = resolve))
      )
    })
    expect(result.current.busy).toBe(true)
    expect(onStart).toHaveBeenCalledTimes(1)
    await act(async () => {
      release()
      await pending
    })
    expect(result.current.busy).toBe(false)
    expect(onError).not.toHaveBeenCalled()
  })

  it("hands a failure and its fallback to onError and frees the action", async () => {
    const onError = vi.fn()
    const { result } = renderHook(() => useBusyAction({ onError }))
    const failure = new Error("boom")
    await act(async () => {
      await result.current.run(() => Promise.reject(failure), "did not start")
    })
    expect(onError).toHaveBeenCalledWith(failure, "did not start")
    expect(result.current.busy).toBe(false)
  })

  it("stays busy after a success when asked to, because the page is leaving", async () => {
    const { result } = renderHook(() =>
      useBusyAction({ keepBusyOnSuccess: true, onError: vi.fn() })
    )
    await act(async () => {
      await result.current.run(() => Promise.resolve())
    })
    expect(result.current.busy).toBe(true)
  })

  it("still frees a keep-busy action when it fails", async () => {
    const { result } = renderHook(() =>
      useBusyAction({ keepBusyOnSuccess: true, onError: vi.fn() })
    )
    await act(async () => {
      await result.current.run(() => Promise.reject(new Error("boom")))
    })
    expect(result.current.busy).toBe(false)
  })
})
