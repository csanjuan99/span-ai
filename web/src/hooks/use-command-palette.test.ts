import { act, renderHook } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"

import { useCommandPalette } from "@/hooks/use-command-palette"

function press(
  key: string,
  init: KeyboardEventInit = {},
  target: EventTarget = document.body
) {
  act(() => {
    target.dispatchEvent(
      new KeyboardEvent("keydown", { key, bubbles: true, ...init })
    )
  })
}

describe("useCommandPalette", () => {
  it("opens the palette on Cmd+K and on Ctrl+K", () => {
    const { result } = renderHook(() => useCommandPalette(() => undefined))
    press("k", { metaKey: true })
    expect(result.current.paletteOpen).toBe(true)
    press("Escape")
    expect(result.current.paletteOpen).toBe(false)
    press("k", { ctrlKey: true })
    expect(result.current.paletteOpen).toBe(true)
  })

  it("opens the shortcut sheet on ? and closes it on Escape", () => {
    const { result } = renderHook(() => useCommandPalette(() => undefined))
    press("?", { shiftKey: true })
    expect(result.current.sheetOpen).toBe(true)
    press("Escape")
    expect(result.current.sheetOpen).toBe(false)
  })

  it("goes to the desk and to settings with g then d or s", () => {
    const go = vi.fn()
    renderHook(() => useCommandPalette(go))
    press("g")
    press("d")
    press("g")
    press("s")
    expect(go.mock.calls).toEqual([["/"], ["/settings"]])
  })

  it("stays out of the way while the person types", () => {
    const go = vi.fn()
    const { result } = renderHook(() => useCommandPalette(go))
    const field = document.createElement("input")
    document.body.append(field)
    press("?", { shiftKey: true }, field)
    press("g", {}, field)
    press("d", {}, field)
    expect(result.current.sheetOpen).toBe(false)
    expect(go).not.toHaveBeenCalled()
    field.remove()
  })

  it("still opens the palette from a field, since Cmd+K is not a letter", () => {
    const { result } = renderHook(() => useCommandPalette(() => undefined))
    const field = document.createElement("input")
    document.body.append(field)
    press("k", { metaKey: true }, field)
    expect(result.current.paletteOpen).toBe(true)
    field.remove()
  })
})
