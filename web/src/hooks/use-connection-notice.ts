import { useEffect, useState } from "react"

/**
 * What the page says about its live connection: nothing, that it is trying again, that it is
 * back, or that a minute passed and the server is not answering.
 */
export type ConnectionNotice = "idle" | "reconnecting" | "back" | "lost"

const backMs = 3000
const lostMs = 60_000

/**
 * A connection that never opened is the page still loading, so it says nothing. One that opened
 * and dropped is `reconnecting`; when it returns, `back` shows for a few seconds and clears.
 */
export function useConnectionNotice(connected: boolean): ConnectionNotice {
  const [seen, setSeen] = useState(connected)
  const [wasDown, setWasDown] = useState(false)
  const [back, setBack] = useState(false)
  const [previous, setPrevious] = useState(connected)
  const [lost, setLost] = useState(false)
  if (previous !== connected) {
    setPrevious(connected)
    setLost(false)
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
  const down = !connected && seen
  useEffect(() => {
    if (!down) {
      return undefined
    }
    const timer = window.setTimeout(() => setLost(true), lostMs)
    return () => window.clearTimeout(timer)
  }, [down])
  if (down) {
    return lost ? "lost" : "reconnecting"
  }
  return connected && back ? "back" : "idle"
}
