import { act, renderHook } from "@testing-library/react"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

import { useNotifications } from "@/hooks/use-notifications"
import { savePrefs, defaultPrefs } from "@/lib/notify"
import type { SpanHead } from "@/lib/types"
import { head } from "@/test/heads"

const running = head({ id: "a", issue: "SPAN-1", verdict: "running" })
const waiting: SpanHead = { ...running, verdict: "hold", stage: "shape", reason: "3 questions" }
const finished: SpanHead = { ...running, verdict: "done", stage: "note" }

function inFront(front: boolean) {
  vi.spyOn(document, "visibilityState", "get").mockReturnValue(front ? "visible" : "hidden")
  vi.spyOn(document, "hasFocus").mockReturnValue(front)
}

beforeEach(() => {
  vi.useFakeTimers()
  localStorage.clear()
  document.head.innerHTML = '<title>Span original</title><link rel="icon" href="/favicon.svg">'
  inFront(false)
  vi.spyOn(HTMLCanvasElement.prototype, "getContext").mockReturnValue(null)
})
afterEach(() => {
  vi.useRealTimers()
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

describe("useNotifications", () => {
  it("does not announce the runs it finds when the desk opens", () => {
    const { result } = renderHook(({ runs }) => useNotifications(runs), { initialProps: { runs: [waiting] } })
    expect(result.current.toasts).toEqual([])
  })

  it("raises a toast when a run starts needing me, and keeps it until it is closed", () => {
    const { result, rerender } = renderHook(({ runs }) => useNotifications(runs), { initialProps: { runs: [running] } })
    rerender({ runs: [waiting] })
    expect(result.current.toasts).toHaveLength(1)
    expect(result.current.toasts[0].title).toBe("SPAN-1 needs you")
    act(() => {
      vi.advanceTimersByTime(60_000)
    })
    expect(result.current.toasts).toHaveLength(1)
    act(() => result.current.dismiss(result.current.toasts[0].id))
    expect(result.current.toasts).toEqual([])
  })

  it("lets a finished toast go by itself after eight seconds", () => {
    const { result, rerender } = renderHook(({ runs }) => useNotifications(runs), { initialProps: { runs: [running] } })
    rerender({ runs: [finished] })
    expect(result.current.toasts).toHaveLength(1)
    act(() => {
      vi.advanceTimersByTime(8100)
    })
    expect(result.current.toasts).toEqual([])
  })

  it("holds toasts back while the tab is in front, but still marks the tab", () => {
    inFront(true)
    const { result, rerender } = renderHook(({ runs }) => useNotifications(runs), { initialProps: { runs: [running] } })
    rerender({ runs: [waiting] })
    expect(result.current.toasts).toEqual([])
    expect(document.title).toBe("(1) SPAN-1 needs you · Span")
  })

  it("restores the title when nothing needs me any more", () => {
    const { rerender } = renderHook(({ runs }) => useNotifications(runs), { initialProps: { runs: [running] } })
    rerender({ runs: [waiting] })
    rerender({ runs: [running] })
    expect(document.title).toBe("Span original")
  })

  it("leaves the tab alone when the person turned the tab off for that event", () => {
    savePrefs({ ...defaultPrefs, events: { ...defaultPrefs.events, needs: { toast: true, tab: false, system: false } } })
    const { rerender } = renderHook(({ runs }) => useNotifications(runs), { initialProps: { runs: [running] } })
    rerender({ runs: [waiting] })
    expect(document.title).toBe("Span original")
  })

  it("puts the badge on the page icon, and puts the original back", () => {
    const ctx = new Proxy({}, { get: () => () => undefined, set: () => true }) as unknown as CanvasRenderingContext2D
    vi.spyOn(HTMLCanvasElement.prototype, "getContext").mockReturnValue(ctx)
    vi.spyOn(HTMLCanvasElement.prototype, "toDataURL").mockReturnValue("data:image/png;base64,AAA")
    const { rerender } = renderHook(({ runs }) => useNotifications(runs), { initialProps: { runs: [running] } })
    const icon = () => document.head.querySelector<HTMLLinkElement>('link[rel="icon"]')?.getAttribute("href")
    rerender({ runs: [waiting] })
    expect(icon()).toBe("data:image/png;base64,AAA")
    rerender({ runs: [running] })
    expect(icon()).toBe("/favicon.svg")
  })

  it("never asks the browser for permission and does not break without Notification", () => {
    vi.stubGlobal("Notification", undefined)
    const { rerender } = renderHook(({ runs }) => useNotifications(runs), { initialProps: { runs: [running] } })
    expect(() => rerender({ runs: [waiting] })).not.toThrow()
  })

  it("sends a system notification only when it is on and the browser allowed it", () => {
    const made = vi.fn()
    class Fake {
      static permission = "granted"
      constructor(title: string) {
        made(title)
      }
    }
    vi.stubGlobal("Notification", Fake)
    savePrefs({ ...defaultPrefs, events: { ...defaultPrefs.events, needs: { toast: true, tab: true, system: true } } })
    const { rerender } = renderHook(({ runs }) => useNotifications(runs), { initialProps: { runs: [running] } })
    rerender({ runs: [waiting] })
    expect(made).toHaveBeenCalledWith("SPAN-1 needs you")
  })
})
