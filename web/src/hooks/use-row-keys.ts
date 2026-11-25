"use client"

import { useEffect } from "react"

import { isTyping } from "@/lib/help"

/** The attribute that marks the link a row of a list opens, so J and K can walk the list. */
const rowLink = "[data-row-link]"

/**
 * J moves focus to the next row of the list in front and K to the previous one; Enter then opens
 * the focused row, because the row is a link. The keys never act while a field has focus.
 */
export function useRowKeys(): void {
  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      const key = event.key.toLowerCase()
      if (
        (key !== "j" && key !== "k") ||
        event.metaKey ||
        event.ctrlKey ||
        event.altKey ||
        isTyping(event.target)
      ) {
        return
      }
      const rows = Array.from(document.querySelectorAll<HTMLElement>(rowLink))
      if (rows.length === 0) {
        return
      }
      const at = rows.findIndex((row) => row === document.activeElement)
      const next =
        key === "j"
          ? Math.min(at + 1, rows.length - 1)
          : Math.max(at === -1 ? 0 : at - 1, 0)
      event.preventDefault()
      rows[next]?.focus()
    }
    window.addEventListener("keydown", onKey)
    return () => window.removeEventListener("keydown", onKey)
  }, [])
}
