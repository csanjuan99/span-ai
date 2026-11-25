"use client"

import { useEffect, useState } from "react"
import { Link, useNavigate, useSearchParams } from "react-router"
import { GitMergeIcon } from "lucide-react"

import { ApiCli, type ApiAction } from "@/components/implementation/ApiCli"
import { Shell } from "@/components/shell/Shell"
import { ConflictCard } from "@/components/shaping/compare/ConflictCard"
import { DuplicateCard } from "@/components/shaping/compare/DuplicateCard"
import { SameList } from "@/components/shaping/compare/SameList"
import { SessionPicker } from "@/components/shaping/compare/SessionPicker"
import { ActionPanel } from "@/components/shaping/session-ui"
import { SummaryCells } from "@/components/shaping/versions/SummaryCells"
import { Button } from "@/components/ui/button"
import { useNotedAction } from "@/components/shaping/use-noted-action"
import { listShapes } from "@/lib/api"
import { reasonText } from "@/lib/guards"
import type { ShapeSession } from "@/lib/types"
import {
  compareShapes,
  keepSeparate,
  mergeCommand,
  mergeShapes,
  suggestedTarget,
  undecided,
  type Comparison,
} from "@/lib/shape-compare"
import { errorText } from "@/components/shaping/styles"

const exits =
  "0 ok · 1 failed · 2 usage · 3 not found · 4 refused, such as open conflicts or a sealed session · 5 agent failed"

