import { useEffect, useState, type FormEvent } from "react"
import { Link, useNavigate } from "react-router"

import { agentList, basePlaceholder, branchChoice, getJSON, postForm, rememberServeToken, serveToken, startRun, UnauthorizedError, type Started } from "@/lib/api"
import { agentReviewMs, deskAgentLine, timelineLines, withAgent, type TimelineAction } from "@/lib/timeline"
import type { AgentChoice, AgentStatus, BranchChoice, SpanHead } from "@/lib/types"
import { setupSteps, type StepId } from "@/lib/setup"
import { spanHeadsOf } from "@/lib/api"
import { Shell } from "@/components/shell/Shell"
import { AgentPicker } from "@/components/agent-picker/AgentPicker"
import { Setup } from "@/components/setup/Setup"
import { Settings } from "@/components/settings/Settings"
import { StartedRuns } from "@/components/started-runs/StartedRuns"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Textarea } from "@/components/ui/textarea"

interface Repo {
  name: string
  checked: boolean
}

interface JiraSession {
  configured: boolean
  email: string
  baseUrl: string
}

interface StoryHit {
  key: string
  summary: string
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null
}

function reposOf(value: unknown): Repo[] {
  if (!isRecord(value) || !Array.isArray(value.repos)) {
    return []
  }
  const names = value.repos.filter(isRecord).flatMap((item) => (typeof item.name === "string" && item.name !== "" ? [item.name] : []))
  return names.map((name) => ({ name, checked: names.length === 1 }))
}

function reasonText(reason: unknown, fallback: string): string {
  return reason instanceof Error ? reason.message : fallback
}

