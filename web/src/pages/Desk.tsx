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
  getPrecheck,
  getShape,
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
import type { AgentStatus, BranchChoice, SpanHead } from "@/lib/types"
import { isRecord, reasonText, rows, text } from "@/lib/guards"
import {
  jiraSessionOf,
  noSession,
  readJiraSession,
  type JiraSession,
} from "@/lib/jira-session"
import { repoNames } from "@/lib/repos"
import { Shell } from "@/components/shell/Shell"
import type {
  FoundStory,
  Lookup,
  StoryHit,
} from "@/components/start-run/JiraLookup"
import type { StartSource } from "@/components/start-run/WhatSection"
import { useStartInfo } from "@/hooks/use-start-info"
import { shapeArrival } from "@/lib/start-info"
import { cn } from "@/lib/utils"
import { touchArea } from "@/lib/button-class"

const noBranches: BranchChoice = { branches: [], defaultBase: "dev" }

function reposOf(value: unknown) {
  const names = repoNames(value)
  return names.map((name) => ({ name, checked: names.length === 1 }))
}

function storiesOf(value: unknown): StoryHit[] {
  return rows(isRecord(value) ? value.issues : []).map((item) => ({
    key: text(item.key),
    summary: text(item.summary),
    status: text(item.status),
  }))
}

function storyOf(value: unknown): { found: FoundStory; brief: string } {
  const body = isRecord(value) ? value : {}
  const key = text(body.key)
  const brief = text(body.brief)
  return {
    brief,
    found: {
      key,
      title: text(body.summary) || (brief.split("\n")[0] ?? key),
      status: text(body.status),
      type: text(body.type),
      url: text(body.url),
      acceptance: text(body.acceptance),
    },
  }
}

