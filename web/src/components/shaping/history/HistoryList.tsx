"use client"

import { cn } from "@/lib/utils"

import { IdChip } from "@/components/list-controls/IdChip"
import { Panel } from "@/components/shaping/session-ui"
import { Button } from "@/components/ui/button"
import { clockTime } from "@/lib/clock"
import { type HistoryEvent } from "@/lib/shape-flows"
import { monoCaption } from "@/components/shaping/styles"
import { touchButtonXs } from "@/lib/button-class"

function who(actor: string): string {
  return actor === "you" ? "you" : actor
}

/** The steps of a session, newest first, each with who did it and a way to undo it. */
export function HistoryList({
  events,
  selected,
  sealed,
  onSelect,
}: {
  events: HistoryEvent[]
  selected: string
  sealed: boolean
  onSelect: (id: string) => void
}) {
  return (
    <Panel aria-label="History" className="overflow-hidden">
      <div className="flex flex-wrap items-center gap-2 p-4">
        <h2 className="text-lg font-semibold">History</h2>
        <span className="text-xs text-muted-foreground">
          Newest first. Every step shows who did it.
        </span>
      </div>
      <ol>
        {events.map((event) => (
          <li
            key={event.id}
            className={cn(
              "grid grid-cols-[48px_minmax(0,1fr)_auto] items-center gap-3 border-t border-border px-4 py-3 sm:grid-cols-[48px_96px_minmax(0,1fr)_auto]",
              event.id === selected && "bg-accent"
            )}
          >
            <span className={monoCaption}>{clockTime(event.at)}</span>
            <span
              className={cn(
                "hidden truncate text-[13px] sm:block",
                event.actor !== "you" && "font-mono"
              )}
            >
              {who(event.actor)}
            </span>
            <div className="min-w-0">
              <div className="text-[13px] font-medium text-pretty">
                {event.summary}
                {event.undoneBy === "" ? null : (
                  <span className="font-normal text-muted-foreground">
                    {" "}
                    · undone by {event.undoneBy}
                  </span>
                )}
              </div>
              <div className="text-xs text-muted-foreground">
                <IdChip>{event.id}</IdChip>{" "}
                <span className="sm:hidden">{who(event.actor)} · </span>
                {event.ref}
                {event.id === selected ? " · selected" : ""}
              </div>
            </div>
            {event.undoable && !sealed ? (
              <Button
                type="button"
                size="sm"
                variant="outline"
                className={touchButtonXs}
                aria-label={`Undo ${event.id}`}
                onClick={() => onSelect(event.id)}
              >
                Undo
              </Button>
            ) : (
              <span className="max-w-32 text-right text-xs text-muted-foreground">
                {event.undoneBy !== ""
                  ? "Undone"
                  : event.why === ""
                    ? ""
                    : event.why}
              </span>
            )}
          </li>
        ))}
      </ol>
      <p className="border-t border-border p-4 text-xs text-muted-foreground">
        Sealing comes last. Steps before it can be undone. Nothing after it can.
      </p>
    </Panel>
  )
}
