"use client"

import { useState } from "react"
import { Link } from "react-router"
import { BotIcon } from "lucide-react"
import { touchArea } from "@/lib/button-class"
import { cn } from "@/lib/utils"

import { FilterChips } from "@/components/list-controls/FilterChips"
import { IdChip } from "@/components/list-controls/IdChip"
import { useNarrow } from "@/hooks/use-narrow"
import { Pill, type Tone } from "@/components/shaping/session-ui"
import { Checkbox } from "@/components/ui/checkbox"
import { dayLabel } from "@/lib/clock"
import type { BoardCard, BoardColumn } from "@/lib/shape-flows"
import { countText, headingText } from "@/components/shaping/styles"

/** What a card says about where its session stands. */
function badge(card: BoardCard): { tone: Tone; label: string } | null {
  switch (card.status) {
    case "interviewing":
      return card.openQuestions > 0
        ? {
            tone: "acc",
            label: `${card.openQuestions} ${card.openQuestions === 1 ? "question" : "questions"} open`,
          }
        : null
    case "ready":
      return { tone: "hold", label: "Ready" }
    case "split":
      return { tone: "hold", label: "Split" }
    default:
      return { tone: "pass", label: "Sealed" }
  }
}

function detail(card: BoardCard): string {
  switch (card.status) {
    case "ready":
      return `${card.assumed} assumed · ${card.openQuestions} open`
    case "split":
      return `${card.stories} stories proposed`
    case "sealed":
      return [
        card.sealedAt === "" ? "sealed" : `sealed ${dayLabel(card.sealedAt)}`,
        card.jiraKey,
      ]
        .filter((part) => part !== "")
        .join(" · ")
    default:
      return card.doing
  }
}

function Card({
  card,
  workspace,
  chosen,
  onToggle,
}: {
  card: BoardCard
  workspace: string
  chosen: boolean
  onToggle: () => void
}) {
  const state = badge(card)
  const line = detail(card)
  return (
    <article
      className={cn(
        "grid grid-cols-[28px_minmax(0,1fr)] gap-2 rounded-[10px] border border-border bg-card p-3",
        chosen && "border-primary bg-accent"
      )}
    >
      <Checkbox
        aria-label={`Select ${card.ref}`}
        checked={chosen}
        onCheckedChange={onToggle}
        className="mt-0.5"
      />
      <div className="flex min-w-0 flex-col gap-1.5">
        <div className="flex flex-wrap items-center gap-1.5">
          <IdChip>{card.ref}</IdChip>
          {state === null ? null : (
            <Pill tone={state.tone} small>
              {state.label}
            </Pill>
          )}
          {chosen ? (
            <Pill tone="acc" small>
              Selected
            </Pill>
          ) : null}
        </div>
        <Link
          to={`/shaping/${card.id}?workspace=${encodeURIComponent(workspace)}`}
          className={cn(
            touchArea,
            "text-sm font-semibold text-pretty hover:underline focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none"
          )}
        >
          {card.title}
        </Link>
        <span className="flex items-center gap-1 font-mono text-xs text-muted-foreground">
          <BotIcon aria-hidden="true" className="size-3" />
          {card.agent === "" ? "auto" : card.agent} ·{" "}
          {card.model === "" ? "auto" : card.model}
        </span>
        {line === "" ? null : (
          <span className="text-xs text-pretty">{line}</span>
        )}
      </div>
    </article>
  )
}

function StatusColumn({
  column,
  workspace,
  chosen,
  onToggle,
}: {
  column: BoardColumn
  workspace: string
  chosen: Set<string>
  onToggle: (id: string) => void
}) {
  return (
    <section
      aria-label={column.name}
      className="flex min-w-0 flex-col gap-2"
    >
      <div className="flex items-center gap-2">
        <h2 className={headingText}>{column.name}</h2>
        <span className={countText}>{column.count}</span>
      </div>
      {column.cards.map((card) => (
        <Card
          key={card.id}
          card={card}
          workspace={workspace}
          chosen={chosen.has(card.id)}
          onToggle={() => onToggle(card.id)}
        />
      ))}
      {column.hidden > 0 ? (
        <Link
          to={`/shaping?view=sessions&status=${column.key}&workspace=${encodeURIComponent(workspace)}`}
          className="text-[13px] text-primary hover:underline"
        >
          See {column.hidden} more
        </Link>
      ) : null}
      {column.count === 0 ? (
        <p className="rounded-[10px] border border-dashed border-border p-3 text-xs text-muted-foreground">
          Nothing here.
        </p>
      ) : null}
    </section>
  )
}

/** The sessions in a column per status; a card is chosen with its checkbox. */
export function BoardColumns({
  columns,
  workspace,
  chosen,
  onToggle,
}: {
  columns: BoardColumn[]
  workspace: string
  chosen: Set<string>
  onToggle: (id: string) => void
}) {
  const narrow = useNarrow()
  const [status, setStatus] = useState(columns[0]?.key ?? "")
  const current =
    columns.find((column) => column.key === status) ?? columns[0]
  if (narrow && current !== undefined) {
    return (
      <div className="flex flex-col gap-3">
        <FilterChips
          label="Status"
          value={current.key}
          onChange={setStatus}
          chips={columns.map((column) => ({
            key: column.key,
            label: column.name,
            count: column.count,
          }))}
        />
        <StatusColumn
          column={current}
          workspace={workspace}
          chosen={chosen}
          onToggle={onToggle}
        />
      </div>
    )
  }
  return (
    <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
      {columns.map((column) => (
        <StatusColumn
          key={column.key}
          column={column}
          workspace={workspace}
          chosen={chosen}
          onToggle={onToggle}
        />
      ))}
    </div>
  )
}
