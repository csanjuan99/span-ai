"use client"

import { useEffect, useRef, useState } from "react"
import { Link, useNavigate, useParams, useSearchParams } from "react-router"

import { ApiCli, type ApiAction } from "@/components/implementation/ApiCli"
import { Shell } from "@/components/shell/Shell"
import { SessionHeader } from "@/components/shaping/SessionHeader"
import { SessionSummary } from "@/components/shaping/SessionSummary"
import { Panel } from "@/components/shaping/session-ui"
import { ChangeCard } from "@/components/shaping/versions/ChangeCard"
import { ReadGuide } from "@/components/shaping/versions/ReadGuide"
import { RestorePanel } from "@/components/shaping/versions/RestorePanel"
import { RevisionTimeline } from "@/components/shaping/versions/RevisionTimeline"
import { StoryDiff } from "@/components/shaping/versions/StoryDiff"
import { SummaryCells } from "@/components/shaping/versions/SummaryCells"
import { ViewTabs } from "@/components/shaping/versions/ViewTabs"
import { useBusyAction } from "@/hooks/use-busy-action"
import { useNarrow } from "@/hooks/use-narrow"
import { getShape } from "@/lib/api"
import { reasonText } from "@/lib/guards"
import {
  defaultPair,
  forkRevision,
  getRevisionDiff,
  listRevisions,
  restoreRevision,
  tallyText,
  type Revision,
  type RevisionDiff,
} from "@/lib/shape-versions"
import type { ShapeSession } from "@/lib/types"
import { emptyPanelTextLarge, errorText } from "@/components/shaping/styles"
import { cn } from "@/lib/utils"
import { touchArea } from "@/lib/button-class"

const exits =
  "0 ok · 1 failed · 2 usage · 3 not found · 4 refused, such as a sealed session · 5 agent failed"

function actionsFor(ref: string, from: string, to: string): ApiAction[] {
  return [
    {
      name: "List revisions",
      method: "GET",
      route: "/shapes/{id}/revisions",
      command: `span shape log ${ref}`,
    },
    {
      name: "Compare two",
      method: "GET",
      route: `/shapes/{id}/diff?from=${from}&to=${to}`,
      command: `span shape diff ${ref} ${from} ${to} --format inline`,
    },
    {
      name: "Restore a revision",
      method: "POST",
      route: `/shapes/{id}/revisions/${from}/restore`,
      command: `span shape restore ${ref} ${from}`,
    },
    {
      name: "New session from a revision",
      method: "POST",
      route: `/shapes/{id}/revisions/${from}/fork`,
      command: `span shape restore ${ref} ${from} --new`,
    },
  ]
}

