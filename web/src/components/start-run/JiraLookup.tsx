"use client"

import {
  CheckIcon,
  ExternalLinkIcon,
  LinkIcon,
  LockIcon,
  RefreshCwIcon,
  SearchIcon,
  WifiOffIcon,
  type LucideIcon,
} from "lucide-react"
import { useState, type ReactNode } from "react"

import { IdChip } from "@/components/list-controls/IdChip"
import { Pill as StatusPill } from "@/components/onboarding/Pill"
import { KeptNext } from "@/components/problem/KeptNext"
import { problemTones } from "@/components/problem/ProblemCard"
import { Panel } from "@/components/shaping/session-ui"
import { noteText } from "@/components/shaping/styles"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import type { JiraSession } from "@/lib/jira-session"
import { cn } from "@/lib/utils"

/** One story of a Jira search. */
export interface StoryHit {
  key: string
  summary: string
  status: string
}

/** A story Jira answered for, with what the page can show of it. */
export interface FoundStory {
  key: string
  title: string
  status: string
  type: string
  url: string
  acceptance: string
}

/** Where a lookup stands: nothing asked, asking, answered, or turned away. */
export type Lookup =
  | { phase: "idle" }
  | { phase: "asking"; key: string }
  | { phase: "found"; story: FoundStory }
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
  /** The words of the last search, for the heading over its results. */
  searched: string
  lookup: Lookup
  /** What happened to the last attempt to save the Jira account. */
  note: string
  onLoadOne: () => void
  onSearch: (words: string) => void
  onSaveSession: () => void
  onPick: (key: string) => void
  onPasteInstead: () => void
}

interface Failure {
  mark: string
  icon: LucideIcon
  title: string
  body: string
  kept: string
  next: string
}

/** What a refused lookup says, from what Jira answered. */
function failureOf(
  lookup: { key: string; reason: string },
  host: string,
  email: string
): Failure {
  const { key, reason } = lookup
  if (/404|not found/i.test(reason)) {
    return {
      mark: "Not found",
      icon: SearchIcon,
      title: `${key} is not in project ${key.split("-")[0] ?? ""}`.trim(),
      body: "Jira answered 404. The key may have a typo, or the story was deleted or moved to another project.",
      kept: "The key you typed and every other choice on this screen.",
      next: "Check the key, or search by words from the title.",
    }
  }
  if (/403|permission|forbidden|rejected/i.test(reason)) {
    return {
      mark: "No permission",
      icon: LockIcon,
      title: `Your Jira account cannot open ${key}`,
      body: `Jira answered 403${email === "" ? "" : ` for ${email}`}. The story exists, but its project or security level is closed to this account.`,
      kept: "Your connection to Jira and the key you typed.",
      next: "Ask the project admin for access, or paste the story text.",
    }
  }
  return {
    mark: "Offline",
    icon: WifiOffIcon,
    title: host === "" ? "Span cannot reach Jira" : `Span cannot reach ${host}`,
    body: "Jira did not answer. Your network or Jira is down. Span itself is fine.",
    kept: "The key, the connection and everything on this screen.",
    next: "Retry when you are back online, or paste the story text and carry on.",
  }
}

function hostOf(baseUrl: string): string {
  try {
    return new URL(baseUrl).host
  } catch {
    return ""
  }
}

function Asking({ lookup, host }: { lookup: { key: string }; host: string }) {
  return (
    <Panel role="status" className="flex flex-col gap-2 p-4">
      <p className="text-sm font-semibold">
        Asking {host === "" ? "Jira" : host} for{" "}
        <span className="font-mono">{lookup.key}</span>…
      </p>
      <span className="h-3 w-1/2 rounded bg-shimmer" />
      <span className="h-3 w-full rounded bg-shimmer" />
      <span className="h-12 w-full rounded bg-shimmer" />
      <p className={noteText}>
        Usually under 2 s. You can edit the key while you wait.
      </p>
    </Panel>
  )
}

