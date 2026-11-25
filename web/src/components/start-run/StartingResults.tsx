"use client"

import { ArrowLeftIcon, ArrowRightIcon } from "lucide-react"

import { KeptNext } from "@/components/problem/KeptNext"
import { Pill, Panel } from "@/components/shaping/session-ui"
import { Button } from "@/components/ui/button"
import type { Started } from "@/lib/api"
import { touchButton } from "@/lib/button-class"

/** N3: one result per repository after Start, with a way into each run. */
export function StartingResults({
  result,
  onOpen,
  onBack,
}: {
  result: Started
  onOpen: (id: string) => void
  onBack: () => void
}) {
  if (result.runs.length <= 1 && !result.partial) {
    return null
  }
  const title = result.partial
    ? "Some runs started"
    : `${result.runs.length} runs started`
  return (
    <section aria-label="Starting results" className="flex flex-col gap-4">
      <Panel className="flex flex-col">
        <header className="flex flex-wrap items-center gap-3 border-b border-border px-4 py-3">
          <h2 className="text-lg leading-[26px] font-semibold">{title}</h2>
          {result.partial ? <Pill tone="hold">Partly started</Pill> : null}
          <span className="text-[13px] text-muted-foreground">
            {result.partial
              ? "The failure does not stop the others."
              : "Each repository runs on its own branch."}
          </span>
        </header>
        <ul>
          {result.runs.map((run) => (
            <li
              key={run.id}
              className="flex flex-wrap items-center gap-3 border-b border-border px-4 py-3 last:border-b-0"
            >
              <div className="flex min-w-0 flex-1 flex-col gap-0.5">
                <span className="flex flex-wrap items-center gap-2 text-sm font-semibold">
                  {run.repo === "" ? "repository" : run.repo}
                  <Pill tone="acc" small>
                    {run.stage} · running
                  </Pill>
                </span>
                <span className="font-mono text-xs break-all text-muted-foreground">
                  {run.id}
                </span>
              </div>
              <Button
                type="button"
                className={touchButton}
                onClick={() => onOpen(run.id)}
              >
                <ArrowRightIcon aria-hidden="true" />
                Open {run.repo === "" ? run.id : run.repo}
              </Button>
            </li>
          ))}
        </ul>
        {result.notes.trim() !== "" ? (
          <p className="border-t border-border bg-fail-soft px-4 py-3 text-sm text-pretty whitespace-pre-line text-fail">
            {result.notes.trim()}
          </p>
        ) : null}
      </Panel>
      <KeptNext
        kept={
          result.partial
            ? `The ${result.runs.length} running runs and their branches. Nothing was created in the others.`
            : "Your folder and files are untouched. Nothing is on the base branch."
        }
        next={
          result.partial
            ? "Start the others later from the desk."
            : "Open a run. Intake asks you only if the story is unclear."
        }
      />
      <Button
        type="button"
        variant="ghost"
        className="self-center"
        onClick={onBack}
      >
        <ArrowLeftIcon aria-hidden="true" />
        Back to Desk
      </Button>
    </section>
  )
}
