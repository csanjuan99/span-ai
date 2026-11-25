"use client"

import { Panel, Pill } from "@/components/shaping/session-ui"
import { stages } from "@/lib/types"

/** What the desk knows while Start is working: how many runs, and when the folder was checked. */
export interface StartPending {
  runs: number
  /** Clock time the folder check passed; empty while it is still running. */
  checkedAt: string
}

function Step({
  mark,
  children,
  when,
}: {
  mark: string
  children: string
  when: string
}) {
  return (
    <li className="flex items-baseline gap-3 py-1 text-sm">
      <span aria-hidden="true" className="w-3 font-mono text-primary">
        {mark}
      </span>
      <span className="min-w-0 flex-1">{children}</span>
      <span className="text-xs text-muted-foreground">{when}</span>
    </li>
  )
}

/**
 * N3, state A: the steps Start has reached, as the page knows them. The rail is the engine's ten
 * stages with intake first in line; nothing here is invented, so a step appears when it is true.
 */
export function StartingProgress({ pending }: { pending: StartPending }) {
  const many = pending.runs > 1
  return (
    <section
      aria-label="Starting"
      aria-live="polite"
      className="flex flex-col gap-4"
    >
      <Panel className="flex flex-col items-center gap-3 px-4 py-5 text-center">
        <ol className="flex flex-wrap justify-center gap-2" aria-label="Stages">
          {stages.map((stage, index) => (
            <li key={stage}>
              <Pill tone={index === 0 ? "acc" : "mut"} small>
                {index === 0 ? "> " : "· "}
                {stage}
              </Pill>
            </li>
          ))}
        </ol>
        <h2 className="text-sm font-semibold">
          {many ? `Starting ${pending.runs} runs` : "Starting the run"}
        </h2>
        <p className="text-[13px] text-muted-foreground">
          You can leave this page; the {many ? "runs start" : "run starts"}{" "}
          anyway.
        </p>
      </Panel>
      <Panel className="px-4 py-2">
        <ul>
          {pending.checkedAt === "" ? (
            <Step mark=">" when="now">
              Checking the folder
            </Step>
          ) : (
            <>
              <Step mark="+" when={pending.checkedAt}>
                Checked the folder
              </Step>
              <Step mark=">" when="now">
                {many
                  ? "Starting intake in each repository"
                  : "Starting intake"}
              </Step>
            </>
          )}
        </ul>
      </Panel>
    </section>
  )
}
