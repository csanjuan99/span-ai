"use client"

import { useEffect, useState } from "react"
import { Link, useNavigate, useParams, useSearchParams } from "react-router"

import { useBusyAction } from "@/hooks/use-busy-action"
import { reasonText } from "@/lib/guards"

import {
  acceptShapeContext,
  answerShape,
  getShape,
  sealShape,
  startRun,
} from "@/lib/api"
import { ContextProposals } from "@/components/shaping/ContextProposals"
import { ExampleMap } from "@/components/shaping/ExampleMap"
import { JiraPanel } from "@/components/shaping/JiraPanel"
import { Questions } from "@/components/shaping/Questions"
import { Split } from "@/components/shaping/Split"
import { Story } from "@/components/shaping/Story"
import { Shell } from "@/components/shell/Shell"
import { Button } from "@/components/ui/button"
import type { ShapeSession as Session } from "@/lib/types"
import { PromptHeading } from "@/components/shell/PromptHeading"

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

/** One shaping session: the map, the questions, then the sealed story and what to do with it. */
export function ShapeSession() {
  const { id = "" } = useParams()
  const [params] = useSearchParams()
  const workspace = params.get("workspace") ?? "."
  const navigate = useNavigate()
  const [session, setSession] = useState<Session | null>(null)
  const [note, setNote] = useState("")

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

  function turn(task: () => Promise<Session>) {
    void act(async () => setSession(await task()))
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

  const open =
    session?.questions.filter((question) => question.status === "open") ?? []
  return (
    <Shell path={`~/span/shaping/${id}`} status={session?.status ?? "shaping"}>
      <header>
        <PromptHeading as="h1">shape a story</PromptHeading>
        <p className="mt-1 text-sm">
          <Link
            className="text-primary underline-offset-4 hover:underline"
            to="/shaping"
          >
            All sessions
          </Link>
        </p>
        {session !== null ? (
          <p className="text-pretty text-muted-foreground">{session.idea}</p>
        ) : null}
      </header>
      {note !== "" ? (
        <p role="alert" className="text-sm text-pretty text-destructive">
          {note}
        </p>
      ) : null}
      {session === null ? null : (
        <>
          <ExampleMap
            rules={session.rules}
            examples={session.examples}
            questions={session.questions}
          />
          {session.status === "split" ? <Split split={session.split} /> : null}
          {session.status === "interviewing" && open.length > 0 ? (
            <Questions
              key={open.map((question) => question.id).join(",")}
              questions={open}
              busy={busy}
              onSubmit={(answers, assume) =>
                turn(() => answerShape(session.id, workspace, answers, assume))
              }
            />
          ) : null}
          {session.status !== "sealed" ? (
            <ContextProposals
              proposals={session.proposals}
              busy={busy}
              onAccept={(accept) =>
                turn(() => acceptShapeContext(session.id, workspace, accept))
              }
            />
          ) : null}
          {session.status === "ready" || session.status === "sealed" ? (
            <Story
              story={session.story}
              digest={session.digest}
              sealed={session.status === "sealed"}
              busy={busy}
              onSeal={() => turn(() => sealShape(session.id, workspace))}
            />
          ) : null}
          {session.status === "sealed" ? (
            <>
              <JiraPanel
                session={session}
                onSession={setSession}
                onReload={reload}
              />
              <Button
                type="button"
                className="self-start"
                disabled={busy}
                onClick={() => beginRun(session)}
              >
                Start run
              </Button>
            </>
          ) : null}
        </>
      )}
    </Shell>
  )
}
