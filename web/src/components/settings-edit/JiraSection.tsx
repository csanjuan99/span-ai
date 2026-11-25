"use client"

import { useEffect, useState } from "react"

import { jiraSessionOf, noSession, readJiraSession } from "@/lib/jira-session"

/** What Span knows about the Jira account saved for the folder. Setting it up lives on the desk. */
export function JiraSection({ folder }: { folder: string }) {
  const [session, setSession] = useState(noSession)
  const [failed, setFailed] = useState(false)

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
    <div className="grid grid-cols-[16px_minmax(0,1fr)] gap-2 text-sm">
      <span aria-hidden="true" className="font-mono font-semibold text-pass">
        +
      </span>
      <span>
        <b className="text-pass">Connected.</b> {session.email}{" "}
        <span className="font-mono text-xs text-muted-foreground">
          {session.baseUrl}
        </span>
      </span>
    </div>
  )
}
