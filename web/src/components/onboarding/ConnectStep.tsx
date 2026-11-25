"use client"

import { useEffect, useState } from "react"

import { Button } from "@/components/ui/button"
import { CheckLine } from "@/components/onboarding/CheckLine"
import { Card, StepHead } from "@/components/onboarding/Frame"
import { agentChoice, branchChoice, getJSON } from "@/lib/api"
import { reasonText } from "@/lib/guards"
import { readJiraSession } from "@/lib/jira-session"
import type { AgentState } from "@/lib/types"

type Folder =
  | { state: "idle" }
  | { state: "checking" }
  | { state: "ok"; branches: number; base: string }
  | { state: "bad"; reason: string }

/** Step 2: the folder, the agent CLIs and Jira, each checked against the running server. */
export function ConnectStep({
  folder,
  onFolder,
  onNext,
  onBack,
  onSkip,
}: {
  folder: string
  onFolder: (value: string) => void
  onNext: () => void
  onBack: () => void
  onSkip: () => void
}) {
  const [result, setFound] = useState<Folder>({ state: "idle" })
  const [agents, setAgents] = useState<AgentState[] | null>(null)
  const [jira, setJira] = useState("")
  const [again, setAgain] = useState(0)

  useEffect(() => {
    const path = folder.trim()
    if (path === "") {
      return
    }
    let live = true
    const timer = setTimeout(() => {
      getJSON(`/branches?path=${encodeURIComponent(path)}`)
        .then((body) => {
          const choice = branchChoice(body)
          if (live) {
            setFound({
              state: "ok",
              branches: choice.branches.length,
              base: choice.defaultBase,
            })
          }
        })
        .catch((reason: unknown) => {
          if (live) {
            setFound({
              state: "bad",
              reason: reasonText(reason, "the folder could not be read"),
            })
          }
        })
      readJiraSession(path)
        .then((body) => {
          const session =
            typeof body === "object" && body !== null
              ? (body as Record<string, unknown>)
              : {}
          if (live) {
            setJira(
              session.configured === true
                ? `Connected${typeof session.baseUrl === "string" ? ` to ${session.baseUrl}` : ""}.`
                : ""
            )
          }
        })
        .catch(() => {
          if (live) {
            setJira("")
          }
        })
    }, 250)
    return () => {
      live = false
      clearTimeout(timer)
    }
  }, [folder, again])

  useEffect(() => {
    let live = true
    getJSON(`/agent?path=${encodeURIComponent(folder.trim())}`)
      .then((body) => {
        if (live) {
          setAgents(agentChoice(body).agents)
        }
      })
      .catch(() => {
        if (live) {
          setAgents([])
        }
      })
    return () => {
      live = false
    }
  }, [folder, again])

  const found: Folder = folder.trim() === "" ? { state: "idle" } : result
  const anyReady = (agents ?? []).some((agent) => agent.ready)
  const canContinue = found.state === "ok" && anyReady

  return (
    <div className="flex flex-col gap-6">
      <StepHead
        step="Step 2 of 3"
        title="Connect your project"
        lead="Each check runs as you type. A problem shows what is wrong and the exact fix."
      />
      <Card
        title="Project folder"
        note={found.state === "ok" ? "Ready" : undefined}
      >
        <p className="text-sm text-muted-foreground">
          The folder with your code. Span creates branches and runs your checks
          here.
        </p>
        <input
          aria-label="Project folder"
          value={folder}
          onChange={(event) => onFolder(event.target.value)}
          placeholder="/path/to/your/project"
          className="rounded-lg border border-border bg-background px-3 py-2 font-mono text-sm"
        />
        <ul className="flex flex-col gap-2">
          {found.state === "ok" ? (
            <CheckLine tone="pass">
              Git repository. {found.branches} branches, new runs start from{" "}
              {found.base}.
            </CheckLine>
          ) : null}
          {found.state === "bad" ? (
            <CheckLine
              tone="fail"
              fix={
                <p className="text-sm">
                  Point this at the root of a git repository, then it checks
                  again.
                </p>
              }
            >
              {found.reason}
            </CheckLine>
          ) : null}
        </ul>
      </Card>
      <Card
        title="Agent CLI"
        note={
          agents === null
            ? undefined
            : `${(agents ?? []).filter((agent) => agent.ready).length} ready`
        }
      >
        <p className="text-sm text-muted-foreground">
          Span drives one of these to do the work. At least one must be ready.
        </p>
        <ul className="flex flex-col gap-3">
          {(agents ?? []).map((agent) => (
            <AgentLine key={agent.id} agent={agent} />
          ))}
        </ul>
        <div>
          <Button
            variant="outline"
            onClick={() => setAgain((value) => value + 1)}
          >
            Check again
          </Button>
        </div>
      </Card>
      <Card title="Jira" note="Optional">
        <p className="text-sm text-muted-foreground">
          Lets Span read stories and move them when a run finishes. Skip it and
          paste a brief by hand instead.
        </p>
        <p className="text-sm">
          {jira !== ""
            ? jira
            : "Not connected. You can set it up later in Settings."}
        </p>
      </Card>
      <div className="flex flex-wrap gap-2">
        <Button onClick={onNext} disabled={!canContinue}>
          Continue
        </Button>
        <Button variant="outline" onClick={onBack}>
          Back
        </Button>
        <Button variant="ghost" onClick={onSkip}>
          Skip setup
        </Button>
      </div>
      <p className="text-xs text-muted-foreground">
        {canContinue
          ? "Continue is on because a folder and one agent are ready. Jira can wait."
          : "Continue turns on when the folder is a git repository and one agent is ready."}
      </p>
    </div>
  )
}

function AgentLine({ agent }: { agent: AgentState }) {
  if (agent.ready) {
    return (
      <CheckLine tone="pass">{agent.label} is installed and ready.</CheckLine>
    )
  }
  if (agent.installed) {
    const why =
      agent.reason !== undefined && agent.reason !== ""
        ? `: ${agent.reason}`
        : ""
    return (
      <CheckLine
        tone="hold"
        fix={
          <p className="text-sm">
            Fix what it says in a terminal, then check again.
          </p>
        }
      >
        {agent.label} is installed but not ready{why}
      </CheckLine>
    )
  }
  return (
    <CheckLine
      tone="fail"
      fix={
        <p className="text-sm">
          Install it so {agent.binary} is on your PATH, then check again.
        </p>
      }
    >
      {agent.label} was not found. No {agent.binary} command on your PATH.
    </CheckLine>
  )
}
