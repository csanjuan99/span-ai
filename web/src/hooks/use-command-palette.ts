import { useEffect, useRef, useState } from "react"

import { isTyping, nextChord } from "@/lib/help"

/** What the palette and the shortcut sheet need: whether each is open, and a way to change that. */
export interface CommandPaletteState {
  paletteOpen: boolean
  setPaletteOpen: (open: boolean) => void
  sheetOpen: boolean
  setSheetOpen: (open: boolean) => void
}

/**
 * The page's keyboard: Cmd/Ctrl+K opens the palette, ? the shortcut sheet, G then D or S goes
 * to the desk or settings, and Escape closes what is open. Letters never act while a field has
 * focus; Cmd/Ctrl+K does, because it cannot be typed by accident.
 */
export function useCommandPalette(
  go: (path: string) => void
): CommandPaletteState {
  const [paletteOpen, setPaletteOpen] = useState(false)
  const [sheetOpen, setSheetOpen] = useState(false)
  const pending = useRef("")
  // The listener is attached once; it reads the latest `go` through a ref instead of re-subscribing.
  const goRef = useRef(go)
  useEffect(() => {
    goRef.current = go
  }, [go])

  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "k") {
        event.preventDefault()
        setPaletteOpen((open) => !open)
        return
      }
      if (event.key === "Escape") {
        setPaletteOpen(false)
        setSheetOpen(false)
        return
      }
      if (
        event.metaKey ||
        event.ctrlKey ||
        event.altKey ||
        isTyping(event.target)
      ) {
        return
      }
      if (event.key === "?") {
        setSheetOpen(true)
        return
      }
      const step = nextChord(pending.current, event.key.toLowerCase())
      pending.current = step.pending
      if (step.go !== "") {
        goRef.current(step.go)
      }
    }
    window.addEventListener("keydown", onKey)
    return () => window.removeEventListener("keydown", onKey)
  }, [])

  return { paletteOpen, setPaletteOpen, sheetOpen, setSheetOpen }
}