function sessionOf(value: unknown): JiraSession {
  if (!isRecord(value)) {
    return { configured: false, email: "", baseUrl: "" }
  }
  return {
    configured: value.configured === true,
    email: typeof value.email === "string" ? value.email : "",
    baseUrl: typeof value.baseUrl === "string" ? value.baseUrl : "",
  }
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



/** Every control the desk had, now as the front page. A run opens the span page. */
export function Desk() {
  const navigate = useNavigate()
  const [workspace, setWorkspace] = useState(".")
  const [repos, setRepos] = useState<Repo[]>([])
  const [session, setSession] = useState<JiraSession>({ configured: false, email: "", baseUrl: "" })
  const [stories, setStories] = useState<StoryHit[]>([])
  const [issue, setIssue] = useState("")
  const [story, setStory] = useState("")
  const [kind, setKind] = useState("infer")
  const [base, setBase] = useState("")
  const [branches, setBranches] = useState<BranchChoice>({ branches: [], defaultBase: "dev" })
  const [spans, setSpans] = useState<SpanHead[]>([])
  const [url, setUrl] = useState("")
  const [email, setEmail] = useState("")
  const [token, setToken] = useState("")
  const [jiraNote, setJiraNote] = useState("")
  const [folderNote, setFolderNote] = useState("")
  const [storyNote, setStoryNote] = useState("")
  const [runNote, setRunNote] = useState("")
  const [started, setStarted] = useState<Started | null>(null)
  const [actions, setActions] = useState<TimelineAction[]>([])
  const [agents, setAgents] = useState<AgentStatus[]>([])
  const [agentAt, setAgentAt] = useState("")
  const lines = withAgent(timelineLines({ statements: [], frames: [], actions }), deskAgentLine(agents, agentAt))
  function note(text: string) {
    setActions((current) => [...current, { at: new Date().toISOString(), text }])
  }
  const [listNote, setListNote] = useState("")
  const [serve, setServe] = useState(serveToken)
  const [settingsOpen, setSettingsOpen] = useState(false)
  const [needsToken, setNeedsToken] = useState(false)
  const [pick, setPick] = useState<AgentChoice | null>(null)

  useEffect(() => {
    let closed = false
    void getJSON("/spans")
      .then((spanBody) => {
        if (!closed) {
          setSpans(spanHeadsOf(spanBody))
        }
      })
      .catch((err: unknown) => {
        if (closed) {
          return
        }
        if (err instanceof UnauthorizedError) {
          setNeedsToken(true)
          setSettingsOpen(true)
        }
        setListNote(err instanceof Error ? err.message : "the desk could not load")
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
        .catch(() => undefined)
    }
    pull()
    const timer = window.setInterval(pull, agentReviewMs)
    return () => {
      gone = true
      window.clearInterval(timer)
    }
  }, [])

  // The session belongs to the folder, so it is read when the folder changes, not only
  // when a button is pressed. Without this, a folder that already has a session kept
  // showing the form until someone happened to blur the field.
  async function readSession(project: string) {
    if (project.trim() === "") {
      setSession({ configured: false, email: "", baseUrl: "" })
      return
    }
    try {
      const body = await getJSON(`/jira?path=${encodeURIComponent(project)}`)
      setSession(sessionOf(body))
    } catch {
      // A folder that cannot be read is reported by the action that needed it, not here.
    }
  }

  // The base list belongs to the folder too. A folder that cannot be read keeps the field
  // usable: a typed branch name is still checked by the run before anything is cut.
  async function readBranches(project: string) {
    if (project.trim() === "") {
      setBranches({ branches: [], defaultBase: "dev" })
      return
    }
    try {
      setBranches(branchChoice(await getJSON(`/branches?path=${encodeURIComponent(project)}`)))
    } catch {
      setBranches({ branches: [], defaultBase: "dev" })
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
    // Settled one by one: a Jira outage must not hide the repositories, nor the reverse.
    const [repoResult, jiraResult] = await Promise.allSettled([
      getJSON(`/repos?path=${encodeURIComponent(workspace)}`),
      getJSON(`/jira?path=${encodeURIComponent(workspace)}`),
    ])
    if (repoResult.status === "fulfilled") {
      setRepos(reposOf(repoResult.value))
    } else {
      setFolderNote(reasonText(repoResult.reason, "the folder could not be read"))
    }
    if (jiraResult.status === "fulfilled") {
      setSession(sessionOf(jiraResult.value))
    } else {
      setJiraNote(reasonText(jiraResult.reason, "the session could not be read"))
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
      const next = await getJSON(`/jira?path=${encodeURIComponent(workspace)}`)
      setSession(sessionOf(next))
    } catch (err: unknown) {
      setJiraNote(err instanceof Error ? err.message : "the session was not saved")
    }
  }

  async function closeSession() {
    const data = new URLSearchParams()
    data.set("project", workspace)
    try {
      await postForm("/jira/logout", data)
      setJiraNote("Session closed for this folder.")
      setSession({ configured: false, email: "", baseUrl: "" })
    } catch (err: unknown) {
      setJiraNote(err instanceof Error ? err.message : "the session was not closed")
    }
  }

  async function loadStory() {
    setStoryNote("")
    try {
      const body = await getJSON(`/jira/issues/${encodeURIComponent(issue)}?path=${encodeURIComponent(workspace)}`)
      const loaded = storyOf(body)
      setIssue(loaded.key)
      setStory(loaded.brief)
      void readSession(workspace)
    } catch (err: unknown) {
      setStoryNote(err instanceof Error ? err.message : "the story could not be loaded")
    }
  }

  async function loadMine() {
    setStoryNote("")
    try {
      const body = await getJSON(`/jira/search?path=${encodeURIComponent(workspace)}`)
      setStories(storiesOf(body))
      void readSession(workspace)
    } catch (err: unknown) {
      setStoryNote(err instanceof Error ? err.message : "the stories could not be loaded")
    }
  }

  async function run(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setRunNote("")
    setStarted(null)
    const data = new FormData(event.currentTarget)
    if (kind === "infer") {
      data.delete("kind")
    }
    // An empty base is no choice: the server applies span.yaml's base, else the kind's default.
    // A typed one is sent without the spaces a paste brings; the server refuses anything else.
    if (base.trim() === "") {
      data.delete("base")
    } else {
      data.set("base", base.trim())
    }
    try {
      const result = await startRun(data)
      if (typeof result === "string") {
        setRunNote(result)
        note(result)
        return
      }
      if (result.runs.length > 1 || result.partial) {
        // Several repos started, or some did not: stay here so every run is visible.
        setStarted(result)
        note(result.partial ? "some runs started" : `${result.runs.length} runs started`)
        return
      }
      note("run started")
      void navigate(`/spans/${result.id}`)
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : "the run did not start"
      setRunNote(message)
      note(message)
    }
  }

  const chosenAgent = pick?.agents.find((row) => row.id === pick.chosen)
  const chosenModel = pick !== null && chosenAgent !== undefined ? (pick.models[chosenAgent.id] ?? "") : ""
  const steps = setupSteps({
    folderReady: repos.length > 0 || branches.branches.length > 0,
    folderDetail: repos.length > 0 ? `${repos.length} ${repos.length === 1 ? "repository" : "repositories"} found in ${workspace}` : workspace === "." ? "the current folder" : workspace,
    agentReady: chosenAgent?.ready === true,
    agentDetail: chosenAgent === undefined ? "" : `${chosenAgent.label} · ${chosenModel === "" ? "model set by Gentle AI" : chosenModel}`,
    jiraConfigured: session.configured,
    jiraDetail: `${session.email} at ${session.baseUrl}`,
    jiraProblem: jiraNote,
    runs: spans.length,
  })

  function go(id: StepId) {
    const target = { folder: "workspace", agent: "agent", jira: "jira-url", story: "issue" }[id]
    const element = document.getElementById(target)
    element?.scrollIntoView({ block: "center" })
    element?.focus()
  }

  const jiraBody = session.configured ? (
    <p className="text-pretty text-sm">Connected as {session.email}. You can close the session under settings.</p>
  ) : (
    <form className="flex max-w-md flex-col gap-3" onSubmit={(event) => void saveSession(event)}>
      <p className="text-pretty text-sm text-muted-foreground">
        Span reads your stories with an Atlassian API token.{" "}
        <a className="text-primary underline-offset-4 hover:underline" href="https://id.atlassian.com/manage-profile/security/api-tokens">
          Create one
        </a>
        , then paste it here. It is stored for this folder only.
      </p>
      <div className="flex flex-col gap-2">
        <Label htmlFor="jira-url">Jira URL</Label>
        <Input id="jira-url" placeholder="https://example.atlassian.net" value={url} onChange={(event) => setUrl(event.target.value)} />
      </div>
      <div className="flex flex-col gap-2">
        <Label htmlFor="jira-email">Email</Label>
        <Input id="jira-email" placeholder="you@example.com" value={email} onChange={(event) => setEmail(event.target.value)} />
      </div>
      <div className="flex flex-col gap-2">
        <Label htmlFor="jira-token">Token</Label>
        <Input id="jira-token" type="password" autoComplete="off" value={token} onChange={(event) => setToken(event.target.value)} />
      </div>
      <Button type="submit" className="self-start">
        Connect Jira
      </Button>
      {jiraNote !== "" ? <p className="text-pretty text-sm">{jiraNote}</p> : null}
    </form>
  )

  return (
    <Shell path="~/span" status="desk" lines={lines}>
      <header>
        <h1 className="text-balance text-xl">
          <span className="text-primary">$ </span>span desk
        </h1>
        <p className="text-pretty text-muted-foreground">Turn a Jira story into a reviewed pull request.</p>
      </header>
      <Setup steps={steps} bodies={{ jira: jiraBody }} onGo={go} />
      <form className="flex flex-col gap-4" onSubmit={(event) => void run(event)}>
        <div className="flex flex-col gap-2">
          <Label htmlFor="workspace">Folder</Label>
          <Input
            id="workspace"
            name="workspace"
            value={workspace}
            onChange={(event) => setWorkspace(event.target.value)}
            onBlur={() => {
              void getJSON(`/jira?path=${encodeURIComponent(workspace)}`)
                .then((body) => setSession(sessionOf(body)))
                .catch((err: unknown) => {
                  setJiraNote(err instanceof Error ? err.message : "the session could not be read")
                })
            }}
          />
        </div>
        <div id="agent-field">
          <AgentPicker project={workspace} onLoaded={setPick} />
        </div>
        <div className="flex flex-col gap-2">
          <Button type="button" variant="secondary" className="self-start" onClick={() => void findRepos()}>
            Find repos
          </Button>
          {folderNote !== "" ? <p className="text-pretty text-sm text-destructive">{folderNote}</p> : null}
          {repos.map((repo) => (
            <label key={repo.name} className="flex items-center gap-2 text-sm">
              <Checkbox
                checked={repo.checked}
                onCheckedChange={(checked) =>
                  setRepos((current) => current.map((item) => (item.name === repo.name ? { ...item, checked: checked === true } : item)))
                }
              />
              {repo.name}
              {repo.checked ? <input type="hidden" name="repo" value={repo.name} /> : null}
            </label>
          ))}
        </div>
        <div className="flex flex-col gap-2">
          <Label htmlFor="issue">Story key</Label>
          <Input id="issue" name="issue" value={issue} onChange={(event) => setIssue(event.target.value)} />
          <div className="flex flex-wrap gap-2">
            <Button type="button" variant="secondary" onClick={() => void loadStory()}>
              Load one story
            </Button>
            <Button type="button" variant="secondary" onClick={() => void loadMine()}>
              My stories
            </Button>
            {storyNote !== "" ? <p className="text-pretty text-sm text-destructive">{storyNote}</p> : null}
          </div>
          {stories.length > 0 ? (
            <ul className="flex flex-col gap-1 text-sm">
              {stories.map((item) => (
                <li key={item.key}>
                  <button
                    type="button"
                    className="text-left text-primary underline-offset-4 hover:underline"
                    onClick={() => {
                      setIssue(item.key)
                      void getJSON(`/jira/issues/${encodeURIComponent(item.key)}?path=${encodeURIComponent(workspace)}`)
                        .then((body) => {
                          const loaded = storyOf(body)
                          setStory(loaded.brief)
                        })
                        .catch((err: unknown) => {
                          setStoryNote(err instanceof Error ? err.message : "the story could not be loaded")
                        })
                    }}
                  >
                    {item.key} {item.summary}
                  </button>
                </li>
              ))}
            </ul>
          ) : null}
        </div>
        <div className="flex flex-col gap-2">
          <Label htmlFor="kind">Kind</Label>
          <Select value={kind} onValueChange={setKind}>
            <SelectTrigger id="kind" className="w-40">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="infer">infer</SelectItem>
              <SelectItem value="feat">feat</SelectItem>
              <SelectItem value="fix">fix</SelectItem>
              <SelectItem value="hotfix">hotfix</SelectItem>
            </SelectContent>
          </Select>
          {kind !== "infer" ? <input type="hidden" name="kind" value={kind} /> : null}
        </div>
        <div className="flex flex-col gap-2">
          <Label htmlFor="base">Base branch</Label>
          <Input
            id="base"
            name="base"
            list="base-branches"
            className="w-64"
            placeholder={basePlaceholder(kind, branches.defaultBase)}
            value={base}
            onChange={(event) => setBase(event.target.value)}
          />
          <datalist id="base-branches">
            {branches.branches.map((name) => (
              <option key={name} value={name} />
            ))}
          </datalist>
          <p className="text-pretty text-sm text-muted-foreground">
            The branch is cut from here and its pull request opens into it. Empty uses {basePlaceholder(kind, branches.defaultBase)}.
          </p>
        </div>
        <div className="flex flex-col gap-2">
          <Label htmlFor="story">Story</Label>
          <Textarea id="story" name="story" value={story} onChange={(event) => setStory(event.target.value)} rows={8} />
        </div>
        <div className="flex flex-col gap-2">
          <Label htmlFor="file">Handoff files</Label>
          <Input id="file" name="file" type="file" multiple className="max-w-full" />
        </div>
        <div className="flex items-center gap-3">
          <Button type="submit">Configure and run</Button>
          {runNote !== "" ? <p className="text-pretty text-sm text-destructive">{runNote}</p> : null}
          {started !== null ? (
            <StartedRuns runs={started.runs} partial={started.partial} notes={started.notes} />
          ) : null}
        </div>
      </form>
      <section>
        <h2 className="text-balance text-sm">
          <span className="text-primary">$ </span>spans
        </h2>
        {listNote !== "" ? <p className="text-pretty text-sm text-destructive">{listNote}</p> : null}
        {spans.length === 0 ? (
          <p className="mt-2 text-pretty text-muted-foreground">No runs yet. Load a story above, then run it; it will appear here.</p>
        ) : (
          <ul className="mt-2 flex flex-col gap-1">
            {spans.map((span) => (
              <li key={span.id}>
                <Link
                  className="text-primary underline-offset-4 hover:underline tabular-nums"
                  to={`/spans/${span.id}`}
                  title={span.id}
                >
                  {span.label === "" ? span.id : span.label}
                </Link>
                <span className="text-muted-foreground">{"  "}{span.stage} · {span.verdict}</span>
              </li>
            ))}
          </ul>
        )}
      </section>
      <Settings
        open={settingsOpen}
        onOpenChange={setSettingsOpen}
        token={serve}
        needsToken={needsToken}
        onSaveToken={(value) => {
          setServe(value)
          rememberServeToken(value)
          setNeedsToken(false)
          note(value === "" ? "access token cleared" : "access token saved")
        }}
        jira={session.configured ? `Logged in as ${session.email}. ${session.baseUrl}` : ""}
        onCloseJira={() => void closeSession()}
      />
    </Shell>
  )
}
