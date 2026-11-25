"use client"

import type { FormEvent, ReactNode } from "react"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import type { JiraSession } from "@/lib/jira-session"

export interface StoryHit {
  key: string
  summary: string
}

/** N2: Jira story lookup inside the start form. */
export function JiraLookup({
  session,
  issue,
  onIssue,
  url,
  onUrl,
  email,
  onEmail,
  token,
  onToken,
  stories,
  note,
  onLoadOne,
  onLoadMine,
  onSaveSession,
  onPick,
}: {
  session: JiraSession
  issue: string
  onIssue: (value: string) => void
  url: string
  onUrl: (value: string) => void
  email: string
  onEmail: (value: string) => void
  token: string
  onToken: (value: string) => void
  stories: StoryHit[]
  note: string
  onLoadOne: () => void
  onLoadMine: () => void
  onSaveSession: (event: FormEvent<HTMLFormElement>) => void
  onPick: (key: string) => void
}) {
  return (
    <MotionJira session={session}>
      <div className="flex flex-col gap-2">
        <Label htmlFor="issue">Story key</Label>
        <Input
          id="issue"
          name="issue"
          value={issue}
          onChange={(event) => onIssue(event.target.value)}
        />
        <div className="flex flex-wrap gap-2">
          <Button
            type="button"
            variant="secondary"
            className="min-h-11"
            onClick={onLoadOne}
          >
            Look up
          </Button>
          <Button
            type="button"
            variant="secondary"
            className="min-h-11"
            onClick={onLoadMine}
          >
            Search stories
          </Button>
        </div>
        {note !== "" ? (
          <p className="text-sm text-pretty text-destructive">{note}</p>
        ) : null}
        {stories.length > 0 ? (
          <ul className="flex flex-col gap-1 text-sm">
            {stories.map((item) => (
              <li key={item.key}>
                <button
                  type="button"
                  className="min-h-11 text-left text-primary underline-offset-4 hover:underline"
                  onClick={() => onPick(item.key)}
                >
                  {item.key} {item.summary}
                </button>
              </li>
            ))}
          </ul>
        ) : null}
      </div>
      {session.configured ? (
        <p className="text-sm text-pretty">
          Connected as {session.email}. You can close the session under
          settings.
        </p>
      ) : (
        <form className="flex max-w-md flex-col gap-3" onSubmit={onSaveSession}>
          <p className="text-sm text-pretty text-muted-foreground">
            Connect Jira with an API token. It is stored for this folder only.
          </p>
          <div className="flex flex-col gap-2">
            <Label htmlFor="jira-url">Jira URL</Label>
            <Input
              id="jira-url"
              value={url}
              onChange={(e) => onUrl(e.target.value)}
            />
          </div>
          <div className="flex flex-col gap-2">
            <Label htmlFor="jira-email">Email</Label>
            <Input
              id="jira-email"
              value={email}
              onChange={(e) => onEmail(e.target.value)}
            />
          </div>
          <div className="flex flex-col gap-2">
            <Label htmlFor="jira-token">Token</Label>
            <Input
              id="jira-token"
              type="password"
              autoComplete="off"
              value={token}
              onChange={(e) => onToken(e.target.value)}
            />
          </div>
          <Button type="submit" className="min-h-11 self-start">
            Connect
          </Button>
        </form>
      )}
    </MotionJira>
  )
}

function MotionJira({
  session,
  children,
}: {
  session: JiraSession
  children: ReactNode
}) {
  return (
    <div className="flex flex-col gap-4" aria-label="Jira story lookup">
      {!session.configured ? (
        <p className="text-xs text-muted-foreground">Not connected</p>
      ) : null}
      {children}
    </div>
  )
}
