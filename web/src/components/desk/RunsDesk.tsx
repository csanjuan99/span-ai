"use client"

import { PlusIcon, SearchIcon } from "lucide-react"
import { useState } from "react"
import { Link } from "react-router"

import { useRowKeys } from "@/hooks/use-row-keys"

import { FilterChips } from "@/components/list-controls/FilterChips"
import { IdChip } from "@/components/list-controls/IdChip"
import { SearchField } from "@/components/list-controls/SearchField"
import { DeleteRun } from "@/components/run-list/DeleteRun"
import { Button, buttonVariants } from "@/components/ui/button"
import {
  actionOf,
  deskCounts,
  groupRuns,
  pillOf,
  searchRuns,
  whenOf,
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

const tone: Record<GroupKey, { colour: string; mark: string }> = {
  waiting: { colour: "bg-hold-soft text-hold", mark: "·" },
  failed: { colour: "bg-fail-soft text-fail", mark: ">" },
  running: { colour: "bg-accent text-primary", mark: ">" },
  done: { colour: "bg-pass-soft text-pass", mark: "+" },
}

const pillBase =
  "inline-flex h-6 max-w-full items-center gap-1.5 rounded-full px-2.5 text-xs leading-4 font-medium whitespace-nowrap"

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
  const action = actionOf(run)
  const name =
    run.title !== "" ? run.title : run.label !== "" ? run.label : run.id
  return (
    <li className="grid grid-cols-[1fr_auto] items-center gap-x-3 gap-y-2 border-t border-border p-4 first:border-t-0 md:grid-cols-[96px_minmax(0,1fr)_230px_104px_124px] md:gap-4 md:px-5 md:py-3">
      <span>{run.issue !== "" ? <IdChip>{run.issue}</IdChip> : null}</span>
      <div className="col-span-2 flex min-w-0 flex-col md:col-span-1">
        <Link
          to={`/spans/${run.id}`}
          data-row-link
          className="text-sm leading-[22px] font-semibold hover:underline max-md:flex max-md:min-h-11 max-md:items-center md:truncate"
          title={run.id}
        >
          {name}
        </Link>
        {run.branch !== "" ? (
          <span className="font-mono text-xs leading-[18px] break-all text-muted-foreground md:truncate">
            {run.branch}
          </span>
        ) : null}
      </div>
      <span className="col-span-2 min-w-0 md:col-span-1">
        <span className={cn(pillBase, style.colour)}>
          <b aria-hidden="true" className="font-mono text-xs font-semibold">
            {style.mark}
          </b>
          <span className="truncate">{pillOf(run)}</span>
        </span>
      </span>
      <span className="text-xs leading-4 text-muted-foreground tabular-nums md:text-right">
        {whenOf(run, now)}
      </span>
      <span className="col-span-2 flex items-center gap-1 md:col-span-1 md:justify-end">
        <Link
          to={action.to}
          className={cn(
            buttonVariants({
              variant: action.primary
                ? "default"
                : group === "running" || group === "done"
                  ? "ghost"
                  : "outline",
              size: "sm",
            }),
            "h-11 flex-1 rounded-[10px] px-3 text-[13px] md:h-8 md:flex-none"
          )}
        >
          {action.label}
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
  useRowKeys()
  const [filter, setFilter] = useState<DeskFilter>("all")
  const [query, setQuery] = useState("")
  const counts = deskCounts(runs, now)
  const matches = searchRuns(runs, query)
  const groups = groupRuns(matches, filter, now)
  return (
    <div className="flex flex-col gap-4 md:gap-6">
      <div className="flex flex-col gap-1">
        <h1 className="text-2xl leading-8 font-semibold tracking-tight">
          Desk
        </h1>
        <p className="text-[13px] leading-5 text-muted-foreground">
          {counts.all} {counts.all === 1 ? "run" : "runs"} · {counts.needs}{" "}
          {counts.needs === 1 ? "needs" : "need"} you
        </p>
      </div>
      <div className="flex flex-wrap items-center gap-3">
        <SearchField
          label="Search runs"
          placeholder="Search by key, title or branch"
          value={query}
          onChange={setQuery}
        />
        <FilterChips
          label="Filter runs"
          chips={filters.map((item) => ({ ...item, count: counts[item.key] }))}
          value={filter}
          onChange={setFilter}
        />
        <Button
          type="button"
          className="h-11 rounded-[10px] px-4 md:h-10"
          onClick={onNew}
        >
          <PlusIcon aria-hidden="true" />
          New run
        </Button>
      </div>
      {groups.map((group) => (
        <section
          key={group.key}
          aria-label={group.title}
          className="flex flex-col"
        >
          <div className="flex flex-wrap items-center gap-2 px-1 py-2">
            <span
              className={cn(
                "inline-flex h-6 items-center rounded-full px-2.5 text-xs leading-4 font-medium tracking-wide",
                tone[group.key].colour
              )}
            >
              {group.title}
            </span>
            <span className="text-xs leading-4 text-muted-foreground tabular-nums">
              {group.runs.length}
            </span>
            {group.hint !== "" ? (
              <span className="text-xs leading-4 text-muted-foreground">
                · {group.hint}
              </span>
            ) : null}
          </div>
          <ul className="overflow-hidden rounded-[14px] border border-border bg-card shadow-sm">
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
        <section className="flex flex-col gap-2 rounded-[14px] border border-border bg-card px-4 py-4 shadow-sm md:px-5">
          <div className="flex items-center gap-2">
            <SearchIcon
              aria-hidden="true"
              className="size-4 text-muted-foreground"
            />
            <p className="text-sm font-semibold">
              {query.trim() !== ""
                ? `No runs match “${query.trim()}”`
                : "No runs here"}
            </p>
          </div>
          <p className="text-[13px] leading-5 text-muted-foreground">
            Search looks at run keys, titles and branch names. Done runs older
            than 30 days are hidden until you filter by Done.
          </p>
          <div>
            <Button
              type="button"
              variant="outline"
              className="h-11 rounded-[10px] md:h-10"
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
