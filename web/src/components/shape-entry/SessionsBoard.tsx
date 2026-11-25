"use client"

import { PlusIcon, SearchIcon } from "lucide-react"
import { Link, useNavigate } from "react-router"

import { FilterChips } from "@/components/list-controls/FilterChips"
import { SearchField } from "@/components/list-controls/SearchField"
import { SelectField } from "@/components/settings-edit/SelectField"
import { TabStrip } from "@/components/shaping/TabStrip"
import { SessionTidy } from "@/components/shape-entry/SessionTidy"
import { SessionList, SessionRow } from "@/components/shape-entry/SessionRow"
import { Button } from "@/components/ui/button"
import {
  listedStates,
  needsYou,
  sessionTitle,
  stateCounts,
  stateLabels,
  type ShapeState,
} from "@/lib/shape-sessions"
import type { ShapeFilter, ShapeSession } from "@/lib/types"
import { noteText, titleText, wrapRowWide } from "@/components/shaping/styles"

/** What the sessions page filters by; the server applies it. */
export interface ListFilter {
  status: "all" | ShapeState
  query: string
  repo: string
  agent: string
  sort: "activity" | "created"
  archived: "" | "include" | "only"
}

/** No filter at all: every active session, newest activity first. */
export const noFilter: ListFilter = {
  status: "all",
  query: "",
  repo: "",
  agent: "",
  sort: "activity",
  archived: "",
}

/** What the filter sends to the server. */
export function filterOf(filter: ListFilter): ShapeFilter {
  return {
    status: filter.status === "all" ? "" : filter.status,
    query: filter.query.trim(),
    repo: filter.repo,
    agent: filter.agent,
    sort: filter.sort,
    archived: filter.archived,
  }
}

/** True when the filter keeps something back. */
export function isFiltered(filter: ListFilter): boolean {
  return JSON.stringify(filter) !== JSON.stringify(noFilter)
}

function unique(values: string[]): string[] {
  return [...new Set(values.filter((value) => value !== ""))].sort()
}

function emptyTitle(filter: ListFilter): string {
  const state =
    filter.status === "all"
      ? ""
      : `${stateLabels[filter.status].toLowerCase()} `
  const query = filter.query.trim()
  return query === ""
    ? `No ${state}session matches these filters`
    : `No ${state}session matches “${query}”`
}

/**
 * The sessions of a folder (SH12): search, a chip per state, repository, agent and sort,
 * and a row for each session. `all` is the unfiltered list the counts and options read.
 */
export function SessionsBoard({
  all,
  rows,
  filter,
  workspace,
  now,
  onFilter,
  onNew,
  onChanged,
}: {
  all: ShapeSession[]
  rows: ShapeSession[]
  filter: ListFilter
  workspace: string
  now: Date
  onFilter: (next: ListFilter) => void
  onNew: () => void
  onChanged: () => void
}) {
  const navigate = useNavigate()
  const counts = stateCounts(all)
  const needing = all.filter(needsYou).length
  const set = (patch: Partial<ListFilter>) => onFilter({ ...filter, ...patch })
  const here = `workspace=${encodeURIComponent(workspace)}`
  return (
    <div className="flex flex-col gap-4 md:gap-6">
      <div className="flex flex-col gap-1">
        <h1 className="text-2xl leading-8 font-semibold tracking-[-0.01em]">
          Sessions
        </h1>
        <p className={noteText}>
          {counts.all} {counts.all === 1 ? "session" : "sessions"} · {needing}{" "}
          {needing === 1 ? "needs" : "need"} you
        </p>
      </div>
      <div className={wrapRowWide}>
        <TabStrip
          label="View"
          value="list"
          options={[
            { value: "list", label: "List" },
            { value: "board", label: "Board" },
          ]}
          onChange={(next) => {
            if (next === "board") {
              void navigate(`/shaping/board?${here}`)
            }
          }}
          className="w-full max-w-[520px] sm:w-auto sm:min-w-64"
        />
        <Button
          asChild
          variant="outline"
          className="ml-auto h-11 rounded-[10px] px-4 md:h-10"
        >
          <Link to={`/shaping/compare?${here}`}>Compare sessions</Link>
        </Button>
      </div>
      <div className={wrapRowWide}>
        <SearchField
          label="Search sessions"
          placeholder="Search by idea, id or repo"
          value={filter.query}
          onChange={(query) => set({ query })}
        />
        <FilterChips
          label="Filter by status"
          value={filter.status}
          onChange={(status) => set({ status })}
          chips={[
            { key: "all" as const, label: "All", count: counts.all },
            ...listedStates.map((state) => ({
              key: state,
              label: stateLabels[state],
              count: counts[state],
            })),
          ]}
        />
      </div>
      <div className={wrapRowWide}>
        <SelectField
          mono
          label="Repository"
          value={filter.repo}
          options={[
            { value: "", label: "All repositories" },
            ...unique(all.map((row) => row.repo)).map((name) => ({
              value: name,
              label: name,
            })),
          ]}
          onChange={(repo) => set({ repo })}
        />
        <SelectField
          mono
          label="Agent"
          value={filter.agent}
          options={[
            { value: "", label: "Any agent" },
            { value: "auto", label: "Auto" },
            ...unique(all.map((row) => row.agent ?? "")).map((name) => ({
              value: name,
              label: name,
            })),
          ]}
          onChange={(agent) => set({ agent })}
        />
        <SelectField
          label="Sort"
          value={filter.sort}
          options={[
            { value: "activity", label: "Sort: last activity" },
            { value: "created", label: "Sort: created" },
          ]}
          onChange={(sort) =>
            set({ sort: sort === "created" ? "created" : "activity" })
          }
        />
        <SelectField
          label="Show"
          value={filter.archived}
          options={[
            { value: "", label: "Active sessions" },
            { value: "include", label: "Active and archived" },
            { value: "only", label: "Archived only" },
          ]}
          onChange={(archived) =>
            set({
              archived:
                archived === "include" || archived === "only" ? archived : "",
            })
          }
        />
        <span className="ml-auto">
          <Button
            type="button"
            className="h-11 rounded-[10px] px-4 md:h-10"
            onClick={onNew}
          >
            <PlusIcon aria-hidden="true" />
            New session
          </Button>
        </span>
      </div>
      {rows.length > 0 ? (
        <SessionList>
          {rows.map((session) => (
            <SessionRow
              key={session.id}
              session={session}
              workspace={workspace}
              now={now}
              tidy={
                <SessionTidy
                  session={session}
                  workspace={workspace}
                  title={sessionTitle(session)}
                  onChanged={onChanged}
                />
              }
            />
          ))}
        </SessionList>
      ) : (
        <section
          aria-label="No sessions match"
          className="flex flex-col gap-2 rounded-[14px] border border-border bg-card p-4 shadow-sm md:px-5"
        >
          <div className="flex items-center gap-2">
            <SearchIcon
              aria-hidden="true"
              className="size-4 text-muted-foreground"
            />
            <h2 className={titleText}>{emptyTitle(filter)}</h2>
          </div>
          <p className={noteText}>
            Search looks at ideas, ids and repo names. Try “All” to see sessions
            in other states.
          </p>
          <div>
            <Button
              type="button"
              variant="outline"
              className="h-11 rounded-[10px] px-4 md:h-10"
              onClick={() => onFilter(noFilter)}
            >
              Clear filters
            </Button>
          </div>
        </section>
      )}
    </div>
  )
}
