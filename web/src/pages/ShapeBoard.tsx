"use client"

import { useCallback, useEffect, useState } from "react"
import { Link, useNavigate, useSearchParams } from "react-router"

import { ApiCli, type ApiAction } from "@/components/implementation/ApiCli"
import { SelectField } from "@/components/settings-edit/SelectField"
import { Shell } from "@/components/shell/Shell"
import { ActivityFeed } from "@/components/shaping/board/ActivityFeed"
import { BoardColumns } from "@/components/shaping/board/BoardColumns"
import { BulkBar } from "@/components/shaping/board/BulkBar"
import { ViewTabs } from "@/components/shaping/versions/ViewTabs"
import { Button } from "@/components/ui/button"
import { useNotedAction } from "@/components/shaping/use-noted-action"
import { reasonText } from "@/lib/guards"
import {
  bulk,
  getBoard,
  type BoardView,
  type BulkAction,
  type BulkResult,
} from "@/lib/shape-flows"
import { errorText, hintText, wrapRowWide } from "@/components/shaping/styles"

const exits =
  "0 ok · 1 failed · 2 usage · 3 not found · 4 refused, such as a sealed session · 5 agent failed"

const actions: ApiAction[] = [
  {
    name: "Sessions by status",
    method: "GET",
    route: "/shapes?group=status&repo=acme/billing-api",
    command: "span shape board --repo acme/billing-api --json",
  },
  {
    name: "What the agents did",
    method: "GET",
    route: "/shapes/activity?who=agents&follow=1",
    command: "span shape activity --who agents --follow",
  },
  {
    name: "One action on several sessions",
    method: "POST",
    route: "/shapes/bulk",
    command: "span shape bulk assume SES-0142 SES-0124 --yes",
  },
]

/** Sessions board: the sessions in a column per status, a bulk action on a selection, and what the agents did. */
export function ShapeBoard() {
  const [params, setParams] = useSearchParams()
  const navigate = useNavigate()
  const workspace = params.get("workspace") ?? "."
  const repo = params.get("repo") ?? ""
  const [board, setBoard] = useState<BoardView | null>(null)
  const [chosen, setChosen] = useState<Set<string>>(new Set())
  const [plan, setPlan] = useState<BulkResult | null>(null)
  const [action, setAction] = useState<BulkAction | null>(null)
  const { note, setNote, busy, run } = useNotedAction()

  // load is the effect's dependency and the refresh the actions below call: a stable identity
  // keeps the effect from re-running on every render.
  const load = useCallback(() => {
    return getBoard(workspace, repo, 5)
      .then(setBoard)
      .catch((err: unknown) =>
        setNote(reasonText(err, "the board could not be read"))
      )
  }, [workspace, repo, setNote])

  useEffect(() => {
    void load()
  }, [load])

  const cards = (board?.columns ?? []).flatMap((column) => column.cards)
  const picked = cards.filter((card) => chosen.has(card.id))

  function toggle(id: string) {
    setChosen((now) => {
      const next = new Set(now)
      if (!next.delete(id)) {
        next.add(id)
      }
      return next
    })
    setPlan(null)
    setAction(null)
  }

  function ask(next: BulkAction) {
    void run(async () => {
      setPlan(await bulk(workspace, next, [...chosen], true))
      setAction(next)
    }, "the plan could not be read")
  }

  function apply() {
    if (action === null) {
      return
    }
    void run(async () => {
      const done = await bulk(workspace, action, [...chosen], false)
      setPlan(null)
      setAction(null)
      setChosen(new Set())
      if (done.failed > 0) {
        setNote(
          `${done.failed} of ${done.items.length} sessions could not change.`
        )
      }
      await load()
    }, "the action did not run")
  }

  function pickRepo(next: string) {
    const query = new URLSearchParams(params)
    if (next === "") {
      query.delete("repo")
    } else {
      query.set("repo", next)
    }
    setParams(query)
  }

  return (
    <Shell path="~/span/shaping/board" status="board">
      <header className="flex flex-col gap-1">
        <h1 className="text-2xl font-semibold tracking-tight">
          Sessions board
        </h1>
        <p className={hintText}>
          {repo === "" ? "All repositories" : repo} · {board?.inFlight ?? 0} in
          flight · {board?.needYou ?? 0} need you
        </p>
      </header>
      <div className={wrapRowWide}>
        <ViewTabs
          label="View"
          value="board"
          options={[
            { value: "list", label: "List" },
            { value: "board", label: "Board" },
          ]}
          onChange={(next) => {
            if (next === "list") {
              void navigate(
                `/shaping?view=sessions&workspace=${encodeURIComponent(workspace)}`
              )
            }
          }}
        />
        <Link
          to={`/shaping?view=sessions&workspace=${encodeURIComponent(workspace)}`}
          className="inline-flex min-h-11 items-center text-[13px] text-primary hover:underline md:min-h-0"
        >
          Open the list
        </Link>
        {(board?.repos.length ?? 0) > 1 ? (
          <SelectField
            mono
            label="Repository"
            value={repo}
            onChange={pickRepo}
            options={[
              { value: "", label: "All repositories" },
              ...(board?.repos ?? []).map((name) => ({
                value: name,
                label: name,
              })),
            ]}
          />
        ) : null}
        <Button asChild variant="outline" className="ml-auto">
          <Link to={`/shaping?workspace=${encodeURIComponent(workspace)}`}>
            New session
          </Link>
        </Button>
      </div>
      {note === "" ? null : (
        <p role="alert" className={errorText}>
          {note}
        </p>
      )}
      {board === null ? (
        <p className="text-sm text-muted-foreground">Reading the sessions…</p>
      ) : (
        <BoardColumns
          columns={board.columns}
          workspace={workspace}
          chosen={chosen}
          onToggle={toggle}
        />
      )}
      <BulkBar
        chosen={picked}
        plan={plan}
        action={action}
        busy={busy}
        onPlan={ask}
        onApply={apply}
        onCancel={() => {
          setPlan(null)
          setAction(null)
        }}
        onClear={() => {
          setChosen(new Set())
          setPlan(null)
          setAction(null)
        }}
      />
      <ActivityFeed workspace={workspace} />
      <ApiCli actions={actions} exits={exits} />
    </Shell>
  )
}
