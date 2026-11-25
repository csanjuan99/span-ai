"use client"

import { CheckIcon, LockIcon } from "lucide-react"
import { useEffect, useState } from "react"

import {
  jiraSessionOf,
  noSession,
  readJiraSession,
  type JiraSession,
} from "@/lib/jira-session"

import { Pill } from "@/components/onboarding/Pill"
import { ActionPanel } from "@/components/shaping/session-ui"
import { Button } from "@/components/ui/button"
import { touchButton } from "@/lib/button-class"
import { cn } from "@/lib/utils"
import { postForm } from "@/lib/api"
import { reasonText } from "@/lib/guards"

import { TextField, type EditProps } from "./fields"

/** What Span knows about the Jira account saved for the folder. Setting it up lives on the desk. */
export function JiraSection({
  folder,
  props,
}: {
  folder: string
  props: EditProps
}) {
  const [session, setSession] = useState(noSession)
  const [failed, setFailed] = useState(false)
  const [note, setNote] = useState("")
  const [confirming, setConfirming] = useState(false)

  useEffect(() => {
    let live = true
    void readJiraSession(folder)
      .then((body) => {
        if (live) {
          setSession(jiraSessionOf(body))
          setFailed(false)
        }
      })
      .catch(() => {
        if (live) {
          setSession(noSession)
          setFailed(true)
        }
      })
    return () => {
      live = false
    }
  }, [folder])

  async function closeSession() {
    const data = new URLSearchParams()
    data.set("project", folder)
    try {
      await postForm("/jira/logout", data)
      setSession(noSession)
      setNote("Session closed for this folder.")
    } catch (err: unknown) {
      setNote(reasonText(err, "the session was not closed"))
    } finally {
      setConfirming(false)
    }
  }

  return (
    <>
      <JiraAccount session={session} failed={failed} />
      {session.configured && !confirming ? (
        <Button
          type="button"
          variant="outline"
          className={cn("self-start", touchButton)}
          onClick={() => setConfirming(true)}
        >
          <LockIcon aria-hidden="true" />
          Close session
        </Button>
      ) : null}
      {session.configured && confirming ? (
        <ActionPanel
          title={`Close the Jira session for ${folder}?`}
          icon={LockIcon}
          rows={[
            {
              label: "Closes",
              body: "Span forgets the token and the site for this folder. Look ups ask you to connect again.",
            },
            {
              label: "Stays",
              body: "Nothing is deleted from Jira. Stories, comments and statuses stay as they are. Runs, branches and the stories copied into runs stay too.",
            },
            {
              label: "Next",
              body: "Reconnect any time from the start of a run, or here.",
            },
          ]}
        >
          <div className="flex flex-wrap items-center gap-3">
            <Button
              type="button"
              variant="outline"
              className={cn(touchButton, "border-fail text-fail")}
              onClick={() => void closeSession()}
            >
              <LockIcon aria-hidden="true" />
              Close session
            </Button>
            <Button
              type="button"
              variant="ghost"
              className={touchButton}
              onClick={() => setConfirming(false)}
            >
              Keep connected
            </Button>
          </div>
        </ActionPanel>
      ) : null}
      {note !== "" ? (
        <p role="status" className="text-sm text-pretty">
          {note}
        </p>
      ) : null}
      <TextField
        props={props}
        name="doneStatus"
        label="Move story on done"
        fallback="leave the story where it is"
        placeholder="Done"
      />
    </>
  )
}

/** Which Jira account is saved for the folder, or why it is not known. */
function JiraAccount({
  session,
  failed,
}: {
  session: JiraSession
  failed: boolean
}) {
  if (failed) {
    return (
      <p className="text-sm text-pretty text-hold">
        The Jira session could not be read.
      </p>
    )
  }
  if (!session.configured) {
    return (
      <p className="text-sm text-pretty text-muted-foreground">
        Not connected. Save your Jira account from the desk and stories load
        there.
      </p>
    )
  }
  return (
    <div className="flex flex-col gap-1 pb-2 text-sm">
      <div className="flex flex-wrap items-center gap-2">
        <span>{session.email}</span>
        <Pill tone="pass">
          <CheckIcon className="size-3.5" aria-hidden="true" />
          Connected
        </Pill>
      </div>
      <span className="font-mono text-xs text-muted-foreground">
        {session.baseUrl}
      </span>
      <span className="text-xs text-muted-foreground">
        The session belongs to this folder.
      </span>
    </div>
  )
}
