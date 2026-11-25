"use client"

import { useEffect, useState } from "react"

/**
 * True below the `md` breakpoint. A missing matchMedia, as in some tests, stays wide
 * so the desktop layout is what those tests render.
 */
export function useNarrow(): boolean {
  const [narrow, setNarrow] = useState(false)
  useEffect(() => {
    if (typeof window.matchMedia !== "function") {
      return
    }
    const query = window.matchMedia("(max-width: 767px)")
    const apply = () => setNarrow(query.matches)
    apply()
    query.addEventListener("change", apply)
    return () => query.removeEventListener("change", apply)
  }, [])
  return narrow
}
