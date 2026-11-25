import { act, renderHook } from "@testing-library/react"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

import { useConnectionNotice } from "@/hooks/use-connection-notice"

beforeEach(() => vi.useFakeTimers())
afterEach(() => vi.useRealTimers())

describe("useConnectionNotice", () => {
  it("says nothing while the first connection is still opening", () => {
    const { result } = renderHook(({ on }) => useConnectionNotice(on), {
      initialProps: { on: false },
    })
    expect(result.current).toBe("idle")
  })

  it("says nothing when the first connection opens", () => {
    const { result, rerender } = renderHook(
      ({ on }) => useConnectionNotice(on),
      { initialProps: { on: false } }
    )
    rerender({ on: true })
    expect(result.current).toBe("idle")
  })

  it("says it is reconnecting after a drop, and back for three seconds", () => {
    const { result, rerender } = renderHook(
      ({ on }) => useConnectionNotice(on),
      { initialProps: { on: true } }
    )
    rerender({ on: false })
    expect(result.current).toBe("reconnecting")
    rerender({ on: true })
    expect(result.current).toBe("back")
    act(() => {
      vi.advanceTimersByTime(3100)
    })
    expect(result.current).toBe("idle")
  })

  it("hands over to the error screen after a minute of reconnecting", () => {
    const { result, rerender } = renderHook(
      ({ on }) => useConnectionNotice(on),
      { initialProps: { on: true } }
    )
    rerender({ on: false })
    act(() => {
      vi.advanceTimersByTime(59_000)
    })
    expect(result.current).toBe("reconnecting")
    act(() => {
      vi.advanceTimersByTime(1_500)
    })
    expect(result.current).toBe("lost")
    rerender({ on: true })
    expect(result.current).toBe("back")
  })

  it("starts counting again at the next drop", () => {
    const { result, rerender } = renderHook(
      ({ on }) => useConnectionNotice(on),
      { initialProps: { on: true } }
    )
    rerender({ on: false })
    act(() => {
      vi.advanceTimersByTime(30_000)
    })
    rerender({ on: true })
    rerender({ on: false })
    act(() => {
      vi.advanceTimersByTime(45_000)
    })
    expect(result.current).toBe("reconnecting")
  })
})
