"use client"

import { useState, useSyncExternalStore } from "react"
import { useNavigate } from "react-router"

import { KeyCaps } from "@/components/help/KeyCaps"
import { ShortcutSheet } from "@/components/help/ShortcutSheet"
import { useOptionalTheme } from "@/components/theme-provider"
import { useCommandPalette } from "@/hooks/use-command-palette"
import {
  filterCommands,
  getRunContext,
  paletteCommands,
  runPaletteCommands,
  subscribeRunContext,
  type PaletteCommand,
} from "@/lib/help"
import { cn } from "@/lib/utils"

const commands = paletteCommands()

/**
 * Search or run a command, from anywhere, with Cmd/Ctrl+K. It also owns the shortcut sheet and the
 * keyboard of the whole page, so mounting it once gives the page both.
 */
export function CommandPalette() {
  const navigate = useNavigate()
  const theme = useOptionalTheme()
  const { paletteOpen, setPaletteOpen, sheetOpen, setSheetOpen } =
    useCommandPalette((path) => void navigate(path))
  const [query, setQuery] = useState("")
  const [active, setActive] = useState(0)

  function close() {
    setPaletteOpen(false)
    setQuery("")
    setActive(0)
  }

  const runContext = useSyncExternalStore(subscribeRunContext, getRunContext)
  const all = [...runPaletteCommands(runContext), ...commands]
  const shown = filterCommands(all, query).filter(
    (command) => command.action.type !== "theme" || theme !== undefined
  )

  function run(command: PaletteCommand) {
    close()
    const { action } = command
    if (action.type === "go") {
      void navigate(action.path)
    } else if (action.type === "theme") {
      theme?.setTheme(action.value)
    } else if (action.type === "run") {
      action.run()
    } else {
      setSheetOpen(true)
    }
  }

  return (
    <>
      <ShortcutSheet open={sheetOpen} onClose={() => setSheetOpen(false)} />
      {paletteOpen ? (
        <div
          className="fixed inset-0 z-50 flex items-start justify-center bg-background/80 p-4 pt-[12vh]"
          onClick={close}
        >
          <section
            role="dialog"
            aria-modal="true"
            aria-label="Command palette"
            className="flex max-h-[70vh] w-full max-w-xl flex-col overflow-hidden rounded-2xl border border-border bg-card shadow-md"
            onClick={(event) => event.stopPropagation()}
          >
            <div className="flex items-center gap-2 border-b border-border px-4">
              <input
                autoFocus
                role="combobox"
                aria-expanded="true"
                aria-controls="palette-list"
                aria-label="Search or run a command"
                placeholder="Search or run a command"
                value={query}
                onChange={(event) => {
                  setQuery(event.target.value)
                  setActive(0)
                }}
                onKeyDown={(event) => {
                  if (event.key === "ArrowDown" || event.key === "ArrowUp") {
                    event.preventDefault()
                    const step = event.key === "ArrowDown" ? 1 : -1
                    setActive((current) =>
                      shown.length === 0
                        ? 0
                        : (current + step + shown.length) % shown.length
                    )
                  } else if (
                    event.key === "Enter" &&
                    shown[active] !== undefined
                  ) {
                    event.preventDefault()
                    run(shown[active])
                  }
                }}
                className="h-12 min-w-0 flex-1 bg-transparent text-sm outline-none placeholder:text-muted-foreground"
              />
              <KeyCaps keys={["Esc"]} />
            </div>
            <ul
              id="palette-list"
              role="listbox"
              aria-label="Commands"
              className="flex flex-col overflow-auto p-2"
            >
              {shown.length === 0 ? (
                <li className="px-3 py-6 text-center text-sm text-muted-foreground">
                  No command matches “{query.trim()}”.
                </li>
              ) : null}
              {shown.map((command, index) => (
                <li key={command.id} role="presentation">
                  {index === 0 || shown[index - 1].group !== command.group ? (
                    <div className="px-3 pt-2 pb-1 text-xs tracking-wide text-muted-foreground uppercase">
                      {command.group}
                    </div>
                  ) : null}
                  <button
                    type="button"
                    role="option"
                    aria-selected={index === active}
                    onMouseEnter={() => setActive(index)}
                    onClick={() => run(command)}
                    className={cn(
                      "flex min-h-11 w-full items-center gap-3 rounded-lg px-3 py-2 text-left",
                      index === active && "bg-accent"
                    )}
                  >
                    <span className="flex min-w-0 flex-1 flex-col">
                      <span className="text-sm font-medium">
                        {command.title}
                      </span>
                      <span className="truncate text-xs text-muted-foreground">
                        {command.hint}
                      </span>
                    </span>
                    {command.keys !== undefined ? (
                      <KeyCaps keys={command.keys} />
                    ) : null}
                  </button>
                </li>
              ))}
            </ul>
            <div className="flex gap-4 border-t border-border px-4 py-2 text-xs text-muted-foreground">
              <span>↑ ↓ move</span>
              <span>↵ run</span>
              <span>esc close</span>
            </div>
          </section>
        </div>
      ) : null}
    </>
  )
}
