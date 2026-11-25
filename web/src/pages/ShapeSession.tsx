"use client"

import { useEffect, useState } from "react"
import { useNavigate, useParams, useSearchParams } from "react-router"

import { useBusyAction } from "@/hooks/use-busy-action"
import { reasonText } from "@/lib/guards"
import {
  answerShape,
  decideShapeSplit,
  forkShape,
  getShape,
  resumeShape,
  sealShape,
  settleShapeContext,
  startRun,
} from "@/lib/api"
import { ContextBoard } from "@/components/shaping/ContextBoard"
import { JiraBoard } from "@/components/shaping/JiraBoard"
import { MapBoard } from "@/components/shaping/MapBoard"
import { QuestionsBoard } from "@/components/shaping/QuestionsBoard"
import { SealBoard } from "@/components/shaping/SealBoard"
import { SealedBoard } from "@/components/shaping/SealedBoard"
import { SessionFailure } from "@/components/shaping/SessionFailure"
import { SessionHeader } from "@/components/shaping/SessionHeader"
import { SessionRail } from "@/components/shaping/SessionRail"
import { SplitBoard } from "@/components/shaping/SplitBoard"
import { Working } from "@/components/shaping/Working"
import { Shell } from "@/components/shell/Shell"
import {
  contextCounts,
  sessionStage,
  sessionSteps,
  type SessionProgress,
} from "@/lib/shape-session"
import type { ShapeSession as Session } from "@/lib/types"
import { errorText } from "@/components/shaping/styles"

const readFailure = "the session could not be read"

function runForm(session: Session): FormData {
  const form = new FormData()
  form.set("workspace", session.workspace)
  form.set("story", session.story)
  if (session.jira !== null) {
    form.set("issue", session.jira.key)
  }
  if (session.repo !== "") {
    form.set("repo", session.repo)
  }
  return form
}

/** One shaping session: the interview, the map, the split, the context, the seal and what to do with the sealed story. */
export function ShapeSession() {
  const { id = "" } = useParams()
  const [params] = useSearchParams()
  const workspace = params.get("workspace") ?? "."
  const navigate = useNavigate()
  const [session, setSession] = useState<Session | null>(null)
  const [note, setNote] = useState("")
  // What the person did on this page: the server only knows the session's status.
  const [seen, setSeen] = useState<SessionProgress & { id: string }>({ id })
  const [revisit, setRevisit] = useState(false)
  const [running, setRunning] = useState(false)

  const progress: SessionProgress = seen.id === id ? seen : {}
  function advance(change: SessionProgress) {
    setSeen({ ...progress, ...change, id })
  }

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
          setNote(reasonText(err, readFailure))
        }
      })
    return () => {
      live = false
    }
  }, [id, workspace])

  // A turn runs on the server and can take a while, so one request at a time.
  const { busy, run: act } = useBusyAction({
    onStart: () => setNote(""),
    onError: (err, fallback) => setNote(reasonText(err, fallback)),
  })

  function change(task: () => Promise<Session>) {
    void act(async () => setSession(await task()))
  }

  // A turn also marks the page as working, so the screen says so until the agent answers.
  function turn(task: () => Promise<Session>) {
    setRunning(true)
    void act(async () => {
      try {
        setSession(await task())
        setRevisit(false)
      } finally {
        setRunning(false)
      }
    })
  }

  async function reload() {
    try {
      setSession(await getShape(id, workspace))
    } catch (err: unknown) {
      setNote(reasonText(err, readFailure))
    }
  }

  function beginRun(current: Session) {
    void act(async () => {
      const result = await startRun(runForm(current))
      if (typeof result === "string") {
        setNote(result)
        return
      }
      void navigate(`/spans/${result.id}`)
    }, "the run did not start")
  }

  function fork(current: Session) {
    void act(async () => {
      const next = await forkShape(current.id, current.workspace)
      void navigate(
        `/shaping/${encodeURIComponent(next.id)}?workspace=${encodeURIComponent(next.workspace)}`
      )
    }, "the new session did not start")
  }

  function settleContext(current: Session, accept: string[], skip: string[]) {
    void act(async () => {
      if (accept.length + skip.length > 0) {
        setSession(
          await settleShapeContext(current.id, current.workspace, {
            accept,
            skip,
          })
        )
      }
      advance({ contextDone: true })
    }, "the notes were not saved")
  }

  function resume(current: Session) {
    turn(() => resumeShape(current.id, current.workspace))
  }

  const stage =
    session === null
      ? "interview"
      : revisit && session.status !== "sealed"
        ? "questions"
        : sessionStage(session, progress)

  function board(current: Session) {
    if (running || stage === "interview") {
      return (
        <Working
          session={current}
          running={running}
          busy={busy}
          onSession={setSession}
          onDraft={() => resume(current)}
        />
      )
    }
    switch (stage) {
      case "questions":
        return (
          <QuestionsBoard
            key={current.questions
              .map((question) => `${question.id}${question.status}`)
              .join(",")}
            session={current}
            busy={busy}
            onSubmit={(answers, assume) => {
              turn(() => answerShape(current.id, workspace, answers, assume))
              setSeen({ id })
            }}
          />
        )
      case "map":
        return (
          <MapBoard
            session={current}
            next={
              current.status === "split"
                ? "split"
                : contextCounts(current).proposed > 0
                  ? "context"
                  : "seal"
            }
            onContinue={() => advance({ reviewed: true })}
          />
        )
      case "split":
        return (
          <SplitBoard
            session={current}
            busy={busy}
            onDecide={(decision) => {
              if (decision === "keep") {
                turn(() => decideShapeSplit(current.id, workspace, "keep"))
                return
              }
              change(() => decideShapeSplit(current.id, workspace, "split"))
            }}
          />
        )
      case "context":
        return (
          <ContextBoard
            session={current}
            busy={busy}
            onContinue={(accept, skip) => settleContext(current, accept, skip)}
          />
        )
      case "seal":
        return (
          <SealBoard
            session={current}
            busy={busy}
            canGoBack={contextCounts(current).proposed > 0}
            onSeal={() => change(() => sealShape(current.id, workspace))}
            onBack={() => advance({ contextDone: false })}
            onRevisit={() => setRevisit(true)}
          />
        )
      case "jira":
        return (
          <JiraBoard
            session={current}
            busyOutside={busy}
            onSession={setSession}
            onReload={reload}
            onBack={() => advance({ jira: false })}
            onRun={() => beginRun(current)}
          />
        )
      default:
        return (
          <SealedBoard
            session={current}
            busy={busy}
            onRun={() => beginRun(current)}
            onPreview={() => advance({ jira: true })}
            onFork={() => fork(current)}
          />
        )
    }
  }

  return (
    <Shell path={`~/span/shaping/${id}`} status={session?.status ?? "shaping"}>
      {session !== null ? (
        <>
          <SessionHeader session={session} />
          <SessionRail steps={sessionSteps(session, stage)} />
        </>
      ) : null}
      {note !== "" ? (
        <p role="alert" className={errorText}>
          {note}
        </p>
      ) : null}
      {session === null ? null : (
        <>
          {session.failure !== undefined &&
          session.failure !== null &&
          session.status !== "sealed" &&
          !running ? (
            <SessionFailure
              failure={session.failure}
              busy={busy}
              onRetry={() => resume(session)}
            />
          ) : null}
          {board(session)}
        </>
      )}
    </Shell>
  )
}
