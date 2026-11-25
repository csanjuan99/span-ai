import { useCallback, useEffect, useRef, useState } from "react"

import {
  faviconWith,
  kindOf,
  playTone,
  loadPrefs,
  notifyEvents,
  prefsChanged,
  titleOf,
  toastCopy,
  wantsToast,
  type NotifyEvent,
  type NotifyKind,
} from "@/lib/notify"
import type { SpanHead } from "@/lib/types"

/** One toast on screen. */
export interface Toast {
  id: number
  kind: NotifyKind
  runId: string
  title: string
  detail: string
  action: string
}

/** What `useNotifications` gives the page: the toasts to show and how to close or hold one. */
export interface Notifications {
  toasts: Toast[]
  dismiss: (id: number) => void
  /** Keeps a toast on screen while the pointer is over it, and restarts its clock after. */
  hold: (id: number, held: boolean) => void
}

const toastMs = 8000

function tabInFront(): boolean {
  return document.visibilityState === "visible" && document.hasFocus()
}

function iconLink(): HTMLLinkElement | null {
  return document.head.querySelector<HTMLLinkElement>('link[rel~="icon"]')
}

function tellSystem(event: NotifyEvent): void {
  if (
    typeof Notification === "undefined" ||
    Notification.permission !== "granted"
  ) {
    return
  }
  const copy = toastCopy(event)
  new Notification(copy.title, { body: copy.detail, tag: event.run.id })
}

/**
 * Tells a person what the runs do: a toast when one needs them, fails or finishes, the tab title
 * and the page icon with how many wait, and the browser's own notification when they allowed it.
 * The first list it sees is not news. A null list means the desk has not loaded yet, so it
 * is not that first list and it raises nothing. It never asks the browser for permission.
 */
export function useNotifications(runs: SpanHead[] | null): Notifications {
  const [prefs, setPrefs] = useState(loadPrefs)
  const [toasts, setToasts] = useState<Toast[]>([])
  const [finished, setFinished] = useState<SpanHead | null>(null)
  const previous = useRef<SpanHead[] | null>(null)
  const next = useRef(1)
  const timers = useRef(new Map<number, number>())
  const title = useRef<string | null>(null)
  const icon = useRef<string | null>(null)

  // dismiss and expireIn keep one identity because the effect below lists expireIn: a new one
  // each render would rerun it and replay the title and icon work.
  const dismiss = useCallback((id: number) => {
    window.clearTimeout(timers.current.get(id))
    timers.current.delete(id)
    setToasts((all) => all.filter((toast) => toast.id !== id))
  }, [])

  const expireIn = useCallback(
    (id: number) => {
      window.clearTimeout(timers.current.get(id))
      timers.current.set(
        id,
        window.setTimeout(() => dismiss(id), toastMs)
      )
    },
    [dismiss]
  )

  const hold = (id: number, held: boolean) => {
    if (held) {
      window.clearTimeout(timers.current.get(id))
      timers.current.delete(id)
      return
    }
    setToasts((all) => {
      if (all.some((toast) => toast.id === id && toast.kind === "finished")) {
        expireIn(id)
      }
      return all
    })
  }

  useEffect(() => {
    const reload = () => setPrefs(loadPrefs())
    window.addEventListener(prefsChanged, reload)
    return () => window.removeEventListener(prefsChanged, reload)
  }, [])

  useEffect(() => {
    // The desk starts with no list. Treating that as an empty desk would announce every
    // run the moment the real list arrives.
    if (runs === null) {
      return
    }
    const events = notifyEvents(previous.current, runs)
    previous.current = runs
    if (events.length === 0) {
      return
    }
    const front = tabInFront()
    const shown: Toast[] = []
    for (const event of events) {
      if (wantsToast(prefs, event.kind, front)) {
        const copy = toastCopy(event)
        const id = next.current++
        shown.push({ id, kind: event.kind, runId: event.run.id, ...copy })
        if (event.kind === "finished") {
          expireIn(id)
        }
      }
      if (prefs.events[event.kind].system && !front) {
        tellSystem(event)
      }
      if (event.kind === "finished" && prefs.events.finished.tab) {
        setFinished(event.run)
      }
    }
    if (shown.length > 0) {
      setToasts((all) => [...all, ...shown])
      if (prefs.sound) {
        playTone()
      }
    }
  }, [runs, prefs, expireIn])

  useEffect(() => {
    const clear = () => {
      if (tabInFront()) {
        setFinished(null)
      }
    }
    window.addEventListener("focus", clear)
    document.addEventListener("visibilitychange", clear)
    return () => {
      window.removeEventListener("focus", clear)
      document.removeEventListener("visibilitychange", clear)
    }
  }, [])

  const marked = (runs ?? []).filter((run) => {
    const kind = kindOf(run)
    return (
      (kind === "needs" ||
        kind === "failed" ||
        kind === "blocked" ||
        kind === "stopped") &&
      prefs.events[kind].tab
    )
  })
  const heading = titleOf(marked, finished)
  const badge = heading === "Span" ? "" : heading.slice(1, heading.indexOf(")"))

  useEffect(() => {
    title.current ??= document.title
    document.title = badge === "" ? title.current : heading
  }, [badge, heading])

  useEffect(() => {
    const link = iconLink()
    if (link === null) {
      return
    }
    icon.current ??= link.getAttribute("href")
    const drawn = badge === "" ? "" : faviconWith(badge)
    const original = icon.current
    if (drawn !== "") {
      link.setAttribute("href", drawn)
    } else if (original !== null) {
      link.setAttribute("href", original)
    }
  }, [badge])

  useEffect(() => {
    const live = timers.current
    return () => {
      live.forEach((timer) => window.clearTimeout(timer))
      live.clear()
    }
  }, [])

  return { toasts, dismiss, hold }
}
