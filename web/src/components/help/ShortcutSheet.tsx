"use client"

import { KeyboardIcon } from "lucide-react"
import { useSyncExternalStore } from "react"

import { KeyCaps } from "@/components/help/KeyCaps"

import {
  getRunContext,
  runKeys,
  shortcuts,
  subscribeRunContext,
} from "@/lib/help"

/** Every key the page answers. Opens on ? and closes on Escape or a click outside. */
export function ShortcutSheet({
  open,
  onClose,
}: {
  open: boolean
  onClose: () => void
}) {
  const run = useSyncExternalStore(subscribeRunContext, getRunContext)
  if (!open) {
    return null
  }
  const groups =
    run === null
      ? shortcuts
      : [
          ...shortcuts,
          {
            group: `On this run · ${run.label}`,
            items: run.commands.map((command) => ({
              label: command.title,
              keys: runKeys(command),
            })),
          },
        ]
  return (
    <div
      className="fixed inset-0 z-50 flex items-start justify-center bg-foreground/40 p-4 pt-[12vh] dark:bg-black/65"
      onClick={onClose}
    >
      <section
        role="dialog"
        aria-modal="true"
        aria-label="Keyboard shortcuts"
        className="flex max-h-[76vh] w-full max-w-[640px] flex-col overflow-auto rounded-[20px] border border-border bg-card shadow-md"
        onClick={(event) => event.stopPropagation()}
      >
        <div className="flex items-center gap-2 border-b border-border px-5 py-4">
          <KeyboardIcon aria-hidden="true" className="size-4" />
          <h2 className="text-lg font-semibold">Keyboard shortcuts</h2>
          <button
            type="button"
            aria-label="Close"
            onClick={onClose}
            className="-my-3 ml-auto flex min-h-11 min-w-11 items-center justify-center"
          >
            <KeyCaps keys={["esc"]} />
          </button>
        </div>
        <div className="grid gap-x-8 gap-y-4 px-5 py-4 sm:grid-cols-2">
          {groups.map((group) => (
            <div key={group.group} className="flex flex-col">
              <h3 className="mb-1 text-xs font-medium tracking-[0.02em] text-muted-foreground">
                {group.group}
              </h3>
              <ul className="flex flex-col divide-y divide-border">
                {group.items.map((item) => (
                  <li
                    key={item.label}
                    className="flex items-center justify-between gap-3 py-2 text-[13px] leading-5"
                  >
                    {item.label}
                    <KeyCaps keys={item.keys} />
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>
        <p className="px-5 pb-4 text-xs text-muted-foreground">
          Shortcuts never run while you type in a field.
        </p>
      </section>
    </div>
  )
}
