"use client"

import { SparklesIcon, XIcon } from "lucide-react"
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
  function dismiss() {
    remember(version)
    setHidden(true)
  }
  if (hidden || items.length === 0) {
    return null
  }
  return (
    <aside
      aria-label="What’s new"
      className="fixed right-4 bottom-4 z-40 flex w-[min(360px,calc(100vw-2rem))] flex-col gap-3 rounded-[14px] border border-border bg-card p-4 shadow-md"
    >
      <div className="flex items-center gap-2">
        <SparklesIcon aria-hidden="true" className="size-4 text-primary" />
        <h2 className="text-sm font-semibold">Span updated to {version}</h2>
        <span className="rounded-full bg-accent px-2 py-0.5 text-xs font-medium text-primary">
          New
        </span>
        <button
          type="button"
          aria-label="Close note"
          onClick={dismiss}
          className="-my-2 -mr-2 ml-auto grid size-11 place-items-center rounded-md hover:bg-muted"
        >
          <XIcon aria-hidden="true" className="size-4" />
        </button>
      </div>
      <ul className="flex flex-col gap-2 text-[13px] leading-5">
        {items.map((item) => (
          <li key={item} className="flex gap-3">
            <span aria-hidden="true" className="font-mono text-pass">
              +
            </span>
            <span className="text-pretty">{item}</span>
          </li>
        ))}
      </ul>
      <button
        type="button"
        onClick={dismiss}
        className="min-h-11 self-start rounded-[10px] px-3 text-sm font-medium hover:bg-muted sm:min-h-10"
      >
        Dismiss
      </button>
      <p className="text-xs text-muted-foreground">
        Shown once after an update, bottom right.
      </p>
    </aside>
  )
}
