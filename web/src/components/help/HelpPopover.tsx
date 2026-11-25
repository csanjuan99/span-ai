"use client"

import { CircleHelpIcon } from "lucide-react"
import { useId, useState } from "react"

/**
 * A small “What is this?” next to a term. It explains in a sentence what the term means and, as
 * short points, what is kept and what to watch for. It opens on press and closes with Got it or Escape.
 */
export function HelpPopover({
  title,
  body,
  points = [],
}: {
  title: string
  body: string
  points?: { mark: "+" | "!"; text: string }[]
}) {
  const [open, setOpen] = useState(false)
  const id = useId()
  return (
    <span className="relative inline-flex">
      <button
        type="button"
        aria-label={`What is “${title}”?`}
        aria-expanded={open}
        aria-controls={id}
        onClick={() => setOpen((current) => !current)}
        onKeyDown={(event) => {
          if (event.key === "Escape") {
            setOpen(false)
          }
        }}
        className="inline-grid size-6 place-items-center rounded-full text-muted-foreground hover:bg-muted hover:text-foreground focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none"
      >
        <CircleHelpIcon aria-hidden="true" className="size-4" />
      </button>
      {open ? (
        <div
          id={id}
          role="dialog"
          aria-label={`What is “${title}”?`}
          className="absolute top-full left-0 z-40 mt-2 flex w-72 flex-col gap-2 rounded-xl border border-border bg-card p-4 shadow-md"
        >
          <span className="text-sm font-semibold">What is “{title}”?</span>
          <p className="text-sm text-pretty">{body}</p>
          {points.length > 0 ? (
            <ul className="flex flex-col gap-1 text-sm">
              {points.map((point) => (
                <li key={point.text} className="flex gap-2">
                  <span
                    aria-hidden="true"
                    className={
                      point.mark === "+"
                        ? "font-mono text-pass"
                        : "font-mono text-hold"
                    }
                  >
                    {point.mark}
                  </span>
                  <span className="text-pretty">{point.text}</span>
                </li>
              ))}
            </ul>
          ) : null}
          <button
            type="button"
            onClick={() => setOpen(false)}
            className="self-start rounded-md border border-border px-2 py-1 text-sm hover:bg-muted"
          >
            Got it
          </button>
        </div>
      ) : null}
    </span>
  )
}