function Found({ story }: { story: FoundStory }) {
  const criteria = story.acceptance
    .split("\n")
    .map((line) => line.replace(/^[-*\s]+/, "").trim())
    .filter((line) => line !== "")
  return (
    <Panel className="flex flex-col gap-3 p-4">
      <div className="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
        <IdChip>{story.key}</IdChip>
        {story.status !== "" ? (
          <StatusPill tone="mut" size="sm">
            {story.status}
          </StatusPill>
        ) : null}
        {story.type !== "" ? <span>{story.type}</span> : null}
      </div>
      <h3 className="text-base font-semibold text-balance">{story.title}</h3>
      {criteria.length > 0 ? (
        <div className="flex flex-col gap-1.5">
          <p className="text-xs font-semibold text-muted-foreground">
            Acceptance criteria · {criteria.length}
          </p>
          <ol className="flex flex-col gap-1.5 text-[13px] leading-5">
            {criteria.map((line, index) => (
              <li key={line} className="flex gap-2">
                <span className="font-mono text-xs font-semibold text-primary">
                  AC{index + 1}
                </span>
                {line}
              </li>
            ))}
          </ol>
        </div>
      ) : null}
      {story.url !== "" ? (
        <JiraLink
          href={story.url}
          className="self-start rounded-lg border border-border px-3 hover:bg-muted md:min-h-9"
        >
          Open in Jira
        </JiraLink>
      ) : null}
      <p className={noteText}>
        Read just now. Span copies the text into the run; later edits in Jira do
        not change it.
      </p>
    </Panel>
  )
}

function Refused({
  lookup,
  session,
  props,
}: {
  lookup: { key: string; reason: string }
  session: JiraSession
  props: JiraProps
}) {
  const failure = failureOf(lookup, hostOf(session.baseUrl), session.email)
  const Icon = failure.icon
  const kind = failure.mark === "Offline" ? "fail" : "hold"
  const open = `${session.baseUrl.replace(/\/$/, "")}/browse/${lookup.key}`
  return (
    <Panel role="alert" className="overflow-hidden">
      <div
        className={cn(
          "flex items-center gap-2 px-4 py-2.5 text-xs font-medium",
          problemTones[kind]
        )}
      >
        <Icon className="size-4" aria-hidden="true" />
        {failure.mark}
      </div>
      <div className="flex flex-col gap-3 p-4">
        <h3 className="text-base font-semibold text-balance">
          {failure.title}
        </h3>
        <p className="text-sm text-pretty">{failure.body}</p>
        <KeptNext kept={failure.kept} next={failure.next} marked={false} />
        <div className="flex flex-wrap items-center gap-3">
          {failure.mark === "Not found" ? (
            <Button type="button" onClick={() => props.onSearch(lookup.key)}>
              <SearchIcon aria-hidden="true" />
              Search by text
            </Button>
          ) : null}
          {failure.mark === "Offline" ? (
            <Button type="button" onClick={props.onLoadOne}>
              <RefreshCwIcon aria-hidden="true" />
              Retry look up
            </Button>
          ) : null}
          <PasteInsteadButton onPaste={props.onPasteInstead} />
          {failure.mark === "No permission" && session.baseUrl !== "" ? (
            <JiraLink
              href={open}
              className="text-primary underline-offset-4 hover:underline"
            >
              Open {lookup.key} in Jira
            </JiraLink>
          ) : null}
        </div>
      </div>
    </Panel>
  )
}

function Answer({ props }: { props: JiraProps }) {
  const { lookup, session } = props
  if (lookup.phase === "asking") {
    return <Asking lookup={lookup} host={hostOf(session.baseUrl)} />
  }
  if (lookup.phase === "found") {
    return <Found story={lookup.story} />
  }
  if (lookup.phase === "failed") {
    return <Refused lookup={lookup} session={session} props={props} />
  }
  return null
}

function Results({ props }: { props: JiraProps }) {
  if (props.stories.length === 0) {
    return null
  }
  return (
    <Panel className="flex flex-col p-4">
      <h3 className="pb-2 text-sm font-semibold">
        {props.stories.length} stories
        {props.searched !== "" ? <> match “{props.searched}”</> : null}
        <span className="pl-2 text-xs font-normal text-muted-foreground">
          newest first
        </span>
      </h3>
      <ul className="flex flex-col">
        {props.stories.map((item) => (
          <li
            key={item.key}
            className="flex flex-wrap items-center gap-3 border-t border-border py-2 text-sm"
          >
            <IdChip>{item.key}</IdChip>
            <span className="min-w-0 flex-1 text-pretty">{item.summary}</span>
            {item.status !== "" ? (
              <StatusPill
                tone={item.status === "Done" ? "pass" : "mut"}
                size="sm"
              >
                {item.status}
              </StatusPill>
            ) : null}
            <Button
              type="button"
              variant="outline"
              className="h-11 md:h-9"
              onClick={() => props.onPick(item.key)}
            >
              <CheckIcon aria-hidden="true" />
              Use
            </Button>
          </li>
        ))}
      </ul>
      <p className={cn(noteText, "pt-2")}>
        A story marked Done in Jira can still start a run. Span shows the status
        so you can check.
      </p>
    </Panel>
  )
}

