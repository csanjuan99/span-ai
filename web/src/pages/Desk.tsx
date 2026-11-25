"use client"

import { EmptyDesk, sampleBrief } from "@/components/desk/EmptyDesk"
import { Toasts } from "@/components/notify/Toasts"
import { useNotifications } from "@/hooks/use-notifications"
import { RunsDesk } from "@/components/desk/RunsDesk"
import { StartRunPanel } from "@/components/start-run/StartRunPanel"
import {
  refusalOf,
  tokenRefusal,
  type StartRefusal,
} from "@/components/start-run/types"
import { useEffect, useRef, useState, type FormEvent } from "react"
import { Link, useLocation, useNavigate } from "react-router"

import {
  agentList,
  branchChoice,
  getJSON,
  postForm,
  spanHeadsOf,
  startRun,
  UnauthorizedError,
  type Started,
} from "@/lib/api"
import {
  agentReviewMs,
  deskAgentLine,
  timelineLines,
  withAgent,
  type TimelineAction,
} from "@/lib/timeline"
import type {
  AgentChoice,
  AgentStatus,
  BranchChoice,
  SpanHead,
} from "@/lib/types"
import { isRecord, reasonText } from "@/lib/guards"
import {
  jiraSessionOf,
  noSession,
  readJiraSession,
  type JiraSession,
} from "@/lib/jira-session"
import { repoNames } from "@/lib/repos"
import { Shell } from "@/components/shell/Shell"
import type { StoryHit } from "@/components/start-run/JiraLookup"
import { cn } from "@/lib/utils"
import { touchArea } from "@/lib/button-class"

const noBranches: BranchChoice = { branches: [], defaultBase: "dev" }

function reposOf(value: unknown) {
  const names = repoNames(value)
  return names.map((name) => ({ name, checked: names.length === 1 }))
}

function storiesOf(value: unknown): StoryHit[] {
  if (!isRecord(value) || !Array.isArray(value.issues)) {
    return []
  }
  return value.issues.filter(isRecord).map((item) => ({
    key: typeof item.key === "string" ? item.key : "",
    summary: typeof item.summary === "string" ? item.summary : "",
  }))
}

function storyOf(value: unknown): { key: string; brief: string } {
  if (!isRecord(value)) {
    return { key: "", brief: "" }
  }
  return {
    key: typeof value.key === "string" ? value.key : "",
    brief: typeof value.brief === "string" ? value.brief : "",
  }
}

