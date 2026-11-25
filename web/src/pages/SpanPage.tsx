"use client"

import { useMemo, useState, type FormEvent } from "react"
import { Link, useParams } from "react-router"

import { answerRun, approveStage, branchLine, decideRun, pauseRun, readStory, relaunchRun, resumeRun, saveInstruction, stopRun } from "@/lib/api"
import { agentBlockLine } from "@/lib/agent-block"
import { conflictHold, proposalKey } from "@/lib/conflict"
import { activeFailure, approvalOf, attemptLine, holdAction, isCheckoutBusy } from "@/lib/failure"
import { changeOf } from "@/lib/change"
import { keptWork } from "@/lib/kept"
import { doneOf } from "@/lib/done"
import { proofOf } from "@/lib/proof"
import { sinceLine } from "@/lib/since"
import { runningOf } from "@/lib/running"
import { needsYou } from "@/lib/needs-you"
import { instructionTiming, noteStage, stageLine } from "@/lib/stage-copy"
import { agentLine, timelineLines, withAgent } from "@/lib/timeline"
import { waitingQuestions } from "@/lib/waiting"
import { useSpan } from "@/hooks/use-span"
import { ConflictPanel } from "@/components/conflict/ConflictPanel"
import { FailureCard } from "@/components/failure/FailureCard"
import { Implementation } from "@/components/implementation/Implementation"
import { useEvents } from "@/hooks/use-events"
import { useFeature } from "@/hooks/use-feature"
import { InstructionNote } from "@/components/instruction-note/InstructionNote"
import { RawOutput } from "@/components/raw-output/RawOutput"
import { Ledger } from "@/components/ledger/Ledger"
import { NeedsYou } from "@/components/needs-you/NeedsYou"
import { ProofSummary } from "@/components/proof-summary/ProofSummary"
import { QACases } from "@/components/qa-cases/QACases"
import { Rewind } from "@/components/rewind/Rewind"
import { DoneCard } from "@/components/done-card/DoneCard"
import { RunningCard } from "@/components/running-card/RunningCard"
import { RunControls } from "@/components/run-controls/RunControls"
import { SafeAndKept } from "@/components/safe-and-kept/SafeAndKept"
import { StageAgent } from "@/components/stage-agent/StageAgent"
import { Shell } from "@/components/shell/Shell"
import { StageRail } from "@/components/stage-rail/StageRail"
import { stageDurations } from "@/lib/stage-times"
import { Decision } from "@/components/decision/Decision"
import { Waiting } from "@/components/waiting/Waiting"
import { ActionWithHint } from "@/components/action-with-hint/ActionWithHint"
import { Button } from "@/components/ui/button"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"

