"use client"

import { useEffect, useState } from "react"
import { useNavigate, useSearchParams } from "react-router"

import { FirstSession } from "@/components/shape-entry/FirstSession"
import { RecentSessions } from "@/components/shape-entry/RecentSessions"
import {
  noFilter,
  SessionsBoard,
  type ListFilter,
} from "@/components/shape-entry/SessionsBoard"
import {
  issueOf,
  StartForm,
  type StartInput,
} from "@/components/shape-entry/StartForm"
import { useShapeSessions } from "@/components/shape-entry/useShapeSessions"
import { Shell } from "@/components/shell/Shell"
import { useBusyAction } from "@/hooks/use-busy-action"
import { createShape, getJSON } from "@/lib/api"
import { reasonText } from "@/lib/guards"
import { repoNames } from "@/lib/repos"
import { noteText } from "@/components/shaping/styles"

/**
 * Shape a story, entry side: the start panel with the latest sessions (SH1), the first
 * session of a folder (SH9), and every session with search and filters (SH12). The folder
 * is `?workspace=`, "." (where Span runs) when absent; `?view=sessions` opens the full list.
 */
export function Shaping() {
  const navigate = useNavigate()
  const [params] = useSearchParams()
  const workspace = params.get("workspace") ?? "."
  const listing = params.get("view") === "sessions"
  const [filter, setFilter] = useState<ListFilter>(noFilter)
  const [repos, setRepos] = useState<string[]>([])
  const [firstIdea, setFirstIdea] = useState("")
  const [note, setNote] = useState("")
  const sessions = useShapeSessions(workspace, filter)
  // A started session leaves this page, so busy stays on once it works.
  const { busy, run: begin } = useBusyAction({
    keepBusyOnSuccess: true,
    onStart: () => setNote(""),
    onError: (err, fallback) => setNote(reasonText(err, fallback)),
  })

  useEffect(() => {
    let live = true
    void getJSON(`/repos?path=${encodeURIComponent(workspace)}`)
      .then((body) => {
        if (live) {
          setRepos(repoNames(body))
        }
      })
      .catch(() => {
        if (live) {
          setRepos([])
        }
      })
    return () => {
      live = false
    }
  }, [workspace])

  function start(input: StartInput) {
    void begin(async () => {
      const issue = issueOf(input.idea)
      const created = await createShape({
        workspace,
        ...(input.repo === "" ? {} : { repo: input.repo }),
        ...(issue === "" ? { idea: input.idea } : { issue, idea: "" }),
        ...(input.agent === "" ? {} : { agent: input.agent }),
        ...(input.model === "" ? {} : { model: input.model }),
      })
      void navigate(
        `/shaping/${created.id}?workspace=${encodeURIComponent(workspace)}`
      )
    }, "the session did not start")
  }

  const here = `/shaping?workspace=${encodeURIComponent(workspace)}`
  const now = new Date()
  const empty = sessions.loaded && sessions.all.length === 0
  const shownNote = note !== "" ? note : sessions.note

  let body
  if (listing && !empty) {
    body = (
      <SessionsBoard
        all={sessions.all}
        rows={sessions.rows}
        filter={filter}
        workspace={workspace}
        now={now}
        onFilter={setFilter}
        onNew={() => void navigate(here)}
        onChanged={sessions.reload}
      />
    )
  } else {
    body = (
      <>
        <div className="flex flex-col gap-1">
          <h1 className="text-2xl leading-8 font-semibold tracking-[-0.01em]">
            Shape a story
          </h1>
          <p className={noteText}>
            {empty
              ? "No sessions yet"
              : "Bring a rough idea. End with a story you can run or put in Jira."}
          </p>
        </div>
        {empty ? (
          <FirstSession
            idea={firstIdea}
            busy={busy}
            note={shownNote}
            onIdea={setFirstIdea}
            onStart={() =>
              start({ idea: firstIdea.trim(), repo: "", agent: "", model: "" })
            }
          />
        ) : (
          <>
            <StartForm
              workspace={workspace}
              repos={repos}
              busy={busy}
              note={shownNote}
              onStart={start}
            />
            {sessions.all.length > 0 ? (
              <RecentSessions
                sessions={sessions.all}
                total={sessions.all.length}
                workspace={workspace}
                now={now}
                allHref={`${here}&view=sessions`}
              />
            ) : null}
          </>
        )}
      </>
    )
  }

  return (
    <Shell path="~/span/shaping" status="shaping">
      {body}
    </Shell>
  )
}
