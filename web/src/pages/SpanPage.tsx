import { useState, type FormEvent } from "react"
import { Link, useParams } from "react-router"

import { answerRun, approveStage, branchLine, pauseRun, readStory, relaunchRun, resumeRun, saveInstruction, stopRun } from "@/lib/api"
import { agentBlockLine } from "@/lib/agent-block"
import { conflictHold, proposalKey } from "@/lib/conflict"
import { activeFailure, approvalOf, holdAction, isCheckoutBusy } from "@/lib/failure"
import { instructionTiming, noteStage, stageLine } from "@/lib/stage-copy"
import { agentLine, timelineLines, withAgent } from "@/lib/timeline"
import { waitingQuestions } from "@/lib/waiting"
import { stages } from "@/lib/types"
import { useSpan } from "@/hooks/use-span"
import { ConflictPanel } from "@/components/conflict/ConflictPanel"
import { FailureCard } from "@/components/failure/FailureCard"
import { Implementation } from "@/components/implementation/Implementation"
import { useFeature } from "@/hooks/use-feature"
import { Ledger } from "@/components/ledger/Ledger"
import { QACases } from "@/components/qa-cases/QACases"
import { Rewind } from "@/components/rewind/Rewind"
import { StageAgent } from "@/components/stage-agent/StageAgent"
import { Shell } from "@/components/shell/Shell"
import { StageRail } from "@/components/stage-rail/StageRail"
import { Waiting } from "@/components/waiting/Waiting"
import { Button } from "@/components/ui/button"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Textarea } from "@/components/ui/textarea"

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

  async function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setNote("")
    try {
      await saveInstruction(id, selected, body)
      setNote("Instruction saved.")
    } catch (err: unknown) {
      setNote(err instanceof Error ? err.message : "the instruction was not saved")
    }
  }

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

  async function answer(answers: { question: string; body: string }[]) {
    setNote("")
    try {
      await answerRun(id, answers)
      state.note("answer saved")
      setNote("Answer saved. Frame is reading the story again.")
    } catch (err: unknown) {
      setNote(err instanceof Error ? err.message : "the answer was not saved")
    }
  }

  async function reloadStory() {
    setNote("")
    try {
      await readStory(id)
      state.note("reading the story again")
      setNote("Reading the story again.")
    } catch (err: unknown) {
      setNote(err instanceof Error ? err.message : "the story was not read")
    }
  }

  async function resume() {
    setNote("")
    try {
      await resumeRun(id)
      state.note("resume")
      setNote("The run is continuing.")
    } catch (err: unknown) {
      setNote(err instanceof Error ? err.message : "the run did not continue")
    }
  }

  async function approve() {
    if (approval === null) {
      return
    }
    setNote("")
    try {
      await approveStage(id, approval.stage, approval.environment)
      state.note(`approved ${approval.stage}`)
      setNote("Approved. The run is continuing.")
    } catch (err: unknown) {
      setNote(err instanceof Error ? err.message : "the approval was not saved")
    }
  }

  async function pause() {
    setNote("")
    try {
      const why = await pauseRun(id)
      state.note("pause")
      setNote(why === "" ? "Pause requested." : why)
    } catch (err: unknown) {
      setNote(err instanceof Error ? err.message : "the pause was not requested")
    }
  }

  async function stop() {
    setNote("")
    try {
      const why = await stopRun(id)
      state.note("stop")
      setNote(why === "" ? "Stop requested." : why)
    } catch (err: unknown) {
      setNote(err instanceof Error ? err.message : "the run was not stopped")
    }
  }

  async function relaunch() {
    setNote("")
    try {
      await relaunchRun(id)
      state.note("relaunch")
      setNote("The run is starting again.")
    } catch (err: unknown) {
      setNote(err instanceof Error ? err.message : "the run did not start")
    }
  }

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
        <StageRail stage={cursor?.stage ?? ""} verdict={verdict} sealed={sealed} />
        <p className="text-pretty">{line}</p>
        {note !== "" ? <p className="text-pretty text-sm text-hold">{note}</p> : null}
        {agentBlock !== null ? (
          <section className="flex flex-col gap-1 border border-border p-3" aria-label="Agent blocked">
            <p className="text-sm text-hold">{agentBlock.title}</p>
            <p className="text-pretty text-sm">{agentBlock.detail}</p>
            <p className="text-pretty text-sm text-muted-foreground">{agentBlock.hint}</p>
          </section>
        ) : null}
        {approval !== null ? (
          <div className="flex flex-col gap-2">
            <Button type="button" className="self-start" onClick={() => void approve()}>
              {approval.label}
            </Button>
            <p className="text-pretty text-sm text-muted-foreground">{approval.hint}</p>
          </div>
        ) : null}
        {failure !== null && !busy && approval === null && conflictKind === null ? <FailureCard failure={failure} /> : null}
        {conflict !== undefined ? <ConflictPanel key={proposalKey(conflict)} id={id} conflict={conflict} /> : null}
        {conflict === undefined && conflictKind === "choose" ? (
          <p className="text-pretty text-sm text-muted-foreground">The options for this conflict are on their way.</p>
        ) : null}
        {!waiting && approval === null && canAct && action !== null ? (
          <div className="flex flex-col gap-2">
            <Button type="button" className="self-start" onClick={() => void resume()}>
              {action.label}
            </Button>
            <p className="text-pretty text-sm text-muted-foreground">{action.hint}</p>
          </div>
        ) : null}
        {state.agent?.relaunch ? (
          <Button type="button" variant="secondary" className="self-start" onClick={() => void relaunch()}>
            Relaunch
          </Button>
        ) : null}
        {state.error !== "" ? <p className="text-pretty text-sm text-destructive">{state.error}</p> : null}
      </header>
      <Tabs defaultValue="ledger">
        <TabsList>
          <TabsTrigger value="ledger">Ledger</TabsTrigger>
          <TabsTrigger value="implementation">Implementation</TabsTrigger>
          <TabsTrigger value="qa">QA</TabsTrigger>
        </TabsList>
        <TabsContent value="ledger" className="flex flex-col gap-6">
          <StageAgent id={id} current={cursor?.stage ?? ""} choices={state.span?.stageAgents} />
          <Rewind id={id} current={cursor?.stage ?? ""} />
          {!paused && !cursor?.done && !idle ? (
            <div className="flex flex-col gap-2">
              <div className="flex flex-wrap gap-2">
                <Button type="button" variant="secondary" className="self-start" onClick={() => void pause()}>
                  Pause after this stage
                </Button>
                <Button type="button" variant="secondary" className="self-start" onClick={() => void stop()}>
                  Stop now
                </Button>
              </div>
              <p className="text-pretty text-sm text-muted-foreground">
                The stage that is running finishes. The run holds before the next one, so you can choose the agent for it.
              </p>
            </div>
          ) : null}
          <Ledger statements={state.span?.statements ?? []} />
          {waiting ? (
            <Waiting key={questions.join("\n")} questions={questions} note={note} onAnswer={answer} onReload={reloadStory} />
          ) : null}

          {cursor?.stop && !waiting ? <p className="text-pretty text-sm">This run will not continue.</p> : null}
          {!waiting && !cursor?.stop ? (
          <div className="flex flex-col gap-3 border-t border-border pt-4">
            {paused || showNote ? (
              <form className="flex flex-col gap-3" onSubmit={(event) => void save(event)}>
                <Label htmlFor="instruction-stage">Instruction</Label>
                <p className="text-pretty text-sm text-muted-foreground">
                  {instructionTiming({ current, selected, hold: Boolean(cursor?.hold), stop: Boolean(cursor?.stop) })}
                </p>
                <Select
                  value={selected}
                  onValueChange={(value) => {
                    setStage(value)
                  }}
                >
                  <SelectTrigger id="instruction-stage" className="w-40">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {stages.map((name) => (
                      <SelectItem key={name} value={name}>
                        {name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <Textarea
                  aria-label="Instruction body"
                  value={body}
                  onChange={(event) => {
                    setStage(selected)
                    setBody(event.target.value)
                  }}
                  rows={6}
                />
                <div className="flex items-center gap-3">
                  <Button type="submit" variant="secondary" className="self-start">
                    Save instruction
                  </Button>
                  {note !== "" ? <p className="text-pretty text-sm">{note}</p> : null}
                </div>
              </form>
            ) : (
              <>
                <p className="text-pretty text-sm text-muted-foreground">
                  A stage reads a note when it starts. The stage already running does not see a note saved now.
                </p>
                <Button
                  type="button"
                  variant="secondary"
                  className="self-start"
                  onClick={() => {
                    setStage(selected)
                    setShowNote(true)
                  }}
                >
                  Leave a note for a stage
                </Button>
              </>
            )}
          </div>
          ) : null}
        </TabsContent>
        <TabsContent value="implementation">
          {state.span !== null ? (
          <Implementation
              stat={state.implementation?.stat ?? ""}
              patch={state.implementation?.patch ?? ""}
              truncated={state.implementation?.truncated ?? false}
              transcript={state.transcript}
              run={{
                cursor: state.span.cursor,
                statements: state.span.statements,
                agents: state.span.stageAgents ?? {},
                feature: featureState.feature,
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
    </Shell>
  )
}
