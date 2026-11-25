"use client"

import { CircleAlertIcon, PlayIcon, RotateCcwIcon } from "lucide-react"
import { useState } from "react"

import { AgentSelect } from "@/components/shape-entry/AgentModelSelects"
import { useInterviewAgents } from "@/components/shape-entry/useInterviewAgents"
import { Consequences } from "@/components/status-block/Consequences"
import { Button } from "@/components/ui/button"
import { resumeShape } from "@/lib/api"
import { clockTime } from "@/lib/clock"
import { reasonText } from "@/lib/guards"
import {
  failureCause,
  keptLines,
  resetHint,
  sessionRef,
} from "@/lib/shape-sessions"
import type { ShapeSession } from "@/lib/types"
import {
  captionText,
  errorText,
  headingText,
  labelText,
  monoWrapped,
  smallTitleText,
  titleText,
} from "@/components/shaping/styles"

const kindLabels = {
  usage_limit: "usage limit",
  repo_changed: "repository changed",
  error: "agent error",
}

/**
 * The "Interview stopped" band of a session (SH10): why the agent stopped, what was kept,
 * and what resuming does. The session page shows it while `session.failure` is set;
 * `onResumed` gets the session the resumed turn returns, and `onLater` is "wait".
 */
export function InterviewStopped({
  session,
  onResumed,
  onLater,
}: {
  session: ShapeSession
  onResumed: (next: ShapeSession) => void
  onLater: () => void
}) {
  const failure = session.failure
  const [picked, setPicked] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const [note, setNote] = useState("")
  const failed = failure?.agent ?? session.agent ?? ""
  const agents = useInterviewAgents(session.workspace, picked ?? failed)
  if (failure === null || failure === undefined) {
    return null
  }
  // A limit is the agent's, so the suggestion is another CLI that is ready.
  const other =
    failure.kind === "usage_limit"
      ? agents.choice.agents.find((row) => row.ready && row.id !== failed)?.id
      : undefined
  const agent = picked ?? other ?? failed
  const reset = resetHint(failure.message)
  const name = agent === "" ? agents.projectAgent : agent
  const repo = session.repo === "" ? "whole folder" : session.repo

  async function resume() {
    setBusy(true)
    setNote("")
    try {
      onResumed(await resumeShape(session.id, session.workspace, agent, ""))
    } catch (err: unknown) {
      setNote(reasonText(err, "the interview did not resume"))
    } finally {
      setBusy(false)
    }
  }

  return (
    <section
      aria-label="Needs you"
      className="overflow-hidden rounded-[14px] border border-input bg-card shadow-sm"
    >
      <div className="flex flex-wrap items-center gap-2 bg-fail-soft px-4 py-2.5 text-fail md:px-5">
        <CircleAlertIcon aria-hidden="true" className="size-4" />
        <span className={labelText}>Interview stopped</span>
        <span className="ml-auto text-xs leading-4">
          {clockTime(failure.at)} · {kindLabels[failure.kind]}
        </span>
      </div>
      <div className="flex flex-col gap-4 p-4 md:grid md:grid-cols-[minmax(0,1fr)_340px] md:gap-0 md:p-0">
        <div className="flex min-w-0 flex-col gap-4 md:p-6">
          <div className="flex flex-col gap-2">
            <h2 className={headingText}>The interview agent stopped</h2>
            <p className="text-sm leading-[22px]">
              <b>Cause.</b>{" "}
              {failureCause(failure, failed === "" ? "The agent" : failed)}
            </p>
            {failure.kind === "error" || failure.kind === "repo_changed" ? (
              <p className={monoWrapped}>{failure.message}</p>
            ) : null}
          </div>
          <div className="flex flex-col gap-2">
            <div className={titleText}>What was kept</div>
            <ul className="flex flex-col gap-2">
              {keptLines(session).map((line) => (
                <li key={line} className="grid grid-cols-[16px_1fr] gap-2 py-1">
                  <span
                    aria-hidden="true"
                    className="font-mono font-semibold text-pass"
                  >
                    +
                  </span>
                  <span className="text-[13px] leading-5">{line}</span>
                </li>
              ))}
              <li className="grid grid-cols-[16px_1fr] gap-2 py-1">
                <span
                  aria-hidden="true"
                  className="font-mono font-semibold text-muted-foreground"
                >
                  ·
                </span>
                <span className="text-[13px] leading-5">
                  Not finished: the rest of the map and the split check
                </span>
              </li>
            </ul>
          </div>
        </div>
        <div className="flex min-w-0 flex-col gap-4 md:border-l md:border-border md:p-6">
          <Consequences
            title="If you resume"
            icon={RotateCcwIcon}
            rows={[
              {
                label: "Changes",
                body: "The interview continues from the saved map. Nothing is redone.",
              },
              {
                label: "Where",
                body: (
                  <>
                    Same session{" "}
                    <span className="font-mono text-xs">
                      {sessionRef(session)}
                    </span>{" "}
                    · <span className="font-mono text-xs">{repo}</span>
                  </>
                ),
              },
              { label: "Kept", body: "Everything on the left." },
              {
                label: "Next",
                body: "It finishes the map, then asks what blocks.",
              },
            ]}
          />
          <div className="flex items-center gap-3">
            <div className="flex min-w-0 flex-1 flex-col">
              <div className={smallTitleText}>Agent for the rest</div>
              <div className={captionText}>This session only</div>
            </div>
            <AgentSelect
              label="Agent"
              agents={agents}
              value={agent}
              onChange={setPicked}
            />
          </div>
          <Button
            type="button"
            className="h-11 w-full rounded-[10px] px-4 md:h-10"
            disabled={busy}
            onClick={() => void resume()}
          >
            <PlayIcon aria-hidden="true" />
            {busy
              ? "Resuming…"
              : `Resume with ${name === "" ? "the project agent" : name}`}
          </Button>
          <Button
            type="button"
            variant="ghost"
            className="h-11 w-full rounded-[10px] px-4 md:h-10"
            onClick={onLater}
          >
            {reset === "" ? "Wait and resume later" : `Wait until ${reset}`}
          </Button>
          {note !== "" ? (
            <p role="alert" className={errorText}>
              {note}
            </p>
          ) : null}
        </div>
      </div>
    </section>
  )
}
