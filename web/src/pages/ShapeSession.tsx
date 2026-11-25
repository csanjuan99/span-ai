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
} from "@/lib/api"
import { InterviewStopped } from "@/components/shape-entry/InterviewStopped"
import { ContextBoard } from "@/components/shaping/ContextBoard"
import { JiraBoard } from "@/components/shaping/JiraBoard"
import { MapBoard } from "@/components/shaping/MapBoard"
import { QuestionsBoard } from "@/components/shaping/QuestionsBoard"
import { SealBoard } from "@/components/shaping/SealBoard"
import { SealedBoard } from "@/components/shaping/SealedBoard"
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
import { loadProgress, saveProgress } from "@/lib/shape-draft"

const readFailure = "the session could not be read"

/** One shaping session: the interview, the map, the split, the context, the seal and what to do with the sealed story. */
export function ShapeSession() {
  const { id = "" } = useParams()
  const [params] = useSearchParams()
  const workspace = params.get("workspace") ?? "."
  const navigate = useNavigate()
  const [session, setSession] = useState<Session | null>(null)
  const [note, setNote] = useState("")
  // What the person did on this page: the server only knows the session's status.
  // The map and context steps are remembered per session, so a reload lands where the person was.
  const [seen, setSeen] = useState<SessionProgress & { id: string }>(() => ({
    ...loadProgress(id),
    id,
  }))
  const [revisit, setRevisit] = useState(false)
  const [running, setRunning] = useState(false)
  // "Wait and resume later" hides the stopped band for the failure it was pressed on.
  const [waited, setWaited] = useState("")

  const progress: SessionProgress = seen.id === id ? seen : loadProgress(id)
  function advance(change: SessionProgress) {
    const next = { ...progress, ...change, id }
    saveProgress(id, next)
    setSeen(next)
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

  /** Shows the session a turn returned, from its own step rather than a revisited one. */
  function adopt(next: Session) {
    setSession(next)
    setRevisit(false)
  }

  // A turn also marks the page as working, so the screen says so until the agent answers.
  function turn(task: () => Promise<Session>) {
    setRunning(true)
    void act(async () => {
      try {
        adopt(await task())
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

  /** Hands the sealed story to the desk's start form, locked, so the run is started there. */
  function beginRun(current: Session) {
    void navigate("/", {
      state: {
        shape: {
          id: current.id,
          title: current.title,
          workspace: current.workspace,
          story: current.story,
          issue: current.jira?.key ?? "",
          repo: current.repo,
        },
      },
    })
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
              saveProgress(id, {})
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
          session.failure.at !== waited &&
          !running ? (
            <InterviewStopped
              session={session}
              onResumed={adopt}
              onLater={() => setWaited(session.failure?.at ?? "")}
            />
          ) : null}
          {board(session)}
        </>
      )}
    </Shell>
  )
}
