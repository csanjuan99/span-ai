"use client"

import {
  ArrowLeftIcon,
  ArrowRightIcon,
  ClockIcon,
  ShieldCheckIcon,
} from "lucide-react"
import { useEffect, useState } from "react"

import { AgentPanel } from "@/components/onboarding/AgentPanel"
import { CheckLine } from "@/components/onboarding/CheckLine"
import { FolderPanel } from "@/components/onboarding/FolderPanel"
import { Panel } from "@/components/onboarding/Frame"
import { JiraPanel } from "@/components/onboarding/JiraPanel"
import { Pill } from "@/components/onboarding/Pill"
import {
  useFolderProbe,
  usable,
} from "@/components/onboarding/use-folder-probe"
import { Button } from "@/components/ui/button"
import { agentChoice, getJSON } from "@/lib/api"
import { jiraSessionOf, noSession, readJiraSession } from "@/lib/jira-session"
import { savedAgo } from "@/lib/onboarding"
import type { AgentState } from "@/lib/types"

/** Step 2: the folder, the agent CLIs and Jira, each checked against the running server. */
export function ConnectStep({
  folder,
  agent,
  savedAt,
  onFolder,
  onAgent,
  onNext,
  onBack,
}: {
  folder: string
  agent: string
  savedAt: number
  onFolder: (value: string) => void
  onAgent: (id: string) => void
  onNext: () => void
  onBack: () => void
}) {
  const [again, setAgain] = useState(0)
  const [agents, setAgents] = useState<AgentState[] | null>(null)
  const [session, setSession] = useState(noSession)
  const [jiraSkipped, setJiraSkipped] = useState(false)
  // Whether the person arrived with a folder from an earlier visit.
  const [returning] = useState(folder.trim() !== "" && savedAt > 0)
  const [now] = useState(() => Date.now())
  const probe = useFolderProbe(folder, again)
  const path = folder.trim()

  useEffect(() => {
    let live = true
    getJSON(`/agent?path=${encodeURIComponent(path)}`)
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
  }, [path, again])

  useEffect(() => {
    if (path === "") {
      return
    }
    let live = true
    readJiraSession(path)
      .then((body) => {
        if (live) {
          setSession(jiraSessionOf(body))
        }
      })
      .catch(() => {
        if (live) {
          setSession(noSession)
        }
      })
    return () => {
      live = false
    }
  }, [path, again])

  const firstReady = (agents ?? []).find((item) => item.ready)?.id ?? ""
  const picked = (agents ?? []).some((item) => item.ready && item.id === agent)
    ? agent
    : firstReady
  const canContinue = usable(probe) && picked !== ""

  return (
    <div className="mx-auto flex w-full max-w-[1020px] flex-col gap-6 sm:gap-8">
      {returning ? (
        <div className="flex flex-wrap items-center gap-2.5 rounded-[14px] border border-border bg-card px-4 py-2.5 shadow-sm">
          <ClockIcon aria-hidden="true" className="size-4 text-primary" />
          <span className="text-[13px]">
            <b>Welcome back.</b> Your folder from last time is already filled
            in.
          </span>
          <span className="ml-auto text-xs text-muted-foreground">
            {savedAgo(savedAt, now)}
          </span>
        </div>
      ) : null}
      <div className="flex flex-col gap-2">
        <Pill tone="acc">Step 2 of 3</Pill>
        <h1 className="text-2xl font-semibold tracking-tight">
          Connect your project
        </h1>
        <p className="text-[13px] text-muted-foreground">
          Each check runs as you type. A problem shows what is wrong and the
          exact fix.
        </p>
      </div>
      <div className="grid items-start gap-6 lg:grid-cols-[minmax(0,1fr)_320px]">
        <div className="flex min-w-0 flex-col gap-6">
          <FolderPanel folder={folder} onFolder={onFolder} probe={probe} />
          <AgentPanel
            agents={agents}
            picked={picked}
            onPick={onAgent}
            onCheck={() => setAgain((value) => value + 1)}
          />
          {jiraSkipped ? (
            <Panel
              title="Jira"
              badge={<Pill tone="mut">Skipped</Pill>}
              lead="You can paste a brief by hand, or connect Jira later from Settings."
            >
              <div>
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => setJiraSkipped(false)}
                >
                  Set up Jira
                </Button>
              </div>
            </Panel>
          ) : (
            <JiraPanel
              key={path}
              folder={path}
              session={session}
              onSaved={() => setAgain((value) => value + 1)}
              onSkip={() => setJiraSkipped(true)}
            />
          )}
          <div className="flex flex-wrap items-center gap-3">
            <Button className="h-10" disabled={!canContinue} onClick={onNext}>
              <ArrowRightIcon aria-hidden="true" />
              Continue
            </Button>
            <Button className="h-10" variant="ghost" onClick={onBack}>
              <ArrowLeftIcon aria-hidden="true" />
              Back
            </Button>
          </div>
          <p className="text-xs text-muted-foreground">
            {canContinue
              ? "Continue is on because a folder and one agent are ready. Jira can wait."
              : "Continue turns on when the folder is a git repository and one agent is ready."}
          </p>
        </div>
        <aside className="flex flex-col gap-4">
          <section className="rounded-[14px] border border-border bg-card px-4 py-4 shadow-sm sm:px-5">
            <div className="mb-2 flex items-center gap-2 text-sm font-semibold">
              <ShieldCheckIcon
                aria-hidden="true"
                className="size-4 text-pass"
              />
              What Span reads and writes
            </div>
            <ul>
              <CheckLine tone="pass">
                <b>Reads</b> files in the folder above.
              </CheckLine>
              <CheckLine tone="pass">
                <b>Writes</b> only on branches named{" "}
                <span className="font-mono text-xs">span/…</span>.
              </CheckLine>
              <CheckLine tone="pass">
                <b>Sends</b> prompts to the agent CLI you choose, and to Jira if
                you connect it.
              </CheckLine>
            </ul>
            <p className="mt-2 text-xs text-muted-foreground">
              Credentials stay in each tool’s own login. A Jira token is kept in
              a file only you can read.
            </p>
          </section>
        </aside>
      </div>
    </div>
  )
}
