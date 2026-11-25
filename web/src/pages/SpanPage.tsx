"use client"

import { useMemo, useState, type FormEvent } from "react"
import { Link, useParams } from "react-router"
import { MessageSquare, RotateCcw, Undo2 } from "lucide-react"

import {
  answerRun,
  approveStage,
  decideRun,
  pauseRun,
  readStory,
  relaunchRun,
  resumeRun,
  saveInstruction,
  stopRun,
} from "@/lib/api"
import { agentBlockLine } from "@/lib/agent-block"
import { conflictHold, proposalKey } from "@/lib/conflict"
import {
  activeFailure,
  approvalLede,
  approvalOf,
  attemptLine,
  checksBefore,
  holdAction,
  holderOf,
  isCheckoutBusy,
} from "@/lib/failure"
import { changeOf } from "@/lib/change"
import { keptWork } from "@/lib/kept"
import { doneOf } from "@/lib/done"
import { proofOf } from "@/lib/proof"
import { runDetails } from "@/lib/run-details"
import { runningOf } from "@/lib/running"
import { needsYou } from "@/lib/needs-you"
import { instructionTiming, noteStage } from "@/lib/stage-copy"
import { clockOf } from "@/lib/timeline"
import { waitingQuestions } from "@/lib/waiting"
import { useSpan } from "@/hooks/use-span"
import { ConflictPanel } from "@/components/conflict/ConflictPanel"
import { FailureCard } from "@/components/failure/FailureCard"
import { Implementation } from "@/components/implementation/Implementation"
import { useEvents } from "@/hooks/use-events"
import { useFeature } from "@/hooks/use-feature"
import { InstructionNote } from "@/components/instruction-note/InstructionNote"
import { RawOutput } from "@/components/raw-output/RawOutput"
import { AgentBlocked } from "@/components/problem/AgentBlocked"
import { ConnectionBanner } from "@/components/problem/ConnectionBanner"
import { ProblemCard } from "@/components/problem/ProblemCard"
import { PageProblem } from "@/components/problem/PageProblem"
import { RunSkeleton } from "@/components/problem/RunSkeleton"
import { useConnectionNotice } from "@/hooks/use-connection-notice"
import { Ledger } from "@/components/ledger/Ledger"
import { LedgerSummary } from "@/components/ledger/LedgerSummary"
import { NeedsYou } from "@/components/needs-you/NeedsYou"
import { ProofSummary } from "@/components/proof-summary/ProofSummary"
import { QACases } from "@/components/qa-cases/QACases"
import { QaTab } from "@/components/qa/QaTab"
import { Rewind } from "@/components/rewind/Rewind"
import { FilesChanged } from "@/components/files-changed/FilesChanged"
import { FinishedRecord } from "@/components/finished-record/FinishedRecord"
import { DoneCard } from "@/components/done-card/DoneCard"
import { RunningCard } from "@/components/running-card/RunningCard"
import { RunControls } from "@/components/run-controls/RunControls"
import { SafeAndKept } from "@/components/safe-and-kept/SafeAndKept"
import { StageAgent } from "@/components/stage-agent/StageAgent"
import { restartStage } from "@/lib/stage-agents"
import { UnsealedCard } from "@/components/unsealed-card/UnsealedCard"
import { unsealedBuild } from "@/lib/unsealed"
import { useRunCommands } from "@/hooks/use-command-palette"
import { runContextOf, type RunActions } from "@/lib/run-commands"
import { summaryOf, toMarkdown } from "@/lib/summary"
import { emptyProof, type RunPosition } from "@/lib/proof-copy"
import { Shell } from "@/components/shell/Shell"
import { StageRail } from "@/components/stage-rail/StageRail"
import { RunHeader } from "@/components/run-header/RunHeader"
import { Consequences } from "@/components/status-block/Consequences"
import { Lede } from "@/components/status-block/Lede"
import { metaTime, type RunState } from "@/lib/run-header"
import { stages } from "@/lib/types"
import { stageDurations } from "@/lib/stage-times"
import { Decision } from "@/components/decision/Decision"
import { Waiting } from "@/components/waiting/Waiting"
import { Button, buttonVariants } from "@/components/ui/button"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { wideButton } from "@/lib/button-class"
import { cn } from "@/lib/utils"