/** Two revisions of one session, side by side or inline, by kind of item, with a restore. */
export function ShapeVersions() {
  const { id = "" } = useParams()
  const [params, setParams] = useSearchParams()
  const workspace = params.get("workspace") ?? "."
  const navigate = useNavigate()
  const [session, setSession] = useState<ShapeSession | null>(null)
  const [revisions, setRevisions] = useState<Revision[] | null>(null)
  const [diff, setDiff] = useState<RevisionDiff | null>(null)
  const [back, setBack] = useState<RevisionDiff | null>(null)
  const [view, setView] = useState<"side" | "inline">("side")
  const narrow = useNarrow()
  // Side by side is the desktop comparison. A phone shows the inline diff only.
  const shown = narrow ? "inline" : view
  const [note, setNote] = useState("")
  const timeline = useRef<HTMLDivElement>(null)
  const { busy, run } = useBusyAction({
    onStart: () => setNote(""),
    onError: (err, fallback) => setNote(reasonText(err, fallback)),
  })

  const fallback =
    revisions === null ? { before: "", after: "" } : defaultPair(revisions)
  const before = params.get("from") ?? fallback.before
  const after = params.get("to") ?? fallback.after
  const last = revisions?.[revisions.length - 1]

  useEffect(() => {
    let live = true
    void Promise.all([getShape(id, workspace), listRevisions(id, workspace)])
      .then(([loaded, list]) => {
        if (live) {
          setSession(loaded)
          setRevisions(list)
        }
      })
      .catch((err: unknown) => {
        if (live) {
          setNote(reasonText(err, "the revisions could not be read"))
        }
      })
    return () => {
      live = false
    }
  }, [id, workspace])

  useEffect(() => {
    if (before === "" || after === "") {
      return
    }
    let live = true
    void getRevisionDiff(id, workspace, before, after)
      .then((loaded) => live && setDiff(loaded))
      .catch(
        (err: unknown) =>
          live &&
          setNote(reasonText(err, "the revisions could not be compared"))
      )
    return () => {
      live = false
    }
  }, [id, workspace, before, after])

  const lastId = last?.id ?? ""
  useEffect(() => {
    if (before === "" || lastId === "" || before === lastId) {
      return
    }
    let live = true
    void getRevisionDiff(id, workspace, before, lastId)
      .then((loaded) => live && setBack(loaded))
      .catch(() => live && setBack(null))
    return () => {
      live = false
    }
  }, [id, workspace, before, lastId])

  function pick(role: "before" | "after", rev: string) {
    const next = new URLSearchParams(params)
    next.set(role === "before" ? "from" : "to", rev)
    next.set("from", role === "before" ? rev : before)
    next.set("to", role === "after" ? rev : after)
    setParams(next, { replace: true })
  }

  function restore(rev: string) {
    void run(async () => {
      const next = await restoreRevision(id, workspace, rev)
      const list = await listRevisions(id, workspace)
      setSession(next)
      setRevisions(list)
      setBack(null)
      const newest = list[list.length - 1]
      const reset = new URLSearchParams(params)
      reset.set("from", rev)
      reset.set("to", newest?.id ?? rev)
      setParams(reset, { replace: true })
    }, "the revision could not be restored")
  }

  function fork(rev: string) {
    void run(async () => {
      const child = await forkRevision(id, workspace, rev)
      navigate(
        `/shaping/${encodeURIComponent(child.id)}?workspace=${encodeURIComponent(child.workspace)}`
      )
    }, "the new session could not be started")
  }

  const target = revisions?.find((rev) => rev.id === before)
  const ref = session?.ref ?? id
  const sealed = session?.status === "sealed"
  const one = revisions !== null && revisions.length < 2

  return (
    <Shell path={`~/span/shaping/${id}/versions`} status="versions">
      {session !== null ? <SessionHeader session={session} /> : null}
      <nav
        aria-label="Breadcrumb"
        className="flex flex-wrap items-center gap-2 text-[13px] leading-5 text-muted-foreground"
      >
        <Link
          className={cn(touchArea, "underline underline-offset-3")}
          to="/shaping"
        >
          Sessions
        </Link>
        <span aria-hidden="true">/</span>
        <Link
          className={cn(touchArea, "underline underline-offset-3")}
          to={`/shaping/${encodeURIComponent(id)}?workspace=${encodeURIComponent(workspace)}`}
        >
          {ref}
        </Link>
        <span aria-hidden="true">/</span>
        <b aria-current="page" className="font-medium text-foreground">
          Compare versions
        </b>
      </nav>
      {note !== "" ? (
        <p role="alert" className={errorText}>
          {note}
        </p>
      ) : null}
      {sealed ? (
        <Panel className="flex items-center gap-3 bg-pass-soft px-4 py-2.5 text-[13px] text-pass">
          <span>
            <b>{ref} is sealed.</b> You can compare its revisions. You cannot
            restore one. A sealed story cannot change.
          </span>
        </Panel>
      ) : null}
      {one ? (
        <Panel className={emptyPanelTextLarge}>
          This session has one revision so far. Answer a question or accept a
          note and there will be two to compare.
        </Panel>
      ) : null}
      {revisions !== null && !one ? (
        <div ref={timeline}>
          <RevisionTimeline
            revisions={revisions}
            before={before}
            after={after}
            onPick={pick}
          />
        </div>
      ) : null}
      {diff !== null && !one ? (
        <>
          <SummaryCells
            label="What changed"
            cells={[
              {
                label: "Rules",
                value: tallyText(diff.rules),
                caption: `${diff.from.id} to ${diff.to.id}`,
              },
              {
                label: "Examples",
                value: tallyText(diff.examples),
                caption: `${diff.from.id} to ${diff.to.id}`,
              },
              {
                label: "Questions",
                value: tallyText(diff.questions),
                caption:
                  diff.items
                    .filter((item) => item.kind === "question")
                    .map((item) => item.id)
                    .join(", ") || "none",
              },
              {
                label: "Story text",
                value: `+${diff.storyAdded} −${diff.storyRemoved} lines`,
                caption: `${diff.from.id} to ${diff.to.id}`,
              },
            ]}
          />
          {narrow ? null : (
            <ViewTabs
              label="Compare view"
              value={view}
              onChange={setView}
              options={[
                { value: "side", label: "Side by side" },
                { value: "inline", label: "Inline" },
              ]}
            />
          )}
          <div className="grid items-start gap-4 sm:gap-6 lg:grid-cols-[minmax(0,1fr)_300px]">
            <div className="flex min-w-0 flex-col gap-4">
              {diff.items.length === 0 && diff.story.length === 0 ? (
                <Panel className={emptyPanelTextLarge}>
                  {diff.from.id === diff.to.id
                    ? "Pick two different revisions to see what changed."
                    : "Nothing differs between these two revisions."}
                </Panel>
              ) : null}
              {diff.items.map((item) => (
                <ChangeCard
                  key={`${item.kind}${item.id}`}
                  item={item}
                  from={diff.from.id}
                  to={diff.to.id}
                  view={shown}
                />
              ))}
              {diff.items.length > 0 || diff.story.length > 0 ? (
                <StoryDiff diff={diff} view={shown} />
              ) : null}
            </div>
            <aside className="flex min-w-0 flex-col gap-4">
              {target !== undefined && last !== undefined ? (
                <RestorePanel
                  target={target}
                  last={last}
                  back={back}
                  sealed={sealed}
                  busy={busy}
                  onRestore={() => restore(target.id)}
                  onFork={() => fork(target.id)}
                  onPickOther={() =>
                    timeline.current?.scrollIntoView({
                      behavior: "smooth",
                      block: "start",
                    })
                  }
                />
              ) : null}
              <ReadGuide />
              {session !== null ? <SessionSummary session={session} /> : null}
            </aside>
          </div>
        </>
      ) : null}
      <ApiCli
        actions={actionsFor(ref, before || "r1", after || "r2")}
        exits={exits}
      />
    </Shell>
  )
}
