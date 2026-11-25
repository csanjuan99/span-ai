import { useEffect, useRef, useState } from "react"

import {
  getRunContext,
  isTyping,
  nextChord,
  openPaletteEvent,
  runCommandFor,
  setRunContext,
  type RunContext,
} from "@/lib/help"

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
      if (pending.current !== "g") {
        const command = runCommandFor(
          getRunContext(),
          event.key,
          event.shiftKey
        )
        if (command !== undefined) {
          event.preventDefault()
          command.run()
          return
        }
      }
      const step = nextChord(pending.current, event.key.toLowerCase())
      pending.current = step.pending
      if (step.go !== "") {
        goRef.current(step.go)
      }
    }
    const onOpen = () => setPaletteOpen(true)
    window.addEventListener("keydown", onKey)
    window.addEventListener(openPaletteEvent, onOpen)
    return () => {
      window.removeEventListener("keydown", onKey)
      window.removeEventListener(openPaletteEvent, onOpen)
    }
  }, [])

  return { paletteOpen, setPaletteOpen, sheetOpen, setSheetOpen }
}

/**
 * Tells the palette what can be done to the run in front: the page calls it with the run's name
 * and its actions, and they appear first in the palette under "This run", with their keys. Pass
 * null while the run cannot be acted on. The actions go away when the page does.
 */
export function useRunCommands(context: RunContext | null): void {
  useEffect(() => setRunContext(context), [context])
}