/** Three readings of one run. The socket wakes the page. The statements stay the record. */
const tabs = [
  { value: "ledger", label: "Ledger" },
  { value: "implementation", label: "Implementation" },
  { value: "qa", label: "QA" },
] as const

const tabClass =
  "h-9 rounded-[10px] text-sm data-active:bg-card data-active:font-semibold data-active:text-primary dark:data-active:border-transparent dark:data-active:bg-card dark:data-active:text-primary"

export function SpanPage() {
  const params = useParams()
  const id = params.id ?? ""
  const state = useSpan(id)
  const [showNote, setShowNote] = useState(false)
  // A stage the failure card asked to restart at; the controls open their restart form on it.
  const [restartAt, setRestartAt] = useState("")
  const [tab, setTab] = useState("ledger")
  const [body, setBody] = useState("")
  const [note, setNote] = useState("")
  const cursor = state.span?.cursor
  const featureState = useFeature(id, cursor?.gitSHA ?? "")
  const journalState = useEvents(id, state.span?.statements.length ?? 0)
  const current = cursor?.stage || "intake"
  const paused = Boolean(cursor?.hold || cursor?.stop)
  // The stage a note targets follows the run until the person touches the form; touching it
  // pins the stage they were looking at, so it does not move under them.
  const [stage, setStage] = useState("")
  const selected =
    stage ||
    noteStage({
      current,
      hold: Boolean(cursor?.hold),
      stop: Boolean(cursor?.stop),
    }) ||
    noteStage({ current, hold: paused, stop: Boolean(cursor?.stop) })
  const verdict: RunState = cursor?.done
    ? "done"
    : cursor?.stop
      ? "stop"
      : cursor?.hold
        ? "hold"
        : "running"
  const sealed = (state.span?.statements ?? [])
    .filter((item) => item.verdict === "advance")
    .map((item) => item.stage)
  const socket = state.connected ? "socket open" : "socket closed"
  const notice = useConnectionNotice(state.connected)
  const lastSeen = clockOf(state.frames.at(-1)?.at ?? state.agentAt)
  const questions = waitingQuestions(cursor ?? null)
  const waiting = questions.length > 0
  const agentBlock = agentBlockLine(cursor?.agentBlock, {
    stage: cursor?.stage ?? "",
    sealed,
  })
  const failure = activeFailure({
    statements: state.span?.statements ?? [],
    cursor: cursor ?? null,
  })
  // A run whose process died is not held and not done, so the cursor alone cannot tell
  // the page that there is nothing to do. The agent poll can.
  const idle = state.agent?.summary === "not running"
  const busy = isCheckoutBusy(cursor?.reason ?? "")
  const holder = holderOf(cursor?.reason ?? "")
  // A conflict hold is answered on the conflict cards, not by fixing a failed check.
  const conflictKind = conflictHold(cursor?.reason ?? "")
  const conflict = cursor?.conflict
  const action = holdAction({
    failure,
    questions: questions.length,
    stop: Boolean(cursor?.stop),
    done: Boolean(cursor?.done),
    idle,
    busy,
    conflict: conflictKind,
  })
  const canAct = Boolean(cursor?.hold) || idle
  const approval = waiting ? null : approvalOf(cursor ?? null)
  // The failing statement stays on disk through a busy checkout, an approval and a conflict,
  // but none of those wait on the fix, so the failed check only shows when nothing else does.
  const showFailure =
    failure !== null &&
    agentBlock === null &&
    !busy &&
    approval === null &&
    conflictKind === null
  const needs = needsYou({
    agentBlocked: agentBlock !== null,
    questions: waiting,
    decision: cursor?.decision !== undefined,
    conflict: conflict !== undefined || conflictKind !== null,
    approval: approval !== null,
    busy,
    failed: showFailure,
    stop: Boolean(cursor?.stop),
    idle,
    hold: Boolean(cursor?.hold),
    done: Boolean(cursor?.done),
  })
  // While the agent works and nothing asks for a person, the page says so instead of staying blank.
  const running =
    needs === null &&
    cursor !== undefined &&
    !cursor.done &&
    state.agent?.running === true
      ? runningOf({
          stage: cursor.stage,
          agent: state.agent.name,
          branch: cursor.branch,
          workspace: cursor.workspace,
          transcript: state.transcript,
        })
      : null
  const done =
    cursor?.done === true
      ? doneOf({
          sealed: new Set(sealed).size,
          merged: sealed.includes("merge"),
          sha: cursor.gitSHA,
          base: cursor.base,
          branch: cursor.branch,
          issue: cursor.issue,
          changeURL: cursor.changeURL,
        })
      : null
  // The ledger is the only clock the page has: a hold began when the last statement was written.
  const lastStamp =
    [...(state.span?.statements ?? [])].reverse().find((item) => item.time)
      ?.time ?? ""
  const attempt = failure === null ? "" : attemptLine(failure)
  const stageNumber =
    Math.max(stages.indexOf(current as (typeof stages)[number]), 0) + 1
  const failedWhere = [
    current,
    attempt,
    failure?.command ? `exit ${failure.command.exitCode}` : "",
  ]
    .filter((part) => part !== "")
    .join(" · ")
  const where =
    needs?.kind === "failed" && failedWhere !== current
      ? failedWhere
      : needs?.kind === "questions"
        ? `${current} · ${questions.length} open`
        : needs?.kind === "busy"
          ? `${current} · checkout in use`
          : needs?.kind === "agent-blocked"
            ? `${current} · ${cursor?.agentBlock?.agent ?? ""}`.replace(
                / · $/,
                ""
              )
            : `${current} · stage ${stageNumber} of ${stages.length}`
  const bandTitle =
    needs === null
      ? ""
      : needs.kind === "questions"
        ? `Waiting · ${questions.length} ${questions.length === 1 ? "question" : "questions"}`
        : needs.kind === "decision"
          ? "Decision · 1 open"
          : needs.kind === "conflict" && cursor?.conflict !== undefined
            ? `Conflict · ${cursor.conflict.groups.length} ${cursor.conflict.groups.length === 1 ? "group" : "groups"}`
            : needs.label
  const earlierChecks =
    needs?.kind === "approval" && approval !== null
      ? checksBefore(state.span?.statements ?? [], approval.stage)
      : []
  const passedChecks = earlierChecks.filter((command) => command.exitCode === 0)
  // Where nothing else explains the hold, say it in a heading and a line.
  const lede =
    needs?.kind === "approval" && approval !== null
      ? {
          title:
            approval.stage === "open"
              ? "Approve open to publish this change"
              : `Approve ${approval.stage} to ${approval.stage === "merge" ? "merge this change" : "go on"}`,
          body: approvalLede(approval.stage, sealed),
        }
      : needs?.kind === "busy"
        ? {
            title: "Another run is editing this checkout",
            body: "This run has not touched the checkout, so nothing is lost. Let the other run finish or stop it, then resume this one.",
          }
        : needs?.kind === "held" || needs?.kind === "idle"
          ? {
              title:
                needs.kind === "idle"
                  ? "Nothing is running this span"
                  : "This run is paused",
              body: cursor?.reason ?? "",
            }
          : null
  const headState: RunState =
    needs?.kind === "questions" || needs?.kind === "decision"
      ? "waiting"
      : verdict

  /**
   * Runs one action the page offers and says what happened. Every action reports its failure
   * the same way, so the person always reads either the confirmation or the reason.
   */
  async function act(
    request: () => Promise<unknown>,
    outcome: {
      log: string
      say: string | ((result: unknown) => string)
      failed: string
    }
  ): Promise<boolean> {
    setNote("")
    try {
      const result = await request()
      if (outcome.log !== "") {
        state.note(outcome.log)
      }
      setNote(
        typeof outcome.say === "function" ? outcome.say(result) : outcome.say
      )
      return true
    } catch (err: unknown) {
      setNote(err instanceof Error ? err.message : outcome.failed)
      return false
    }
  }

  const asked = (what: string) => (why: unknown) =>
    typeof why === "string" && why !== "" ? why : `${what} requested.`

  const saveNote = () =>
    act(() => saveInstruction(id, selected, body), {
      log: "",
      say: "Instruction saved.",
      failed: "the instruction was not saved",
    })

  async function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    await saveNote()
  }

  // The note is read when the stage starts again, so it is saved first and only then does the run go on.
  async function saveAndResume() {
    if (await saveNote()) {
      await resume()
    }
  }

  const answer = (answers: { question: string; body: string }[]) =>
    act(() => answerRun(id, answers), {
      log: "answer saved",
      say: "Answer saved. Frame is reading the story again.",
      failed: "the answer was not saved",
    })

  async function decide(option: string, text: string) {
    const open = cursor?.decision
    if (open === undefined) {
      return
    }
    await act(() => decideRun(id, open.id, option, text), {
      log: "decision saved",
      say: "Decision saved. The run is going on.",
      failed: "the decision was not saved",
    })
  }

  const reloadStory = () =>
    act(() => readStory(id), {
      log: "reading the story again",
      say: "Reading the story again.",
      failed: "the story was not read",
    })

  const resume = () =>
    act(() => resumeRun(id), {
      log: "resume",
      say: "The run is continuing.",
      failed: "the run did not continue",
    })

  const openNote = () => {
    setStage(selected)
    setShowNote(true)
  }

  async function approve() {
    if (approval === null) {
      return
    }
    await act(() => approveStage(id, approval.stage, approval.environment), {
      log: `approved ${approval.stage}`,
      say: "Approved. The run is continuing.",
      failed: "the approval was not saved",
    })
  }

  async function copySummary() {
    if (state.span === null) {
      return
    }
    const markdown = toMarkdown(
      summaryOf({
        cursor: state.span.cursor,
        statements: state.span.statements,
        implementation: state.implementation,
        label: state.span.label ?? "",
        title:
          state.span.title !== undefined && state.span.title !== ""
            ? state.span.title
            : undefined,
      })
    )
    try {
      await navigator.clipboard.writeText(markdown)
      setNote("Summary copied.")
    } catch (err: unknown) {
      setNote(err instanceof Error ? err.message : "the summary was not copied")
    }
  }

  const pause = () =>
    act(() => pauseRun(id), {
      log: "pause",
      say: asked("Pause"),
      failed: "the pause was not requested",
    })

  const stop = () =>
    act(() => stopRun(id), {
      log: "stop",
      say: asked("Stop"),
      failed: "the run was not stopped",
    })

  const relaunch = () =>
    act(() => relaunchRun(id), {
      log: "relaunch",
      say: "The run is starting again.",
      failed: "the run did not start",
    })

  // Reading the whole patch is the costly part, and the page renders again on every transcript
  // frame, so the change is rebuilt only when the diff, the statements or the feature change.
  const change = useMemo(
    () =>
      state.span === null
        ? null
        : changeOf({
            stat: state.implementation?.stat ?? "",
            patch: state.implementation?.patch ?? "",
            truncated: state.implementation?.truncated ?? false,
            statements: state.span.statements,
            cursor: state.span.cursor,
            feature: featureState.feature,
          }),
    [state.implementation, state.span, featureState.feature]
  )
  // Before the diff is known the panel still says where the run stands, with nothing measured.
  const proof =
    change === null || state.implementation === null
      ? emptyProof()
      : proofOf(change.report, change.files)
  const kept = keptWork({
    statements: state.span?.statements ?? [],
    branch: cursor?.branch ?? "",
    head: cursor?.gitSHA ?? "",
    stat: state.implementation?.stat ?? "",
    commits: state.implementation?.commits ?? 0,
  })
  const keptText =
    kept === null
      ? ""
      : `${kept.sealed} ${kept.sealed === 1 ? "stage" : "stages"} sealed${kept.commits > 0 ? ` · ${kept.commits} ${kept.commits === 1 ? "commit" : "commits"}` : ""}`
  const neverSealed = unsealedBuild(
    Boolean(cursor?.done),
    state.span?.statements ?? []
  )
  const position: RunPosition = {
    kind: cursor?.done
      ? neverSealed
        ? "unsealed"
        : "done"
      : needs?.kind === "agent-blocked"
        ? "blocked"
        : needs?.kind === "busy"
          ? "busy"
          : needs?.kind === "approval"
            ? "approval"
            : needs?.kind === "questions" || needs?.kind === "decision"
              ? "paused"
              : paused
                ? "held"
                : "running",
    stage: current,
    sealed: kept?.sealed ?? 0,
  }
  // The engine refuses to restart a stage while the checkout is in the middle of a merge.
  const restartBlocked =
    conflict !== undefined || conflictKind !== null
      ? "Unavailable while a merge conflict is open. Abort the merge first."
      : ""

  // The right column of the block that waits on a person: what the action does, then the action.
  const next = stages[stages.indexOf(current as (typeof stages)[number]) + 1]
  const primary =
    needs === null
      ? null
      : approval !== null
        ? {
            label: approval.label,
            hint: approval.hint,
            run: () => void approve(),
          }
        : !waiting &&
            cursor?.decision === undefined &&
            canAct &&
            action !== null
          ? { label: action.label, hint: action.hint, run: () => void resume() }
          : null
  const asideTitle =
    needs?.kind === "failed"
      ? "If you fix and retry"
      : needs?.kind === "approval"
        ? "If you approve"
        : needs?.kind === "busy"
          ? "If you resume when it is free"
          : "If you resume"
  const whereLine = [state.span?.repo, cursor?.branch]
    .filter(Boolean)
    .join(" · ")
  const stageWord =
    current === ""
      ? "This stage"
      : current.charAt(0).toUpperCase() + current.slice(1)
  const changes =
    needs?.kind === "agent-blocked"
      ? `Nothing in the code changes. ${stageWord} starts again with the same instructions.`
      : needs?.kind === "busy"
        ? `Nothing yet. ${stageWord} starts from the last sealed stage.`
        : needs?.kind === "approval" && cursor?.stage === "open"
          ? `Opens a change from ${cursor.branch} into ${cursor.base}. Nothing is merged yet.`
          : needs?.kind === "approval" && cursor?.stage === "merge"
            ? `Merges ${cursor.branch} into ${cursor.base}.`
            : (primary?.hint ?? "")
  const nextRows =
    needs?.kind === "failed" && next !== undefined
      ? [
          {
            label: "Next",
            body: `${stageWord} runs again. If it passes, the run moves to ${next}.`,
          },
        ]
      : needs?.kind === "busy"
        ? [
            {
              label: "Next",
              body: `${stageWord} starts as soon as the checkout is free.`,
            },
          ]
        : needs?.kind === "agent-blocked"
          ? [
              {
                label: "Next",
                body: `${stageWord} runs with the agent you choose in Run controls.`,
              },
            ]
          : needs?.kind === "approval" && next !== undefined
            ? [{ label: "Next", body: `The run moves to ${next}.` }]
            : []
  // A decision or a conflict draws its own right column, so the band gets none beside it.
  const ownsAside = cursor?.decision !== undefined || conflict !== undefined
  const aside =
    primary === null || state.span === null || ownsAside ? undefined : (
      <>
        <Consequences
          title={asideTitle}
          icon={RotateCcw}
          rows={[
            { label: "Changes", body: changes },
            {
              label: "Where",
              body: (
                <span className="font-mono text-xs break-all">{whereLine}</span>
              ),
            },
            ...(kept !== null
              ? [
                  {
                    label: "Kept",
                    body: keptText,
                  },
                ]
              : []),
            ...nextRows,
          ]}
        />
        <Button type="button" className={wideButton} onClick={primary.run}>
          {primary.label}
        </Button>
        {holder !== "" ? (
          <div className="flex flex-col gap-1">
            <Link
              className="self-start text-sm text-primary underline-offset-4 hover:underline"
              to={`/spans/${holder}`}
            >
              Open the other run
            </Link>
            <p className="text-xs text-pretty text-muted-foreground">
              You can let it finish or stop it from there.
            </p>
          </div>
        ) : null}
        {state.agent?.relaunch ? (
          <Button
            type="button"
            variant="outline"
            className={wideButton}
            onClick={() => void relaunch()}
          >
            Relaunch
          </Button>
        ) : null}
        {!cursor?.stop && !waiting && cursor?.decision === undefined ? (
          <div className="flex flex-col gap-3 pt-1">
            <span className="text-xs font-medium tracking-wide text-muted-foreground">
              Other ways out
            </span>
            <div className="flex flex-col gap-1.5">
              <Button
                type="button"
                variant="outline"
                className={wideButton}
                onClick={openNote}
              >
                <MessageSquare aria-hidden="true" />
                Leave a note for {selected}
              </Button>
              <p className="text-xs text-pretty text-muted-foreground">
                Save an instruction for {selected}, then resume. Resume reads it
                when this stage starts again.
              </p>
            </div>
            {(needs?.kind === "failed" ||
              needs?.kind === "agent-blocked" ||
              needs?.kind === "approval") &&
            restartBlocked === "" ? (
              <div className="flex flex-col gap-1.5">
                <Button
                  type="button"
                  variant="outline"
                  className={cn(
                    wideButton,
                    "border-fail text-fail hover:bg-fail/10 hover:text-fail dark:border-fail dark:hover:bg-fail/10"
                  )}
                  onClick={() => setRestartAt(restartStage(current))}
                >
                  <Undo2 aria-hidden="true" />
                  {needs?.kind === "failed"
                    ? `Restart from ${restartStage(current)}…`
                    : "Restart from a stage…"}
                </Button>
                <p className="text-xs text-pretty text-muted-foreground">
                  The attempt so far stays on disk as its own iteration. The
                  repair budget starts fresh.
                </p>
              </div>
            ) : null}
          </div>
        ) : null}
      </>
    )

  // The palette lists what this run can take now, so a key does what its button does. The
  // context is rebuilt each render on purpose: its handlers close over this render's state.
  const span = state.span
  const controls: RunActions =
    span === null || cursor?.done
      ? {}
      : {
          ...(canAct && !waiting ? { resume: () => void resume() } : {}),
          ...(!waiting && !cursor?.stop && cursor?.decision === undefined
            ? {
                note: openNote,
              }
            : {}),
          ...(!paused && !idle && running === null
            ? { pause: () => void pause() }
            : {}),
          ...(!paused && !idle ? { stop: () => void stop() } : {}),
          ...(restartBlocked === ""
            ? { rewind: () => setRestartAt(restartStage(current)) }
            : {}),
        }
  const reading: RunActions =
    span === null
      ? {}
      : {
          transcript: () => {
            setTab("ledger")
            window.requestAnimationFrame(() =>
              document
                .querySelector('[aria-label="Raw output"]')
                ?.scrollIntoView({ behavior: "smooth" })
            )
          },
          diff: () => setTab("implementation"),
          copySummary: () => void copySummary(),
        }
  useRunCommands(
    span === null
      ? null
      : runContextOf(span.label || id, { ...controls, ...reading })
  )

  if (state.span === null) {
    return (
      <Shell path={`~/spans/${id}`} status={state.problem ?? "loading"}>
        {state.problem !== null ? (
          <PageProblem problem={state.problem} id={id} />
        ) : (
          <RunSkeleton />
        )}
      </Shell>
    )
  }

  return (
    <Shell
      path={`~/spans/${id}`}
      status={`${cursor?.stage || "intake"} · ${verdict} · ${socket}`}
    >
      <div className="mx-auto flex w-full max-w-[1200px] flex-col gap-6">
        <RunHeader
          chip={cursor?.issue || id}
          state={headState}
          conflict={needs?.kind === "conflict"}
          title={state.span?.title || state.span?.label || id}
          repo={state.span?.repo ?? ""}
          branch={cursor?.branch ?? ""}
          time={metaTime({
            state: headState,
            startedAt: cursor?.startedAt ?? "",
            lastStamp,
            now: new Date(),
          })}
          summaryTo={`/spans/${id}/summary`}
        />
        <StageRail
          stage={cursor?.stage ?? ""}
          verdict={headState === "waiting" ? "waiting" : verdict}
          sealed={sealed}
          durations={stageDurations(state.span?.statements ?? [])}
        />
        {note !== "" ? (
          <p className="text-sm text-pretty text-hold">{note}</p>
        ) : null}
        {state.error !== "" ? (
          <p className="text-sm text-pretty text-destructive">{state.error}</p>
        ) : null}
        <ConnectionBanner notice={notice} lastSeen={lastSeen} />
        {/*
        One column on a narrow screen, in the order the design draws it: what needs the person, the
        proof, what is safe, the tabs, then the controls. From lg up the first and last pairs split
        into a main column and a side column; `contents` lets the order classes reach the grid.
      */}
        <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_300px] lg:items-start">
          <div className="contents lg:flex lg:min-w-0 lg:flex-col lg:gap-6">
            <div className="order-1 flex min-w-0 flex-col gap-6 lg:order-none">
              {needs !== null ? (
                <NeedsYou
                  needs={needs}
                  title={bandTitle}
                  where={where}
                  aside={aside}
                >
                  {agentBlock !== null ? (
                    <AgentBlocked line={agentBlock} />
                  ) : null}
                  {lede !== null ? (
                    <Lede title={lede.title} body={lede.body} />
                  ) : null}
                  {passedChecks.length > 0 ? (
                    <div className="flex flex-col gap-2">
                      <div className="flex flex-wrap items-baseline gap-2">
                        <h3 className="text-sm leading-5 font-semibold">
                          Checks that passed on this commit
                        </h3>
                        <span className="text-xs text-muted-foreground">
                          {passedChecks.length} of {earlierChecks.length}
                        </span>
                      </div>
                      <ul className="flex flex-col gap-1">
                        {passedChecks.map((command) => (
                          <li
                            key={command.command}
                            className="flex items-baseline justify-between gap-3 font-mono text-xs"
                          >
                            <span className="min-w-0 truncate">
                              $ {command.command}
                            </span>
                            <span className="shrink-0 text-muted-foreground">
                              exit {command.exitCode}
                            </span>
                          </li>
                        ))}
                      </ul>
                    </div>
                  ) : null}
                  {showFailure ? (
                    <FailureCard id={id} failure={failure} />
                  ) : null}
                  {conflict !== undefined ? (
                    <ConflictPanel
                      key={proposalKey(conflict)}
                      id={id}
                      conflict={conflict}
                      where={whereLine}
                      kept={keptText}
                    />
                  ) : null}
                  {conflict === undefined && conflictKind === "choose" ? (
                    <p className="text-sm text-pretty text-muted-foreground">
                      The options for this conflict are on their way.
                    </p>
                  ) : null}
                  {waiting ? (
                    <Waiting
                      key={questions.join("\n")}
                      questions={questions}
                      note={note}
                      onAnswer={answer}
                      onReload={reloadStory}
                    />
                  ) : null}
                  {cursor?.decision !== undefined && !waiting ? (
                    <Decision
                      key={cursor.decision.id}
                      request={cursor.decision}
                      where={whereLine}
                      kept={keptText}
                      note={note}
                      onDecide={decide}
                    />
                  ) : null}
                  {cursor?.stop && !waiting ? (
                    <p className="text-sm text-pretty">
                      This run will not continue.
                    </p>
                  ) : null}
                </NeedsYou>
              ) : null}
              {done !== null ? <DoneCard done={done} /> : null}
              {running !== null ? (
                <RunningCard running={running} onPause={() => void pause()} />
              ) : null}
              {neverSealed ? (
                <UnsealedCard
                  sealed={kept?.sealed ?? 0}
                  commits={kept?.commits ?? 0}
                />
              ) : null}
              <ProofSummary proof={proof} run={position} />
              {done !== null && change !== null ? (
                <FilesChanged
                  files={change.files}
                  onOpenDiff={() => setTab("implementation")}
                />
              ) : null}
            </div>
            <Tabs
              value={tab}
              onValueChange={setTab}
              className="order-3 min-w-0 gap-0 overflow-hidden rounded-[14px] border border-border bg-card shadow-sm lg:order-none [&>[role=tabpanel]]:p-4"
            >
              <TabsList className="m-3 mb-0 h-auto w-auto gap-1 rounded-[14px] p-1">
                {tabs.map((tab) => (
                  <TabsTrigger
                    key={tab.value}
                    value={tab.value}
                    className={tabClass}
                  >
                    {tab.label}
                  </TabsTrigger>
                ))}
              </TabsList>
              <TabsContent value="ledger" className="flex flex-col gap-6">
                <LedgerSummary statements={state.span?.statements ?? []} />
                <details className="group rounded-xl border border-border">
                  <summary className="flex min-h-11 cursor-pointer items-center gap-2 px-4 text-sm text-muted-foreground select-none">
                    <span>
                      All records · {state.span?.statements.length ?? 0}
                    </span>
                    <span className="ml-auto text-xs">
                      evidence, commits and commands
                    </span>
                  </summary>
                  <div className="px-4 pb-4">
                    <Ledger statements={state.span?.statements ?? []} />
                  </div>
                </details>
                <RawOutput
                  transcript={state.transcript}
                  statements={state.span?.statements ?? []}
                />
                {done !== null && change !== null ? (
                  <FinishedRecord
                    risks={change.report.risks}
                    details={runDetails(
                      state.span?.statements ?? [],
                      state.span?.stageAgents
                    )}
                  />
                ) : null}
              </TabsContent>
              <TabsContent value="implementation">
                {done !== null &&
                change !== null &&
                change.files.length === 0 ? (
                  <ProblemCard
                    label="No diff to read"
                    tone="info"
                    title="This run finished, and its diff is gone"
                    cause="The branch was merged without a merge commit, or deleted, and Span did not keep the patch."
                    kept={`${done.landed}. The change itself is still in your repository.`}
                    next="Open the change to read what shipped. Proof and test results are still in the ledger."
                  >
                    {done.change !== null ? (
                      <a className={buttonVariants()} href={done.change.href}>
                        {done.change.label}
                      </a>
                    ) : null}
                  </ProblemCard>
                ) : state.span !== null && change !== null ? (
                  <Implementation
                    change={change}
                    commits={state.implementation?.commitLog}
                    truncated={state.implementation?.truncated ?? false}
                    run={{
                      cursor: state.span.cursor,
                      statements: state.span.statements,
                      agents: state.span.stageAgents ?? {},
                      feature: featureState.feature,
                      journal: journalState,
                    }}
                  />
                ) : null}
              </TabsContent>
              <TabsContent value="qa">
                <QaTab id={id} />
                <QACases
                  skipped={state.acceptance?.skipped ?? false}
                  lines={state.acceptance?.lines ?? []}
                  command={state.acceptance?.command ?? ""}
                  exitCode={state.acceptance?.exitCode ?? 0}
                />
              </TabsContent>
            </Tabs>
          </div>
          <div className="contents lg:flex lg:min-w-0 lg:flex-col lg:gap-4">
            {kept !== null ? (
              <SafeAndKept kept={kept} className="order-2 lg:order-none" />
            ) : null}
            {cursor?.done ? null : (
              <RunControls className="order-4 lg:order-none">
                <StageAgent
                  id={id}
                  current={cursor?.stage ?? ""}
                  choices={state.span?.stageAgents}
                />
                {!waiting && !cursor?.stop && cursor?.decision === undefined ? (
                  <InstructionNote
                    open={showNote}
                    selected={selected}
                    timing={instructionTiming({
                      current,
                      selected,
                      hold: Boolean(cursor?.hold),
                      stop: Boolean(cursor?.stop),
                    })}
                    body={body}
                    note={note}
                    onSelect={setStage}
                    onBody={setBody}
                    onSave={(event) => void save(event)}
                    onSaveAndResume={
                      canAct && !cursor?.done
                        ? () => void saveAndResume()
                        : undefined
                    }
                    hold={Boolean(cursor?.hold)}
                    onOpen={openNote}
                  />
                ) : null}
                <Rewind
                  id={id}
                  current={cursor?.stage ?? ""}
                  blocked={restartBlocked}
                  startOpen={false}
                  requested={restartAt}
                />
                {!paused && !cursor?.done && !idle ? (
                  <div className="flex flex-col gap-2">
                    <div className="flex flex-wrap gap-2">
                      {running === null ? (
                        <Button
                          type="button"
                          variant="secondary"
                          onClick={() => void pause()}
                        >
                          Pause after this stage
                        </Button>
                      ) : null}
                      <Button
                        type="button"
                        variant="secondary"
                        onClick={() => void stop()}
                      >
                        Stop now
                      </Button>
                    </div>
                    <p className="text-sm text-pretty text-muted-foreground">
                      The stage that is running finishes. The run holds before
                      the next one, so you can choose the agent for it.
                    </p>
                  </div>
                ) : null}
              </RunControls>
            )}
          </div>
        </div>
      </div>
    </Shell>
  )
}