/** Three readings of one run. The socket wakes the page. The statements stay the record. */
export function SpanPage() {
  const params = useParams()
  const id = params.id ?? ""
  const state = useSpan(id)
  const [showNote, setShowNote] = useState(false)
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
    noteStage({ current, hold: Boolean(cursor?.hold), stop: Boolean(cursor?.stop) }) ||
    noteStage({ current, hold: paused, stop: Boolean(cursor?.stop) })
  const verdict = cursor?.done ? "done" : cursor?.stop ? "stop" : cursor?.hold ? "hold" : "running"
  const line =
    state.agent?.summary === "not running"
      ? "No agent is running this stage. Relaunch starts it again."
      : stageLine({
          stage: cursor?.stage ?? "intake",
          reason: cursor?.reason ?? "",
          hold: cursor?.hold ?? false,
          stop: cursor?.stop ?? false,
          done: cursor?.done ?? false,
        })

  const lines = withAgent(
    timelineLines({
      statements: state.span?.statements ?? [],
      gate: state.span?.gate,
      frames: state.frames,
      actions: state.actions,
    }),
    agentLine(state.agent, state.agentAt),
  )
  const sealed = (state.span?.statements ?? []).filter((item) => item.verdict === "advance").map((item) => item.stage)
  const socket = state.connected ? "socket open" : "socket closed"
  const questions = waitingQuestions(cursor ?? null)
  const waiting = questions.length > 0
  const agentBlock = agentBlockLine(cursor?.agentBlock)
  const failure = activeFailure({
    statements: state.span?.statements ?? [],
    cursor: cursor ?? null,
  })
  // A run whose process died is not held and not done, so the cursor alone cannot tell
  // the page that there is nothing to do. The agent poll can.
  const idle = state.agent?.summary === "not running"
  const busy = isCheckoutBusy(cursor?.reason ?? "")
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
  const showFailure = failure !== null && !busy && approval === null && conflictKind === null
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
    needs === null && cursor !== undefined && !cursor.done && state.agent?.running === true
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
  const lastStamp = [...(state.span?.statements ?? [])].reverse().find((item) => item.time)?.time ?? ""
  const since =
    cursor?.done === true
      ? ""
      : sinceLine(waiting ? "waiting since" : cursor?.stop ? "stopped" : "held", lastStamp, new Date())
  const attempt = failure === null ? "" : attemptLine(failure)
  const where = needs?.kind === "failed" && attempt !== "" ? `${current} · ${attempt}` : current

  /**
   * Runs one action the page offers and says what happened. Every action reports its failure
   * the same way, so the person always reads either the confirmation or the reason.
   */
  async function act(
    request: () => Promise<unknown>,
    outcome: { log: string; say: string | ((result: unknown) => string); failed: string },
  ) {
    setNote("")
    try {
      const result = await request()
      if (outcome.log !== "") {
        state.note(outcome.log)
      }
      setNote(typeof outcome.say === "function" ? outcome.say(result) : outcome.say)
    } catch (err: unknown) {
      setNote(err instanceof Error ? err.message : outcome.failed)
    }
  }

  const asked = (what: string) => (why: unknown) => (typeof why === "string" && why !== "" ? why : `${what} requested.`)

  async function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    await act(() => saveInstruction(id, selected, body), {
      log: "",
      say: "Instruction saved.",
      failed: "the instruction was not saved",
    })
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
    act(() => resumeRun(id), { log: "resume", say: "The run is continuing.", failed: "the run did not continue" })

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

  const pause = () => act(() => pauseRun(id), { log: "pause", say: asked("Pause"), failed: "the pause was not requested" })

  const stop = () => act(() => stopRun(id), { log: "stop", say: asked("Stop"), failed: "the run was not stopped" })

  const relaunch = () =>
    act(() => relaunchRun(id), { log: "relaunch", say: "The run is starting again.", failed: "the run did not start" })

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
    [state.implementation, state.span, featureState.feature],
  )
  // Before the diff is known the proof would only say "nothing yet", so the panel waits.
  const proof = change === null || state.implementation === null ? null : proofOf(change.report, change.files)
  const kept = keptWork({
    statements: state.span?.statements ?? [],
    branch: cursor?.branch ?? "",
    head: cursor?.gitSHA ?? "",
    stat: state.implementation?.stat ?? "",
  })
  // The engine refuses to restart a stage while the checkout is in the middle of a merge.
  const restartBlocked =
    conflict !== undefined || conflictKind !== null ? "Unavailable while a merge conflict is open. Abort the merge first." : ""

  return (
    <Shell path={`~/spans/${id}`} status={`${cursor?.stage || "intake"} · ${verdict} · ${socket}`} lines={lines}>
      <header className="flex flex-col gap-3">
        <Link className="text-sm text-primary underline-offset-4 hover:underline" to="/">
          $ desk
        </Link>
        <h1 className="text-balance text-xl tabular-nums" title={id}>
          <span className="text-primary">$ </span>
          {state.span?.label || id}
        </h1>
        {cursor !== undefined && branchLine(cursor) !== "" ? (
          <p className="text-pretty text-sm text-muted-foreground">{branchLine(cursor)}</p>
        ) : null}
        <StageRail
          stage={cursor?.stage ?? ""}
          verdict={verdict}
          sealed={sealed}
          durations={stageDurations(state.span?.statements ?? [])}
        />
        <p className="text-pretty">{line}</p>
        {since !== "" && (cursor?.hold || cursor?.stop) ? (
          <p className="text-pretty text-sm tabular-nums text-muted-foreground">{since}</p>
        ) : null}
        {note !== "" ? <p className="text-pretty text-sm text-hold">{note}</p> : null}
        {state.error !== "" ? <p className="text-pretty text-sm text-destructive">{state.error}</p> : null}
      </header>
      {/*
        One column on a narrow screen, in the order the design draws it: what needs the person, the
        proof, what is safe, the tabs, then the controls. From lg up the first and last pairs split
        into a main column and a side column; `contents` lets the order classes reach the grid.
      */}
      <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_20rem] lg:items-start">
        <div className="contents lg:flex lg:min-w-0 lg:flex-col lg:gap-6">
          <div className="order-1 flex min-w-0 flex-col gap-6 lg:order-none">
            {needs !== null ? (
              <NeedsYou needs={needs} where={where}>
                {agentBlock !== null ? (
                  <section className="flex flex-col gap-1" aria-label="Agent blocked">
                    <h2 className="text-base font-medium">{agentBlock.title}</h2>
                    <p className="text-pretty text-sm">{agentBlock.detail}</p>
                    <p className="text-pretty text-sm text-muted-foreground">{agentBlock.hint}</p>
                  </section>
                ) : null}
                {approval !== null ? (
                  <ActionWithHint label={approval.label} hint={approval.hint} onClick={() => void approve()} />
                ) : null}
                {showFailure ? <FailureCard failure={failure} /> : null}
                {conflict !== undefined ? <ConflictPanel key={proposalKey(conflict)} id={id} conflict={conflict} /> : null}
                {conflict === undefined && conflictKind === "choose" ? (
                  <p className="text-pretty text-sm text-muted-foreground">The options for this conflict are on their way.</p>
                ) : null}
                {waiting ? (
                  <Waiting key={questions.join("\n")} questions={questions} note={note} onAnswer={answer} onReload={reloadStory} />
                ) : null}
                {cursor?.decision !== undefined && !waiting ? (
                  <Decision key={cursor.decision.id} request={cursor.decision} note={note} onDecide={decide} />
                ) : null}
                {!waiting && cursor?.decision === undefined && approval === null && canAct && action !== null ? (
                  <ActionWithHint label={action.label} hint={action.hint} onClick={() => void resume()} />
                ) : null}
                {state.agent?.relaunch ? (
                  <Button type="button" variant="secondary" className="self-start" onClick={() => void relaunch()}>
                    Relaunch
                  </Button>
                ) : null}
                {cursor?.stop && !waiting ? <p className="text-pretty text-sm">This run will not continue.</p> : null}
              </NeedsYou>
            ) : null}
            {done !== null ? <DoneCard done={done} /> : null}
            {running !== null ? <RunningCard running={running} onPause={() => void pause()} /> : null}
            {proof !== null ? <ProofSummary proof={proof} /> : null}
          </div>
          <Tabs defaultValue="ledger" className="order-3 min-w-0 lg:order-none">
            <TabsList>
              <TabsTrigger value="ledger">Ledger</TabsTrigger>
              <TabsTrigger value="implementation">Implementation</TabsTrigger>
              <TabsTrigger value="qa">QA</TabsTrigger>
            </TabsList>
            <TabsContent value="ledger" className="flex flex-col gap-6">
              <Ledger statements={state.span?.statements ?? []} />
              <RawOutput transcript={state.transcript} statements={state.span?.statements ?? []} />
            </TabsContent>
            <TabsContent value="implementation">
              {state.span !== null && change !== null ? (
                <Implementation
                  change={change}
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
          {kept !== null ? <SafeAndKept kept={kept} className="order-2 lg:order-none" /> : null}
          <RunControls className="order-4 lg:order-none">
            <StageAgent id={id} current={cursor?.stage ?? ""} choices={state.span?.stageAgents} />
            {!waiting && !cursor?.stop && cursor?.decision === undefined ? (
              <InstructionNote
                open={paused || showNote}
                selected={selected}
                timing={instructionTiming({ current, selected, hold: Boolean(cursor?.hold), stop: Boolean(cursor?.stop) })}
                body={body}
                note={note}
                onSelect={setStage}
                onBody={setBody}
                onSave={(event) => void save(event)}
                onOpen={() => {
                  setStage(selected)
                  setShowNote(true)
                }}
              />
            ) : null}
            <Rewind id={id} current={cursor?.stage ?? ""} blocked={restartBlocked} />
            {!paused && !cursor?.done && !idle ? (
              <div className="flex flex-col gap-2">
                <div className="flex flex-wrap gap-2">
                  {running === null ? (
                    <Button type="button" variant="secondary" onClick={() => void pause()}>
                      Pause after this stage
                    </Button>
                  ) : null}
                  <Button type="button" variant="secondary" onClick={() => void stop()}>
                    Stop now
                  </Button>
                </div>
                <p className="text-pretty text-sm text-muted-foreground">
                  The stage that is running finishes. The run holds before the next one, so you can choose the agent for it.
                </p>
              </div>
            ) : null}
          </RunControls>
        </div>
      </div>
    </Shell>
  )
}
