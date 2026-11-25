"use client"

import { useState } from "react"

const seenKey = "span-whats-new"

/** Whether this version's note was already dismissed. A blocked storage counts as not seen. */
function wasSeen(version: string): boolean {
  try {
    return localStorage.getItem(seenKey) === version
  } catch {
    return false
  }
}

/** Remembers the dismissal. Without storage the note simply shows again next time. */
function remember(version: string): boolean {
  try {
    localStorage.setItem(seenKey, version)
    return true
  } catch {
    return false
  }
}

/** A note shown once, bottom right, after an update. Dismissing it remembers the version. */
export function WhatsNew({
  version,
  items,
}: {
  version: string
  items: string[]
}) {
  const [hidden, setHidden] = useState(() => wasSeen(version))
  if (hidden || items.length === 0) {
    return null
  }
  return (
    <aside
      aria-label="What’s new"
      className="fixed right-4 bottom-4 z-40 flex w-80 flex-col gap-3 rounded-xl border border-border bg-card p-4 shadow-md"
    >
      <div className="flex items-center gap-2">
        <h2 className="text-sm font-semibold">Span updated to {version}</h2>
        <span className="rounded-full bg-pass-soft px-2 py-0.5 text-xs text-pass">
          New
        </span>
      </div>
      <ul className="flex flex-col gap-1 text-sm">
        {items.map((item) => (
          <li key={item} className="flex gap-2">
            <span aria-hidden="true" className="font-mono text-pass">
              +
            </span>
            <span className="text-pretty">{item}</span>
          </li>
        ))}
      </ul>
      <button
        type="button"
        onClick={() => {
          remember(version)
          setHidden(true)
        }}
        className="self-start rounded-md border border-border px-2 py-1 text-sm hover:bg-muted"
      >
        Dismiss
      </button>
    </aside>
  )
}