function ConnectPanel({ props }: { props: JiraProps }) {
  const key = props.issue.trim()
  return (
    <Panel aria-label="Connect Jira" className="flex flex-col gap-3 p-4">
      <p className="flex flex-wrap items-center gap-2 text-sm font-semibold">
        Connect Jira
        {key !== "" ? (
          <>
            {" "}
            to look up <span className="font-mono">{key}</span>
          </>
        ) : null}
        <StatusPill tone="mut" size="sm">
          Optional
        </StatusPill>
      </p>
      <p className={noteText}>
        Same connection as in setup. Nothing leaves this screen: your choices
        stay where they are.
      </p>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="jira-url">Jira site</Label>
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
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="jira-token">API token</Label>
        <Input
          id="jira-token"
          type="password"
          autoComplete="off"
          placeholder="Paste your token"
          value={props.token}
          onChange={(event) => props.onToken(event.target.value)}
        />
      </div>
      {props.note !== "" ? (
        <p role="alert" className="text-sm text-pretty text-destructive">
          {props.note}
        </p>
      ) : null}
      <div className="flex flex-wrap items-center gap-3">
        <Button type="button" onClick={props.onSaveSession}>
          <LinkIcon aria-hidden="true" />
          {key !== "" ? `Connect and look up ${key}` : "Connect"}
        </Button>
        <PasteInsteadButton onPaste={props.onPasteInstead} />
      </div>
      <p className={noteText}>
        Create a token at{" "}
        <span className="font-mono">
          id.atlassian.com/manage-profile/security/api-tokens
        </span>
        . Span keeps it for this folder only.
      </p>
    </Panel>
  )
}

/** A link out to Jira that opens in a new tab. */
function JiraLink({
  href,
  className,
  children,
}: {
  href: string
  className: string
  children: ReactNode
}) {
  return (
    <a
      className={cn(
        "inline-flex min-h-11 items-center gap-1.5 text-sm",
        className
      )}
      href={href}
      target="_blank"
      rel="noreferrer"
    >
      <ExternalLinkIcon className="size-4" aria-hidden="true" />
      {children}
    </a>
  )
}

/** The way out of Jira for a person who has the text already. */
function PasteInsteadButton({ onPaste }: { onPaste: () => void }) {
  return (
    <Button type="button" variant="ghost" onClick={onPaste}>
      Paste the story text instead
    </Button>
  )
}

/** A labelled input with the button that acts on it: the key lookup and the word search. */
function LookupField({
  id,
  name,
  label,
  value,
  onChange,
  action,
  disabled,
  onAction,
}: {
  id: string
  name?: string
  label: string
  value: string
  onChange: (value: string) => void
  action: string
  disabled: boolean
  onAction: () => void
}) {
  return (
    <div className="flex flex-col gap-1.5">
      <Label htmlFor={id}>{label}</Label>
      <div className="flex gap-2">
        <Input
          id={id}
          name={name}
          className="h-11 min-w-0 flex-1 md:h-10"
          value={value}
          onChange={(event) => onChange(event.target.value)}
        />
        <Button
          type="button"
          variant="outline"
          className="h-11 md:h-10"
          disabled={disabled}
          onClick={onAction}
        >
          <SearchIcon aria-hidden="true" />
          {action}
        </Button>
      </div>
    </div>
  )
}

/** N2: look a Jira story up by key or by words, connecting Jira first when it is not. */
export function JiraLookup(props: JiraProps) {
  const { session, issue } = props
  const [words, setWords] = useState("")
  const [byWords, setByWords] = useState(false)
  const connected = session.configured
  return (
    <div className="flex flex-col gap-3" aria-label="Jira story lookup">
      {byWords ? (
        <LookupField
          id="words"
          label="Search Jira by words"
          value={words}
          onChange={setWords}
          action="Search"
          disabled={!connected || words.trim() === ""}
          onAction={() => props.onSearch(words.trim())}
        />
      ) : (
        <LookupField
          id="issue"
          name="issue"
          label="Story key"
          value={issue}
          onChange={props.onIssue}
          action="Look up"
          disabled={!connected}
          onAction={props.onLoadOne}
        />
      )}
      {connected ? (
        <p className={noteText}>
          Connected as {session.email}.{" "}
          <button
            type="button"
            className="min-h-11 text-primary underline-offset-4 hover:underline md:min-h-0"
            onClick={() => setByWords((on) => !on)}
          >
            {byWords ? "Look up by key" : "Search by words"}
          </button>
        </p>
      ) : (
        <ConnectPanel props={props} />
      )}
      <Answer props={props} />
      <Results props={props} />
    </div>
  )
}
