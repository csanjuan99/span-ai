import { useEffect, useState } from "react"

/** What the page says about its live connection: nothing, that it is trying again, or that it is back. */
export type ConnectionNotice = "idle" | "reconnecting" | "back"

const backMs = 3000

/**
 * A connection that never opened is the page still loading, so it says nothing. One that opened
 * and dropped is `reconnecting`; when it returns, `back` shows for a few seconds and clears.
 */
export function useConnectionNotice(connected: boolean): ConnectionNotice {
  const [seen, setSeen] = useState(connected)
  const [wasDown, setWasDown] = useState(false)
  const [back, setBack] = useState(false)
  const [previous, setPrevious] = useState(connected)
  if (previous !== connected) {
    setPrevious(connected)
    if (connected) {
      setSeen(true)
      if (seen || wasDown) {
        setBack(true)
      }
    } else if (seen) {
      setWasDown(true)
    }
  }
  if (connected && !seen) {
    setSeen(true)
  }
  useEffect(() => {
    if (!back) {
      return undefined
    }
    const timer = window.setTimeout(() => setBack(false), backMs)
    return () => window.clearTimeout(timer)
  }, [back])
  if (!connected && seen) {
    return "reconnecting"
  }
  return connected && back ? "back" : "idle"
}