/** Desk: list runs or start a new one (N1–N4). */
export function Desk() {
  const navigate = useNavigate()
  const location = useLocation()
  const arrival = shapeArrival(location.state)
  const [workspace, setWorkspace] = useState(arrival?.workspace ?? ".")
  const [repos, setRepos] = useState<{ name: string; checked: boolean }[]>(
    arrival !== null && arrival.repo !== ""
      ? [{ name: arrival.repo, checked: true }]
      : []
  )
  const [session, setSession] = useState<JiraSession>(noSession)
  const [stories, setStories] = useState<StoryHit[]>([])
  const [searched, setSearched] = useState("")
  const [issue, setIssue] = useState(arrival?.issue ?? "")
  const [story, setStory] = useState(() =>
    (location.state as { sample?: boolean } | null)?.sample === true
      ? sampleBrief.text
      : (arrival?.story ?? "")
  )
  const [source, setSource] = useState<StartSource>(
    arrival !== null ? "shape" : story === "" ? "jira" : "text"
  )
  const [lookup, setLookup] = useState<Lookup>({ phase: "idle" })
  const [shapeId, setShapeId] = useState(arrival?.id ?? "")
  const [kind, setKind] = useState("infer")
  const [base, setBase] = useState("")
  const [branches, setBranches] = useState<BranchChoice>(noBranches)
  const [spans, setSpans] = useState<SpanHead[]>([])
  const [listed, setListed] = useState(false)
  const [listOk, setListOk] = useState(false)
  const notifications = useNotifications(listOk ? spans : null)
  const formRef = useRef<HTMLFormElement>(null)
  const [starting, setStarting] = useState(arrival !== null)
  const [now, setNow] = useState(() => new Date())
  const [url, setUrl] = useState("")
  const [email, setEmail] = useState("")
  const [token, setToken] = useState("")
  const [jiraNote, setJiraNote] = useState("")
  const [folderNote, setFolderNote] = useState("")
  const [runNote, setRunNote] = useState("")
  const [refusal, setRefusal] = useState<StartRefusal | null>(null)
  const [started, setStarted] = useState<Started | null>(null)
  const [actions, setActions] = useState<TimelineAction[]>([])
  const [agents, setAgents] = useState<AgentStatus[]>([])
  const [agentAt, setAgentAt] = useState("")
  const [listNote, setListNote] = useState("")
  const info = useStartInfo(workspace, kind, base)
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
        setListNote(reasonText(err, "the desk could not load"))
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

  /** A refused token shows its N4 card; any other failure is left to the caller. */
  function refuseToken(err: unknown): boolean {
    if (!(err instanceof UnauthorizedError)) {
      return false
    }
    setRefusal(tokenRefusal(err))
    return true
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
    } catch (err: unknown) {
      setBranches(noBranches)
      setListNote(reasonText(err, "the branches could not be read"))
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

  async function saveSession() {
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
      if (issue.trim() !== "") {
        void loadStory(issue.trim())
      }
    } catch (err: unknown) {
      setJiraNote(reasonText(err, "the session was not saved"))
    }
  }

  async function loadStory(key: string) {
    setLookup({ phase: "asking", key })
    try {
      const body = await getJSON(
        `/jira/issues/${encodeURIComponent(key)}?path=${encodeURIComponent(workspace)}`
      )
      const loaded = storyOf(body)
      setIssue(loaded.found.key)
      setStory(loaded.brief)
      setLookup({ phase: "found", story: loaded.found })
      void readSession(workspace)
    } catch (err: unknown) {
      setLookup({
        phase: "failed",
        key,
        reason: reasonText(err, "the story could not be loaded"),
      })
    }
  }

  async function searchStories(words: string) {
    const jql =
      words === ""
        ? ""
        : `text ~ "${words.replaceAll('"', " ")}" ORDER BY updated DESC`
    try {
      const body = await getJSON(
        `/jira/search?path=${encodeURIComponent(workspace)}&jql=${encodeURIComponent(jql)}`
      )
      setStories(storiesOf(body))
      setSearched(words)
      setLookup({ phase: "idle" })
      void readSession(workspace)
    } catch (err: unknown) {
      setLookup({
        phase: "failed",
        key: issue,
        reason: reasonText(err, "the stories could not be loaded"),
      })
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
    if (source === "shape" && shapeId !== "" && arrival === null) {
      try {
        const sealed = await getShape(shapeId, workspace)
        data.set("story", sealed.story)
        if (sealed.jira !== null) {
          data.set("issue", sealed.jira.key)
        }
      } catch (err: unknown) {
        setRunNote(reasonText(err, "the sealed story could not be read"))
        return
      }
    }
    if (kind === "infer") {
      data.delete("kind")
    }
    if (base.trim() === "") {
      data.delete("base")
    } else {
      data.set("base", base.trim())
    }
    try {
      const pre = await getPrecheck(
        `/start/precheck?${precheckQuery(data).toString()}`
      )
      const blocked = refusalOf(pre)
      if (blocked !== null) {
        setRefusal(blocked)
        return
      }
    } catch (err: unknown) {
      if (refuseToken(err)) {
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
      if (refuseToken(err)) {
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
          onBack={() => {
            setStarted(null)
            setStarting(false)
          }}
          form={{
            formRef,
            where: {
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
            },
            what: {
              source,
              onSource: setSource,
              story,
              onStory: setStory,
              sealed: info.sealed,
              shapeId,
              onShape: setShapeId,
              locked:
                arrival === null
                  ? null
                  : {
                      id: arrival.id,
                      title: arrival.title,
                      workspace: arrival.workspace,
                    },
              sample: listOk && spans.length === 0 ? sampleBrief : null,
              jira: {
                session,
                issue,
                onIssue: setIssue,
                url,
                onUrl: setUrl,
                email,
                onEmail: setEmail,
                token,
                onToken: setToken,
                stories,
                searched,
                lookup,
                note: jiraNote,
                onLoadOne: () => void loadStory(issue),
                onSearch: (words) => void searchStories(words),
                onSaveSession: () => void saveSession(),
                onPick: (key) => {
                  setIssue(key)
                  void loadStory(key)
                },
                onPasteInstead: () => setSource("text"),
              },
            },
            how: {
              kind,
              onKind: setKind,
              base,
              onBase: setBase,
              branches,
              savedBase: info.summary.base,
            },
            aside: {
              summary: info.summary,
              runs: Math.max(repos.filter((repo) => repo.checked).length, 1),
              story,
              shapeable: source === "text",
              onShapeFirst: () =>
                void navigate("/shaping", { state: { idea: story } }),
            },
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
