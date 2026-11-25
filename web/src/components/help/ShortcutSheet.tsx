"use client"

import { KeyCaps } from "@/components/help/KeyCaps"
import { shortcuts } from "@/lib/help"

/** Every key the page answers. Opens on ? and closes on Escape or a click outside. */
export function ShortcutSheet({
  open,
  onClose,
}: {
  open: boolean
  onClose: () => void
}) {
  if (!open) {
    return null
  }
  return (
    <div
      className="fixed inset-0 z-50 flex items-start justify-center bg-background/80 p-4 pt-[12vh]"
      onClick={onClose}
    >
      <section
        role="dialog"
        aria-modal="true"
        aria-label="Keyboard shortcuts"
        className="flex max-h-[76vh] w-full max-w-lg flex-col gap-4 overflow-auto rounded-2xl border border-border bg-card p-5 shadow-md"
        onClick={(event) => event.stopPropagation()}
      >
        <div className="flex items-center gap-2">
          <h2 className="text-lg font-semibold">Keyboard shortcuts</h2>
          <button
            type="button"
            onClick={onClose}
            className="ml-auto rounded-md px-2 py-1 text-sm text-muted-foreground hover:bg-muted"
          >
            Close
          </button>
        </div>
        {shortcuts.map((group) => (
          <div key={group.group} className="flex flex-col gap-2">
            <h3 className="text-xs tracking-wide text-muted-foreground uppercase">
              {group.group}
            </h3>
            <ul className="flex flex-col divide-y divide-border">
              {group.items.map((item) => (
                <li
                  key={item.label}
                  className="flex items-center justify-between gap-3 py-2 text-sm"
                >
                  {item.label}
                  <KeyCaps keys={item.keys} />
                </li>
              ))}
            </ul>
          </div>
        ))}
        <p className="text-xs text-muted-foreground">
          Shortcuts never run while you type in a field.
        </p>
      </section>
    </div>
  )
}