/** Desk: list runs or start a new one (N1–N4). */
export function Desk() {
  const navigate = useNavigate()
  const location = useLocation()
  const [workspace, setWorkspace] = useState(".")
  const [repos, setRepos] = useState<{ name: string; checked: boolean }[]>([])
  const [session, setSession] = useState<JiraSession>(noSession)
  const [stories, setStories] = useState<StoryHit[]>([])
  const [issue, setIssue] = useState("")
  const [story, setStory] = useState(() =>
    (location.state as { sample?: boolean } | null)?.sample === true
      ? sampleBrief.text
      : ""
  )
  const [kind, setKind] = useState("infer")
  const [base, setBase] = useState("")
  const [branches, setBranches] = useState<BranchChoice>(noBranches)
  const [spans, setSpans] = useState<SpanHead[]>([])
  const [listed, setListed] = useState(false)
  const [listOk, setListOk] = useState(false)
  const notifications = useNotifications(listOk ? spans : null)
  const formRef = useRef<HTMLFormElement>(null)
  const [starting, setStarting] = useState(false)
  const [now, setNow] = useState(() => new Date())
  const [url, setUrl] = useState("")
  const [email, setEmail] = useState("")
  const [token, setToken] = useState("")
  const [jiraNote, setJiraNote] = useState("")
  const [folderNote, setFolderNote] = useState("")
  const [storyNote, setStoryNote] = useState("")
  const [runNote, setRunNote] = useState("")
  const [refusal, setRefusal] = useState<StartRefusal | null>(null)
  const [started, setStarted] = useState<Started | null>(null)
  const [actions, setActions] = useState<TimelineAction[]>([])
  const [agents, setAgents] = useState<AgentStatus[]>([])
  const [agentAt, setAgentAt] = useState("")
  const [listNote, setListNote] = useState("")
  const [pick, setPick] = useState<AgentChoice | null>(null)
  const lines = withAgent(
    timelineLines({ statements: [], frames: [], actions }),
    deskAgentLine(agents, agentAt)
  )

  function note(text: string) {
    setActions((current) => [
      ...current,
      { at: new Date().toISOString(), text },
    ])
  }

  useEffect(() => {
    let closed = false
    void getJSON("/spans")
      .then((spanBody) => {
        if (!closed) {
          setSpans(spanHeadsOf(spanBody))
          setNow(new Date())
          setListOk(true)
          setListed(true)
        }
      })
      .catch((err: unknown) => {
        if (closed) return
        if (err instanceof UnauthorizedError) {
          setListNote(
            "This server needs its access token. Add it under Settings."
          )
        } else {
          setListNote(reasonText(err, "the desk could not load"))
        }
        setListOk(false)
        setListed(true)
      })
    return () => {
      closed = true
    }
  }, [])

  useEffect(() => {
    let gone = false
    const pull = () => {
      void getJSON("/agents")
        .then((body) => {
          if (!gone) {
            setAgents(agentList(body))
            setAgentAt(new Date().toISOString())
          }
        })
        .catch((err: unknown) => {
          if (!gone) {
            setListNote(reasonText(err, "the agent list could not be read"))
          }
        })
    }
    pull()
    const timer = window.setInterval(pull, agentReviewMs)
    return () => {
      gone = true
      window.clearInterval(timer)
    }
  }, [])

  async function readSession(project: string) {
    if (project.trim() === "") {
      setSession(noSession)
      return
    }
    try {
      const body = await readJiraSession(project)
      setSession(jiraSessionOf(body))
    } catch (err: unknown) {
      const detail = reasonText(err, "")
      setListNote(
        detail === ""
          ? "Jira session not read"
          : `Jira session not read: ${detail}`
      )
    }
  }

  async function readBranches(project: string) {
    if (project.trim() === "") {
      setBranches(noBranches)
      return
    }
    try {
      setBranches(
        branchChoice(
          await getJSON(`/branches?path=${encodeURIComponent(project)}`)
        )
      )
    } catch {
      setBranches(noBranches)
    }
  }

  useEffect(() => {
    const timer = window.setTimeout(() => {
      void readSession(workspace)
      void readBranches(workspace)
    }, 250)
    return () => window.clearTimeout(timer)
  }, [workspace])

  async function findRepos() {
    setFolderNote("")
    const [repoResult, jiraResult] = await Promise.allSettled([
      getJSON(`/repos?path=${encodeURIComponent(workspace)}`),
      readJiraSession(workspace),
    ])
    if (repoResult.status === "fulfilled") {
      setRepos(reposOf(repoResult.value))
    } else {
      setFolderNote(
        reasonText(repoResult.reason, "the folder could not be read")
      )
    }
    if (jiraResult.status === "fulfilled") {
      setSession(jiraSessionOf(jiraResult.value))
    } else {
      setJiraNote(
        reasonText(jiraResult.reason, "the session could not be read")
      )
    }
  }

  async function saveSession(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const data = new URLSearchParams()
    data.set("url", url)
    data.set("email", email)
    data.set("token", token)
    data.set("project", workspace)
    try {
      await postForm("/jira/account", data)
      setToken("")
      setJiraNote("Session saved for this folder.")
      const next = await readJiraSession(workspace)
      setSession(jiraSessionOf(next))
    } catch (err: unknown) {
      setJiraNote(reasonText(err, "the session was not saved"))
    }
  }

  async function loadStory(key: string) {
    setStoryNote("")
    try {
      const body = await getJSON(
        `/jira/issues/${encodeURIComponent(key)}?path=${encodeURIComponent(workspace)}`
      )
      const loaded = storyOf(body)
      setIssue(loaded.key)
      setStory(loaded.brief)
      void readSession(workspace)
    } catch (err: unknown) {
      setStoryNote(reasonText(err, "the story could not be loaded"))
    }
  }

  async function loadMine() {
    setStoryNote("")
    try {
      const body = await getJSON(
        `/jira/search?path=${encodeURIComponent(workspace)}`
      )
      setStories(storiesOf(body))
      void readSession(workspace)
    } catch (err: unknown) {
      setStoryNote(reasonText(err, "the stories could not be loaded"))
    }
  }

  function run(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    void begin(new FormData(event.currentTarget))
  }

  function startSample() {
    setStory(sampleBrief.text)
    setStarting(true)
    if (formRef.current === null) {
      return
    }
    const data = new FormData(formRef.current)
    data.set("story", sampleBrief.text)
    void begin(data)
  }

  async function begin(data: FormData) {
    setRunNote("")
    setStarted(null)
    setRefusal(null)
    if (kind === "infer") {
      data.delete("kind")
    }
    if (base.trim() === "") {
      data.delete("base")
    } else {
      data.set("base", base.trim())
    }
    try {
      const pre = await getJSON(
        `/start/precheck?${precheckQuery(data).toString()}`
      )
      const blocked = refusalOf(pre)
      if (blocked !== null) {
        setRefusal(blocked)
        return
      }
    } catch (err: unknown) {
      if (err instanceof UnauthorizedError) {
        setRefusal(tokenRefusal(err))
        return
      }
      // An old server has no precheck route; the start itself still answers for the story.
      note(reasonText(err, "the precheck could not run"))
    }
    try {
      const result = await startRun(data)
      if (typeof result === "string") {
        setRunNote(result)
        note(result)
        return
      }
      if (result.runs.length > 1 || result.partial) {
        setStarted(result)
        note(
          result.partial
            ? "some runs started"
            : `${result.runs.length} runs started`
        )
        return
      }
      note("run started")
      void navigate(`/spans/${result.id}`)
    } catch (err: unknown) {
      if (err instanceof UnauthorizedError) {
        setRefusal(tokenRefusal(err))
        return
      }
      const message = reasonText(err, "the run did not start")
      setRunNote(message)
      note(message)
    }
  }

  const showStart = starting || (listOk && spans.length === 0)
  const panelOpen = starting

  return (
    <Shell path="~/span" status="desk" lines={lines}>
      {listNote !== "" ? (
        <p className="text-sm text-pretty text-destructive">{listNote}</p>
      ) : null}
      {listed && listOk && spans.length > 0 ? (
        <RunsDesk
          runs={spans}
          now={now}
          onNew={() => setStarting(true)}
          onDeleted={(gone) =>
            setSpans((current) => current.filter((item) => item.id !== gone))
          }
        />
      ) : null}
      {listed && listOk && spans.length === 0 ? (
        <EmptyDesk
          folder={workspace}
          onSample={startSample}
          onStory={() => setStarting(true)}
        />
      ) : null}
      {showStart ? (
        <StartRunPanel
          visible={panelOpen}
          refusal={refusal}
          onDismissRefusal={() => setRefusal(null)}
          started={started}
          onOpenRun={(id) => void navigate(`/spans/${id}`)}
          form={{
            formRef,
            workspace,
            onWorkspace: setWorkspace,
            repos,
            onToggleRepo: (name, checked) =>
              setRepos((current) =>
                current.map((item) =>
                  item.name === name ? { ...item, checked } : item
                )
              ),
            folderNote,
            onFindRepos: () => void findRepos(),
            pick,
            onPickLoaded: setPick,
            issue,
            onIssue: setIssue,
            kind,
            onKind: setKind,
            base,
            onBase: setBase,
            branches,
            story,
            onStory: setStory,
            session,
            jiraUrl: url,
            onJiraUrl: setUrl,
            jiraEmail: email,
            onJiraEmail: setEmail,
            jiraToken: token,
            onJiraToken: setToken,
            stories,
            jiraNote,
            storyNote,
            onLoadStory: () => void loadStory(issue),
            onLoadMine: () => void loadMine(),
            onSaveSession: (event) => void saveSession(event),
            onPickStory: (key) => void loadStory(key),
            runNote,
            onSubmit: run,
          }}
        />
      ) : null}
      <nav
        aria-label="More"
        className="flex flex-wrap gap-x-6 gap-y-1 text-[13px]"
      >
        <Link
          className={cn(
            touchArea,
            "text-primary underline-offset-4 hover:underline"
          )}
          to="/settings"
        >
          Settings
        </Link>
      </nav>
      <Toasts
        toasts={notifications.toasts}
        onOpen={(runId) => navigate(`/spans/${runId}`)}
        onDismiss={notifications.dismiss}
        onHold={notifications.hold}
      />
    </Shell>
  )
}

function precheckQuery(data: FormData): URLSearchParams {
  const q = new URLSearchParams()
  for (const [key, value] of data.entries()) {
    if (typeof value === "string") {
      q.append(key, value)
    }
  }
  return q
}
