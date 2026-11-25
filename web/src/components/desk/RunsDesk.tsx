"use client"

import { PlusIcon, SearchIcon } from "lucide-react"
import { useState } from "react"
import { Link } from "react-router"

import { DeleteRun } from "@/components/run-list/DeleteRun"
import { Button, buttonVariants } from "@/components/ui/button"
import {
  agoOf,
  deskCounts,
  groupRuns,
  pillOf,
  searchRuns,
  type DeskFilter,
  type GroupKey,
} from "@/lib/desk"
import type { SpanHead } from "@/lib/types"
import { cn } from "@/lib/utils"

const filters: { key: DeskFilter; label: string }[] = [
  { key: "all", label: "All" },
  { key: "needs", label: "Needs me" },
  { key: "running", label: "Running" },
  { key: "done", label: "Done" },
]

const tone: Record<
  GroupKey,
  { badge: string; pill: string; mark: string; action: string }
> = {
  waiting: {
    badge: "bg-hold-soft text-hold",
    pill: "text-hold",
    mark: "·",
    action: "Review",
  },
  failed: {
    badge: "bg-fail-soft text-fail",
    pill: "text-fail",
    mark: ">",
    action: "Open",
  },
  running: {
    badge: "bg-accent text-primary",
    pill: "text-primary",
    mark: ">",
    action: "Open",
  },
  done: {
    badge: "bg-pass-soft text-pass",
    pill: "text-pass",
    mark: "+",
    action: "Summary",
  },
}

function Row({
  run,
  group,
  now,
  onDeleted,
}: {
  run: SpanHead
  group: GroupKey
  now: Date
  onDeleted: (id: string) => void
}) {
  const style = tone[group]
  const name =
    run.title !== "" ? run.title : run.label !== "" ? run.label : run.id
  const ago = agoOf(run.updatedAt, now)
  return (
    <li className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-x-4 gap-y-1 border-t border-border px-4 py-3 first:border-t-0 md:grid-cols-[88px_minmax(0,1fr)_minmax(0,16rem)_96px_auto]">
      <span className="hidden self-start font-mono text-xs text-muted-foreground md:block">
        {run.issue}
      </span>
      <div className="flex min-w-0 flex-col">
        <Link
          to={`/spans/${run.id}`}
          className="truncate font-medium hover:underline"
          title={run.id}
        >
          {run.issue !== "" ? (
            <span className="mr-2 font-mono text-xs text-muted-foreground md:hidden">
              {run.issue}
            </span>
          ) : null}
          {name}
        </Link>
        {run.branch !== "" ? (
          <span className="truncate font-mono text-xs text-muted-foreground">
            {run.branch}
          </span>
        ) : null}
      </div>
      <span
        className={cn("col-span-2 truncate text-sm md:col-span-1", style.pill)}
      >
        <b aria-hidden="true" className="mr-1.5 font-mono">
          {style.mark}
        </b>
        {pillOf(run)}
      </span>
      <span className="text-right text-xs text-muted-foreground tabular-nums">
        {ago}
      </span>
      <span className="flex items-center justify-end gap-1">
        <Link
          to={`/spans/${run.id}`}
          className={buttonVariants({
            variant: group === "waiting" ? "default" : "outline",
            size: "sm",
          })}
        >
          {style.action}
        </Link>
        <DeleteRun id={run.id} label={name} onDeleted={onDeleted} />
      </span>
    </li>
  )
}

/** The runs as the desk shows them: grouped by what needs a person, with a search and filters. */
export function RunsDesk({
  runs,
  now,
  onNew,
  onDeleted,
}: {
  runs: SpanHead[]
  now: Date
  onNew: () => void
  onDeleted: (id: string) => void
}) {
  const [filter, setFilter] = useState<DeskFilter>("all")
  const [query, setQuery] = useState("")
  const counts = deskCounts(runs)
  const matches = searchRuns(runs, query)
  const groups = groupRuns(matches, filter)
  return (
    <div className="flex flex-col gap-5">
      <div className="flex flex-col gap-1">
        <h2 className="text-xl font-semibold">Desk</h2>
        <p className="text-sm text-muted-foreground">
          {counts.all} {counts.all === 1 ? "run" : "runs"} · {counts.needs}{" "}
          {counts.needs === 1 ? "needs" : "need"} you
        </p>
      </div>
      <div className="flex flex-wrap items-center gap-3">
        <label className="relative min-w-52 flex-1">
          <span className="sr-only">Search runs</span>
          <SearchIcon
            aria-hidden="true"
            className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground"
          />
          <input
            type="search"
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder="Search by key, title or branch"
            className="h-9 w-full rounded-lg border border-input bg-card pr-3 pl-9 text-sm focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none"
          />
        </label>
        <div
          role="group"
          aria-label="Filter runs"
          className="flex flex-wrap gap-1.5"
        >
          {filters.map((item) => (
            <button
              key={item.key}
              type="button"
              aria-pressed={filter === item.key}
              onClick={() => setFilter(item.key)}
              className="inline-flex h-9 items-center gap-1.5 rounded-lg border border-border px-3 text-sm text-muted-foreground aria-pressed:border-primary aria-pressed:bg-accent aria-pressed:text-primary"
            >
              {item.label}
              <span className="text-xs tabular-nums">{counts[item.key]}</span>
            </button>
          ))}
        </div>
        <Button type="button" onClick={onNew}>
          <PlusIcon aria-hidden="true" />
          New run
        </Button>
      </div>
      {groups.map((group) => (
        <section
          key={group.key}
          aria-label={group.title}
          className="flex flex-col gap-2"
        >
          <div className="flex items-center gap-2">
            <span
              className={cn(
                "rounded-md px-2 py-0.5 text-xs font-medium",
                tone[group.key].badge
              )}
            >
              {group.title}
            </span>
            <span className="text-xs text-muted-foreground tabular-nums">
              {group.runs.length}
            </span>
            {group.hint !== "" ? (
              <span className="text-xs text-muted-foreground">
                · {group.hint}
              </span>
            ) : null}
          </div>
          <ul className="overflow-hidden rounded-xl border border-border bg-card shadow-sm">
            {group.runs.map((run) => (
              <Row
                key={run.id}
                run={run}
                group={group.key}
                now={now}
                onDeleted={onDeleted}
              />
            ))}
          </ul>
        </section>
      ))}
      {groups.length === 0 ? (
        <section className="flex flex-col gap-2 rounded-xl border border-border bg-card p-4">
          <p className="font-medium">
            {query.trim() !== ""
              ? `No runs match “${query.trim()}”`
              : "No runs here"}
          </p>
          <p className="text-sm text-muted-foreground">
            Search looks at run keys, titles and branch names.
          </p>
          <div>
            <Button
              type="button"
              variant="outline"
              onClick={() => {
                setQuery("")
                setFilter("all")
              }}
            >
              Clear search
            </Button>
          </div>
        </section>
      ) : null}
    </div>
  )
}