/** Two to five sessions of one repo: what they share, where they disagree, and a merge. */
export function ShapeCompare() {
  const [params, setParams] = useSearchParams()
  const workspace = params.get("workspace") ?? "."
  const ids = (params.get("ids") ?? "").split(",").filter((id) => id !== "")
  const navigate = useNavigate()
  const [catalog, setCatalog] = useState<ShapeSession[]>([])
  const [loaded, setLoaded] = useState<Comparison | null>(null)
  const [picks, setPicks] = useState<Record<string, string>>({})
  const { note, setNote, busy, run } = useNotedAction()
  const key = ids.join(",")
  const few = ids.length < 2
  const result = few ? null : loaded

  useEffect(() => {
    let live = true
    void listShapes(workspace)
      .then((rows) => {
        if (live) {
          setCatalog(rows)
        }
      })
      .catch((err: unknown) => {
        if (live) {
          setNote(reasonText(err, "the sessions could not be read"))
        }
      })
    return () => {
      live = false
    }
  }, [workspace, setNote])

  useEffect(() => {
    if (few) {
      return
    }
    let live = true
    void compareShapes(workspace, key.split(","))
      .then((body) => live && setLoaded(body))
      .catch(
        (err: unknown) =>
          live && setNote(reasonText(err, "the sessions could not be compared"))
      )
    return () => {
      live = false
    }
  }, [workspace, key, few, setNote])

  const sessions = result?.sessions ?? []
  const target =
    result === null
      ? ""
      : suggestedTarget(
          sessions,
          ids.length > 0 ? sessions.map((s) => s.id) : []
        )
  const targetRef = sessions.find((s) => s.id === target)?.ref ?? target
  const from = sessions.filter((s) => s.id !== target && !s.sealed)
  const against = sessions.filter((s) => s.id !== target && s.sealed)
  const open = result === null ? [] : undecided(result.conflicts, picks)
  const command = mergeCommand(
    targetRef,
    from.map((s) => s.ref),
    against.map((s) => s.ref),
    Object.fromEntries(
      Object.entries(picks).map(([id, session]) => [
        id,
        sessions.find((s) => s.id === session)?.ref ?? session,
      ])
    )
  )
  const actions: ApiAction[] = [
    {
      name: "Compare",
      method: "GET",
      route: "/shapes/compare?ids=a,b,c",
      command: `span shape compare ${sessions.map((s) => s.ref).join(" ")}`,
    },
    {
      name: "Keep separate",
      method: "POST",
      route: "/shapes/compare/dismiss",
      command: `span shape compare ${sessions.map((s) => s.ref).join(" ")} --dismiss`,
    },
    { name: "Merge", method: "POST", route: "/shapes/merge", command },
  ]

  function choose(next: string[]) {
    const query = new URLSearchParams(params)
    if (next.length === 0) {
      query.delete("ids")
    } else {
      query.set("ids", next.join(","))
    }
    setParams(query, { replace: true })
  }

  function merge() {
    void run(async () => {
      await mergeShapes(workspace, {
        into: target,
        from: from.map((s) => s.id),
        against: against.map((s) => s.id),
        resolve: picks,
      })
      navigate(
        `/shaping/${encodeURIComponent(target)}?workspace=${encodeURIComponent(workspace)}`
      )
    }, "the sessions could not be merged")
  }

  function separate(a: string, b: string) {
    void run(
      async () => setLoaded(await keepSeparate(workspace, [a, b])),
      "could not keep them separate"
    )
  }

  return (
    <Shell path="~/span/shaping/compare" status="compare">
      <header className="flex flex-col gap-1">
        <h1 className="text-2xl leading-8 font-semibold tracking-[-0.01em]">
          Compare sessions
        </h1>
        <p className="text-[13px] leading-5 text-muted-foreground">
          Find overlaps and conflicts before they become two stories.
        </p>
      </header>
      <nav
        aria-label="Breadcrumb"
        className="flex items-center gap-2 text-[13px] text-muted-foreground"
      >
        <Link
          className="underline underline-offset-3"
          to={`/shaping?view=sessions&workspace=${encodeURIComponent(workspace)}`}
        >
          Sessions
        </Link>
        <span aria-hidden="true">/</span>
        <b aria-current="page" className="font-medium text-foreground">
          Compare
        </b>
      </nav>
      {note !== "" ? (
        <p role="alert" className={errorText}>
          {note}
        </p>
      ) : null}
      <SessionPicker sessions={catalog} chosen={ids} onChange={choose} />
      {result !== null ? (
        <>
          <SummaryCells
            label="What the sessions share"
            cells={[
              {
                label: "Sessions",
                value: String(sessions.length),
                caption: result.repo,
              },
              {
                label: "In common",
                value: String(result.same.length),
                caption: "rules and examples",
              },
              {
                label: "Conflicts",
                value: String(result.conflicts.length),
                caption: `${open.length} undecided`,
              },
              {
                label: "Unique",
                value: String(result.unique),
                caption: "only in one session",
              },
            ]}
          />
          <p className="text-xs text-muted-foreground">
            Matching is by shared words, so every figure here is an estimate.
          </p>
          <div className="grid items-start gap-4 sm:gap-6 lg:grid-cols-[minmax(0,1fr)_300px]">
            <div className="flex min-w-0 flex-col gap-4">
              {result.conflicts.map((conflict) => (
                <ConflictCard
                  key={conflict.id}
                  conflict={conflict}
                  picked={picks[conflict.id]}
                  onPick={(session) =>
                    setPicks((now) => ({ ...now, [conflict.id]: session }))
                  }
                />
              ))}
              {result.duplicates.map((dup) => (
                <DuplicateCard
                  key={`${dup.a}${dup.b}`}
                  duplicate={dup}
                  busy={busy}
                  onKeepSeparate={() => separate(dup.a, dup.b)}
                />
              ))}
              <SameList groups={result.same} />
            </div>
            <aside className="flex min-w-0 flex-col gap-4">
              <ActionPanel
                title={`Merge into ${targetRef || "…"}`}
                icon={GitMergeIcon}
                rows={[
                  {
                    label: "Moves",
                    body: `The unique items of ${from.map((s) => s.ref).join(", ") || "the other sessions"} go into ${targetRef || "the target"}.`,
                  },
                  {
                    label: "Kept",
                    body: "The other sessions stay readable and link to the target. Sealed sessions are only compared, never changed.",
                  },
                  {
                    label: "Needs",
                    body:
                      open.length === 0
                        ? "Every conflict is decided."
                        : `${open.length} conflict${open.length === 1 ? "" : "s"} still need a decision: ${open.join(", ")}.`,
                  },
                ]}
              >
                <Button
                  className="w-full"
                  size="lg"
                  disabled={
                    busy ||
                    target === "" ||
                    open.length > 0 ||
                    from.length === 0
                  }
                  onClick={merge}
                >
                  Merge into {targetRef}
                </Button>
              </ActionPanel>
            </aside>
          </div>
        </>
      ) : null}
      <ApiCli actions={actions} exits={exits} />
    </Shell>
  )
}
