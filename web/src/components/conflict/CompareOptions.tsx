"use client"

import { Check } from "lucide-react"
import { cn } from "cn"
import { useId, useRef, useState, type KeyboardEvent } from "react"

import { differingLines, riskLabel } from "@/lib/conflict"
import type { ConflictGroup } from "@/lib/types"
import { ConflictDiff } from "@/components/conflict/ConflictDiff"
import { optionLetter } from "@/lib/option-letter"
import { RiskBadge } from "@/components/conflict/ConflictOptionCard"
import { Button } from "@/components/ui/button"

/** Columns on a wide screen, by the number of options (a group has two to four). */
const columns: Record<number, string> = {
  2: "lg:grid-cols-2",
  3: "lg:grid-cols-3",
}

/**
 * Every option's diff next to the others, before anything is chosen. Wide screens show one
 * column per option; narrow ones show one at a time behind tabs. Lines the options do not
 * share are marked. The arrow keys move between options and Enter on a column's button
 * chooses it; each move and each choice is announced.
 */
export function CompareOptions({
  group,
  chosen,
  onChoose,
}: {
  group: ConflictGroup
  chosen: string
  onChoose: (option: string) => void
}) {
  const id = useId()
  const [active, setActive] = useState(0)
  const [said, setSaid] = useState("")
  const buttons = useRef<(HTMLButtonElement | null)[]>([])
  const differs = differingLines(group.options.map((option) => option.diff))

  function describe(at: number): string {
    const option = group.options[at]
    return option === undefined
      ? ""
      : `Option ${optionLetter(at)}, ${option.title}, ${riskLabel(option.risk)}. Press Enter to choose it.`
  }

  function move(at: number) {
    setActive(at)
    setSaid(describe(at))
    buttons.current[at]?.focus()
  }

  function onKeyDown(event: KeyboardEvent<HTMLDivElement>) {
    const last = group.options.length - 1
    if (event.key === "ArrowRight" || event.key === "ArrowDown") {
      event.preventDefault()
      move(active === last ? 0 : active + 1)
    } else if (event.key === "ArrowLeft" || event.key === "ArrowUp") {
      event.preventDefault()
      move(active === 0 ? last : active - 1)
    }
  }

  return (
    <div
      role="group"
      aria-label="Compare options"
      className="flex flex-col gap-3"
      onKeyDown={onKeyDown}
    >
      <div role="tablist" aria-label="Options" className="flex gap-1 lg:hidden">
        {group.options.map((option, at) => (
          <button
            key={option.id}
            type="button"
            role="tab"
            id={`${id}-tab-${at}`}
            aria-selected={active === at}
            className={cn(
              "rounded-md border px-3 py-1.5 text-sm",
              active === at
                ? "border-primary bg-accent font-semibold"
                : "border-input"
            )}
            onClick={() => move(at)}
          >
            {optionLetter(at)}
            {option.id === chosen ? " ✓" : ""}
          </button>
        ))}
      </div>
      <div
        className={cn(
          "grid gap-3",
          columns[group.options.length] ?? "lg:grid-cols-4"
        )}
      >
        {group.options.map((option, at) => (
          <section
            key={option.id}
            aria-label={`Option ${optionLetter(at)}`}
            className={cn(
              "min-w-0 flex-col gap-2 rounded-[10px] border p-3",
              active === at ? "flex" : "hidden lg:flex",
              option.id === chosen
                ? "border-primary bg-accent"
                : "border-input bg-card"
            )}
          >
            <div className="flex flex-wrap items-center gap-2">
              <span className="text-sm font-semibold">
                {optionLetter(at)} · {option.title}
              </span>
              <RiskBadge risk={option.risk} />
            </div>
            <Button
              type="button"
              size="sm"
              variant={option.id === chosen ? "default" : "outline"}
              aria-pressed={option.id === chosen}
              ref={(node) => {
                buttons.current[at] = node
              }}
              onFocus={() => setActive(at)}
              onClick={() => {
                onChoose(option.id)
                setSaid(`Chosen: option ${optionLetter(at)}, ${option.title}.`)
              }}
            >
              {option.id === chosen ? (
                <>
                  <Check aria-hidden="true" />
                  Chosen
                </>
              ) : (
                "Choose"
              )}
            </Button>
            <ConflictDiff diff={option.diff} differs={differs[at]} />
          </section>
        ))}
      </div>
      <p className="sr-only" role="status" aria-live="polite">
        {said}
      </p>
    </div>
  )
}
