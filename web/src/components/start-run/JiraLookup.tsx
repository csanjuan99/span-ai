"use client"

import { LinkIcon, SearchIcon } from "lucide-react"

import { KeptNext } from "@/components/problem/KeptNext"
import { IdChip } from "@/components/list-controls/IdChip"
import { Panel } from "@/components/shaping/session-ui"
import { noteText } from "@/components/shaping/styles"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import type { JiraSession } from "@/lib/jira-session"

/** One story of a Jira search. */
export interface StoryHit {
  key: string
  summary: string
}

/** Where a lookup stands: nothing asked, asking, answered, or turned away. */
export type Lookup =
  | { phase: "idle" }
  | { phase: "asking"; key: string }
  | { phase: "found"; key: string; title: string }
  | { phase: "failed"; key: string; reason: string }

/** The Jira side of the form: the account, the key, the search and the answer. */
export interface JiraProps {
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
  lookup: Lookup
  /** What happened to the last attempt to save the Jira account. */
  note: string
  onLoadOne: () => void
  onLoadMine: () => void
  onSaveSession: () => void
  onPick: (key: string) => void
  onPasteInstead: () => void
}

const failures = [
  {
    match: /404|not found/i,
    mark: "Not found",
    kept: "The key you typed and every other choice on this screen.",
    next: "Check the key, or search by words from the title.",
  },
  {
    match: /403|permission|forbidden/i,
    mark: "No permission",
    kept: "Your connection to Jira and the key you typed.",
    next: "Ask the project admin for access, or paste the story text.",
  },
]

function failureOf(reason: string) {
  return (
    failures.find((item) => item.match.test(reason)) ?? {
      mark: "Jira did not answer",
      kept: "The key you typed and every other choice on this screen.",
      next: "Try again in a moment, or paste the story text.",
    }
  )
}

function Answer({
  lookup,
  onPasteInstead,
}: {
  lookup: Lookup
  onPasteInstead: () => void
}) {
  if (lookup.phase === "asking") {
    return (
      <Panel
        role="status"
        className="flex flex-col gap-2 p-4 text-sm font-semibold"
      >
        Asking Jira for <span className="font-mono">{lookup.key}</span>…
        <span className="bg-shimmer h-3 w-1/2 rounded" />
        <span className="bg-shimmer h-3 w-full rounded" />
      </Panel>
    )
  }
  if (lookup.phase === "found") {
    return (
      <Panel className="flex flex-col gap-2 p-4">
        <IdChip>{lookup.key}</IdChip>
        <h3 className="text-base font-semibold">{lookup.title}</h3>
        <p className={noteText}>
          Read just now. Span copies the text into the run; later edits in Jira
          do not change it.
        </p>
      </Panel>
    )
  }
  if (lookup.phase === "failed") {
    const failure = failureOf(lookup.reason)
    return (
      <Panel role="alert" className="flex flex-col gap-3 p-4">
        <span className="text-xs font-semibold text-hold">{failure.mark}</span>
        <p className="text-sm text-pretty">{lookup.reason}</p>
        <KeptNext kept={failure.kept} next={failure.next} />
        <Button
          type="button"
          variant="ghost"
          className="self-start"
          onClick={onPasteInstead}
        >
          Paste the story text instead
        </Button>
      </Panel>
    )
  }
  return null
}

/** N2: look a Jira story up by key, or connect Jira first. */
export function JiraLookup(props: JiraProps) {
  const { session, issue, lookup } = props
  if (!session.configured) {
    return (
      <Panel
        aria-label="Connect Jira"
        className="flex flex-col gap-3 p-4"
      >
        <p className="text-sm text-pretty">
          Connect Jira with an API token. It is stored for this folder only.
        </p>
        <div className="grid gap-3 md:grid-cols-2">
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="jira-url">Jira URL</Label>
            <Input
              id="jira-url"
              value={props.url}
              onChange={(event) => props.onUrl(event.target.value)}
            />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="jira-email">Email</Label>
            <Input
              id="jira-email"
              value={props.email}
              onChange={(event) => props.onEmail(event.target.value)}
            />
          </div>
        </div>
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="jira-token">Token</Label>
          <Input
            id="jira-token"
            type="password"
            autoComplete="off"
            value={props.token}
            onChange={(event) => props.onToken(event.target.value)}
          />
        </div>
        {props.note !== "" ? (
          <p className="text-sm text-pretty text-destructive">{props.note}</p>
        ) : null}
        <div className="flex flex-wrap items-center gap-3">
          <Button type="button" onClick={props.onSaveSession}>
            <LinkIcon aria-hidden="true" />
            Connect
          </Button>
          <Button type="button" variant="ghost" onClick={props.onPasteInstead}>
            Paste the story text instead
          </Button>
        </div>
      </Panel>
    )
  }
  return (
    <div className="flex flex-col gap-3" aria-label="Jira story lookup">
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="issue">Story key</Label>
        <div className="flex gap-2">
          <Input
            id="issue"
            name="issue"
            className="h-11 min-w-0 flex-1 md:h-10"
            value={issue}
            onChange={(event) => props.onIssue(event.target.value)}
          />
          <Button
            type="button"
            variant="outline"
            className="h-11 md:h-10"
            onClick={props.onLoadOne}
          >
            <SearchIcon aria-hidden="true" />
            Look up
          </Button>
          <Button
            type="button"
            variant="outline"
            className="h-11 md:h-10"
            onClick={props.onLoadMine}
          >
            Search stories
          </Button>
        </div>
      </div>
      <p className={noteText}>Connected as {session.email}.</p>
      <Answer lookup={lookup} onPasteInstead={props.onPasteInstead} />
      {props.stories.length > 0 ? (
        <ul className="flex flex-col gap-1 text-sm">
          {props.stories.map((item) => (
            <li key={item.key}>
              <button
                type="button"
                className="min-h-11 text-left text-primary underline-offset-4 hover:underline"
                onClick={() => props.onPick(item.key)}
              >
                {item.key} {item.summary}
              </button>
            </li>
          ))}
        </ul>
      ) : null}
    </div>
  )
}
