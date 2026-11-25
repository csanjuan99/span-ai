"use client"

import { useState } from "react"

import { cn } from "@/lib/utils"
import { ApiCli, type ApiAction } from "@/components/implementation/ApiCli"
import { AuditTab } from "@/components/implementation/AuditTab"
import { CommitsTab } from "@/components/implementation/CommitsTab"
import { DiffTab } from "@/components/implementation/DiffTab"
import type { FileFlags } from "@/components/implementation/FilesTable"
import { OverviewTab } from "@/components/implementation/OverviewTab"
import { ReasoningTab } from "@/components/implementation/ReasoningTab"
import { Audit as AuditTrail } from "@/components/implementation/report/Audit"
import { Evidence } from "@/components/implementation/report/Evidence"
import { RanWithList } from "@/components/implementation/report/RanWith"
import { Risks } from "@/components/implementation/report/Risks"
import { Verdict } from "@/components/implementation/report/Verdict"
import { useRunResource } from "@/hooks/use-run-resource"
import type { JournalState } from "@/hooks/use-events"
import type { Change } from "@/lib/change"
import { totals } from "@/lib/diff"
import type { FeatureReport } from "@/lib/featuredoc"
import { downloadFile } from "@/lib/download"
import { reasonText } from "@/lib/guards"
import {
  checkDecisions,
  markViewed,
  readAudit,
  readCommits,
  readFiles,
  readReasoning,
  type CommitStat,
} from "@/lib/implementation-api"
import type { CommitEntry, Cursor, Statement } from "@/lib/types"

/** What the page knows about the run, for the report beside the diff. */
export interface RunFacts {
  cursor: Cursor
  statements: Statement[]
  agents: Record<string, string>
  feature: FeatureReport | null
  journal: JournalState
}

type View = "overview" | "diff" | "commits" | "reasoning" | "audit"

const views: { value: View; label: string }[] = [
  { value: "overview", label: "Overview" },
  { value: "diff", label: "Diff" },
  { value: "commits", label: "Commits" },
  { value: "reasoning", label: "Reasoning" },
  { value: "audit", label: "Audit" },
]

/** The route and the command of each action of a board, for its API / CLI strip. */
function actionsFor(view: View, id: string): ApiAction[] {
  const route = `/spans/${id}`
  const cmd = `./span implementation ${id}`
  const common: Record<View, ApiAction[]> = {
    overview: [
      {
        name: "Overview",
        method: "GET",
        route: `${route}/implementation`,
        command: `./span runs implementation ${id} --json`,
      },
      {
        name: "Files",
        method: "GET",
        route: `${route}/files`,
        command: `${cmd} --files --json`,
      },
      {
        name: "Mark viewed",
        method: "POST",
        route: `${route}/files/{path}/viewed`,
        command: `${cmd} --viewed <path>`,
      },
    ],
    diff: [
      {
        name: "File diff",
        method: "GET",
        route: `${route}/files/{path}/diff`,
        command: `${cmd} --diff <path>`,
      },
      {
        name: "Mark viewed",
        method: "POST",
        route: `${route}/files/{path}/viewed`,
        command: `${cmd} --viewed <path>`,
      },
    ],
    commits: [
      {
        name: "Commits",
        method: "GET",
        route: `${route}/commits`,
        command: `${cmd} --commits --json`,
      },
      {
        name: "One commit",
        method: "GET",
        route: `${route}/commits/{sha}/diff`,
        command: `${cmd} --commit <sha>`,
      },
    ],
    reasoning: [
      {
        name: "Reasoning",
        method: "GET",
        route: `${route}/reasoning`,
        command: `${cmd} --reasoning --json`,
      },
      {
        name: "Check assumptions",
        method: "POST",
        route: `${route}/decisions/check`,
        command: `${cmd} --check D1 D3`,
      },
    ],
    audit: [
      {
        name: "Audit log",
        method: "GET",
        route: `${route}/audit`,
        command: `${cmd} --audit --json`,
      },
      {
        name: "Verify the chain",
        method: "GET",
        route: `${route}/audit/verify`,
        command: `${cmd} --audit --verify`,
      },
      {
        name: "Export",
        method: "GET",
        route: `${route}/audit?format=json`,
        command: `${cmd} --audit --json > ${id}-audit.json`,
      },
    ],
  }
  return common[view]
}

/** Commits the page already has, shown as a list with no sizes when the server could not size them. */
function plainCommits(log: CommitEntry[]): CommitStat[] {
  return [...log].reverse().map((item) => ({
    ...item,
    parent: "",
    files: 0,
    added: 0,
    removed: 0,
    renameOnly: false,
  }))
}

/** Saves the audit log as a file the person can hand to someone else. */
function download(name: string, body: unknown) {
  downloadFile(name, JSON.stringify(body, null, 2), "application/json")
}

/**
 * The Implementation tab: an overview of the change, the diff file by file, the commits, the
 * reasoning behind the choices, and the audit chain. Each reads what the API serves, and each
 * has a `span implementation` command for the same thing.
 */
