"use client"

import { useCallback, useEffect, useState } from "react"
import { Link, useParams, useSearchParams } from "react-router"
import { LockIcon } from "lucide-react"

import { ApiCli, type ApiAction } from "@/components/implementation/ApiCli"
import { FilterChips } from "@/components/list-controls/FilterChips"
import { Shell } from "@/components/shell/Shell"
import { SessionHeader } from "@/components/shaping/SessionHeader"
import { HistoryList } from "@/components/shaping/history/HistoryList"
import { UndoPanel } from "@/components/shaping/history/UndoPanel"
import { Panel, wideColumns } from "@/components/shaping/session-ui"
import { useNotedAction } from "@/components/shaping/use-noted-action"
import { getShape } from "@/lib/api"
import { reasonText } from "@/lib/guards"
import type { ShapeSession } from "@/lib/types"
import {
  getHistory,
  getUndoPlan,
  undoStep,
  type HistoryView,
  type UndoPlan,
} from "@/lib/shape-flows"
import { emptyPanelText, errorText } from "@/components/shaping/styles"

const exits =
  "0 ok · 1 failed · 2 usage · 3 not found · 4 refused, such as a sealed session or a step others depend on · 5 agent failed"

const actors = [
  { value: "", label: "Everyone" },
  { value: "you", label: "You" },
  { value: "agents", label: "Agents" },
]
const kinds = [
  { value: "", label: "All" },
  { value: "answers", label: "Answers" },
  { value: "assumptions", label: "Assumptions" },
  { value: "context", label: "Context" },
  { value: "split", label: "Split" },
]

/** How many steps each "who" filter keeps, keyed by the filter's value. */
function actorCounts(view: HistoryView): Record<string, number> {
  return {
    "": view.counts.everyone,
    you: view.counts.you,
    agents: view.counts.agents,
  }
}

/** The steps of one session with who did each, and the undo of a step before the story is sealed. */
export function ShapeHistory() {
  const { id = "" } = useParams()
  const [params] = useSearchParams()
  const workspace = params.get("workspace") ?? "."
  const [actor, setActor] = useState("")
  const [kind, setKind] = useState("")
  const [view, setView] = useState<HistoryView | null>(null)
  const [selected, setSelected] = useState("")
  const [plan, setPlan] = useState<UndoPlan | null>(null)
  const [session, setSession] = useState<ShapeSession | null>(null)
  const { note, setNote, busy, run } = useNotedAction()

  // The header is the session's own: the same chip, status, links and meta line as its page.
  useEffect(() => {
    let live = true
    void getShape(id, workspace)
      .then((loaded) => {
        if (live) {
          setSession(loaded)
        }
      })
      .catch((err: unknown) => {
        if (live) {
          setNote(reasonText(err, "the session could not be read"))
        }
      })
    return () => {
      live = false
    }
  }, [id, workspace, setNote])

  // load is the effect's dependency and the refresh the actions below call: a stable identity
  // keeps the effect from re-running on every render.
  const load = useCallback(() => {
    return getHistory(id, workspace, actor, kind)
      .then(setView)
      .catch((err: unknown) =>
        setNote(reasonText(err, "the history could not be read"))
      )
  }, [id, workspace, actor, kind, setNote])

  useEffect(() => {
    void load()
  }, [load])

  function pick(step: string) {
    setSelected(step)
    void run(async () => {
      setPlan(await getUndoPlan(id, workspace, step))
    }, "the step could not be read")
  }

  function undo(cascade: boolean) {
    void run(async () => {
      await undoStep(id, workspace, selected, cascade)
      setPlan(null)
      setSelected("")
      await load()
    }, "the step was not undone")
  }

  const ref = view?.ref ?? id
  const actions: ApiAction[] = [
    {
      name: "History",
      method: "GET",
      route: "/shapes/{id}/history?actor=you&kind=answers",
      command: `span shape history ${ref} --actor you --kind answers --json`,
    },
    {
      name: "What undoing a step changes",
      method: "GET",
      route: `/shapes/{id}/history/${selected || "h5"}`,
      command: `span shape undo ${ref} ${selected || "h5"} --plan`,
    },
    {
      name: "Undo a step",
      method: "POST",
      route: `/shapes/{id}/history/${selected || "h5"}/undo`,
      command: `span shape undo ${ref} ${selected || "h5"} --cascade --yes`,
    },
  ]

  return (
    <Shell path={`~/span/shaping/${id}/history`} status="history">
      {session === null ? null : <SessionHeader session={session} />}
      <nav
        aria-label="Breadcrumb"
        className="flex gap-2 text-[13px] text-muted-foreground"
      >
        <Link
          to={`/shaping?view=sessions&workspace=${encodeURIComponent(workspace)}`}
          className="hover:underline"
        >
          Sessions
        </Link>
        <span aria-hidden="true">/</span>
        <Link
          to={`/shaping/${id}?workspace=${encodeURIComponent(workspace)}`}
          className="hover:underline"
        >
          {ref}
        </Link>
        <span aria-hidden="true">/</span>
        <b aria-current="page" className="font-medium text-foreground">
          History
        </b>
      </nav>
      {view?.sealed === true ? (
        <p
          role="note"
          className="flex items-center gap-2 rounded-[14px] border border-border bg-card p-3 text-[13px]"
        >
          <LockIcon aria-hidden="true" className="size-4" />
          This story is sealed. Nothing here can be undone; start a new session
          from it to change it.
        </p>
      ) : null}
      <div className="flex flex-wrap gap-4">
        <FilterChips
          label="Filter by who"
          value={actor}
          chips={actors.map((option) => ({
            key: option.value,
            label: option.label,
            count: view === null ? undefined : actorCounts(view)[option.value],
          }))}
          onChange={setActor}
        />
        <FilterChips
          label="Filter by type"
          value={kind}
          chips={kinds.map((option) => ({
            key: option.value,
            label: option.label,
          }))}
          onChange={setKind}
        />
      </div>
      {note === "" ? null : (
        <p role="alert" className={errorText}>
          {note}
        </p>
      )}
      <div className={wideColumns}>
        {view === null ? (
          <p className="text-sm text-muted-foreground">Reading the history…</p>
        ) : (
          <HistoryList
            events={view.events}
            selected={selected}
            sealed={view.sealed}
            onSelect={pick}
          />
        )}
        <aside className="flex min-w-0 flex-col gap-4 lg:sticky lg:top-4 lg:self-start">
          {plan === null ? (
            <Panel className={emptyPanelText}>
              Pick Undo on a step to see what it would change before anything
              does.
            </Panel>
          ) : (
            <UndoPanel
              plan={plan}
              busy={busy}
              onUndo={undo}
              onCancel={() => {
                setPlan(null)
                setSelected("")
              }}
            />
          )}
        </aside>
      </div>
      <ApiCli actions={actions} exits={exits} />
    </Shell>
  )
}