export function Implementation({
  change,
  truncated,
  run,
  commits = [],
}: {
  change: Change
  truncated: boolean
  run: RunFacts
  /** The branch's commits, oldest first, as the page already read them. Used while the server's sized list is not there. */
  commits?: CommitEntry[]
}) {
  const id = run.cursor.id
  const [view, setView] = useState<View>("overview")
  const [current, setCurrent] = useState("")
  const [marks, setMarks] = useState<Record<string, boolean>>({})
  const [note, setNote] = useState("")
  const [busy, setBusy] = useState(false)
  const filesRes = useRunResource(id, readFiles)
  const commitsRes = useRunResource(id, readCommits)
  const reasoningRes = useRunResource(id, readReasoning)
  const auditRes = useRunResource(id, readAudit)
  const { files, report } = change
  const sum = totals(files)

  const flags: Record<string, FileFlags> = {}
  if (filesRes.state.status === "ready") {
    for (const entry of filesRes.state.value) {
      flags[entry.path] = {
        viewed: entry.viewed,
        generated: entry.generated,
        large: entry.large,
        binary: entry.binary,
      }
    }
  }
  for (const [path, viewed] of Object.entries(marks)) {
    const prior = flags[path] ?? {
      viewed: false,
      generated: false,
      large: false,
      binary: false,
    }
    flags[path] = { ...prior, viewed }
  }

  async function setViewed(path: string, viewed: boolean) {
    setMarks((now) => ({ ...now, [path]: viewed }))
    setNote("")
    try {
      await markViewed(id, path, viewed)
    } catch (err: unknown) {
      setMarks((now) => ({ ...now, [path]: !viewed }))
      setNote(reasonText(err, "the viewed mark was not saved"))
    }
  }

  async function check(ids: string[]) {
    setBusy(true)
    setNote("")
    try {
      reasoningRes.set(await checkDecisions(id, ids))
    } catch (err: unknown) {
      setNote(reasonText(err, "the decisions could not be marked"))
    } finally {
      setBusy(false)
    }
  }

  async function exportAudit() {
    try {
      download(`${id}-audit.json`, await readAudit(id))
    } catch (err: unknown) {
      setNote(reasonText(err, "the audit log could not be exported"))
    }
  }

  function open(path: string) {
    setCurrent(path)
    setView("diff")
  }

  const sized =
    commitsRes.state.status === "ready" && commitsRes.state.value.length > 0
      ? commitsRes.state.value
      : plainCommits(commits)
  const assessment = (
    <>
      <Verdict report={report} />
      <RanWithList events={run.journal.events} />
      <Risks risks={report.risks} reported={run.feature?.risks ?? []} />
      <Evidence evidence={report.evidence} gaps={report.gaps} />
    </>
  )
  return (
    <div className="flex flex-col gap-4">
      <div
        role="tablist"
        aria-label="Implementation sections"
        className="flex flex-wrap gap-2"
      >
        {views.map((item) => (
          <button
            key={item.value}
            type="button"
            role="tab"
            aria-selected={view === item.value}
            className={cn(
              "inline-flex h-8 items-center rounded-full border border-input bg-card px-3 text-[13px] focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none max-sm:h-11",
              view === item.value &&
                "border-primary bg-accent font-semibold text-primary"
            )}
            onClick={() => setView(item.value)}
          >
            {item.label}
          </button>
        ))}
      </div>
      {note !== "" && view !== "reasoning" && view !== "audit" ? (
        <p role="alert" className="text-sm text-destructive">
          {note}
        </p>
      ) : null}
      {view === "overview" ? (
        <OverviewTab
          change={change}
          flags={flags}
          cursor={run.cursor}
          commits={sized}
          commitCount={sized.length > 0 ? sized.length : commits.length}
          onOpen={open}
          onViewed={(path, viewed) => void setViewed(path, viewed)}
          assessment={assessment}
        />
      ) : null}
      {view === "diff" ? (
        <DiffTab
          runId={id}
          files={files}
          flags={flags}
          current={current === "" ? (files[0]?.path ?? "") : current}
          onSelect={setCurrent}
          onViewed={(path, viewed) => void setViewed(path, viewed)}
          decisions={
            reasoningRes.state.status === "ready"
              ? reasoningRes.state.value.decisions
              : []
          }
        />
      ) : null}
      {view === "commits" ? (
        <CommitsTab
          runId={id}
          commits={sized}
          base={run.cursor.base}
          branch={run.cursor.branch}
          fileCount={sum.files}
          added={sum.added}
          removed={sum.removed}
          onOpenBranchDiff={() => setView("overview")}
          note={
            commitsRes.state.status === "error" && sized.length === 0
              ? commitsRes.state.message
              : ""
          }
        />
      ) : null}
      {view === "reasoning" ? (
        <ReasoningTab
          view={
            reasoningRes.state.status === "ready"
              ? reasoningRes.state.value
              : null
          }
          note={
            note !== ""
              ? note
              : reasoningRes.state.status === "error"
                ? reasoningRes.state.message
                : ""
          }
          busy={busy}
          onCheck={(ids) => void check(ids)}
          transcriptLines={0}
        />
      ) : null}
      {view === "audit" ? (
        <>
          <AuditTab
            audit={
              auditRes.state.status === "ready" ? auditRes.state.value : null
            }
            note={
              note !== ""
                ? note
                : auditRes.state.status === "error"
                  ? auditRes.state.message
                  : ""
            }
            onExport={() => void exportAudit()}
          />
          <AuditTrail
            cursor={run.cursor}
            statements={run.statements}
            agents={run.agents}
            journal={run.journal}
          />
        </>
      ) : null}
      {truncated && (view === "overview" || view === "diff") ? (
        <p className="text-sm text-hold">
          The patch was cut at 256 KiB. Files past the cut show their counts
          only.
        </p>
      ) : null}
      <ApiCli actions={actionsFor(view, id)} />
    </div>
  )
}
