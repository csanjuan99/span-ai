"use client"

import { useState, type ReactNode } from "react"

import type { DecisionEntry, Reasoning } from "@/lib/implementation-api"

function Card({
  title,
  caption,
  children,
}: {
  title: string
  caption: string
  children: ReactNode
}) {
  return (
    <section
      aria-label={title}
      className="rounded-[14px] border border-border bg-card p-4 shadow-sm sm:p-5"
    >
      <div className="mb-3 flex flex-wrap items-baseline gap-2">
        <h2 className="text-lg leading-[26px] font-semibold">{title}</h2>
        <span className="text-xs text-muted-foreground">{caption}</span>
      </div>
      {children}
    </section>
  )
}

function Decision({
  decision,
  onCheck,
  busy,
}: {
  decision: DecisionEntry
  onCheck: (id: string) => void
  busy: boolean
}) {
  return (
    <li className="flex flex-col gap-2 rounded-[10px] border border-border p-3">
      <div className="flex flex-wrap items-start gap-2">
        <span className="font-mono text-xs text-muted-foreground">
          {decision.id}
        </span>
        <p className="min-w-0 flex-1 text-[14px] leading-[22px] font-medium text-pretty">
          {decision.what}
        </p>
        {decision.assumed ? (
          <span className="rounded-full bg-hold-soft px-2.5 py-0.5 text-xs font-medium text-hold">
            Assumed: check before you approve
          </span>
        ) : (
          <span className="rounded-full bg-pass-soft px-2.5 py-0.5 text-xs font-medium text-pass">
            From the story
          </span>
        )}
      </div>
      {decision.why !== "" ? (
        <p className="text-[13px] leading-5 text-pretty text-muted-foreground">
          <span className="font-medium text-foreground">Because </span>
          {decision.why}
        </p>
      ) : (
        <p className="text-[13px] text-hold">No reason given.</p>
      )}
      <p className="text-xs text-muted-foreground">
        Source: {decision.source === "" ? "not in the story" : decision.source}
        {decision.files.length > 0 ? (
          <>
            {" "}
            · Files:{" "}
            <span className="font-mono">{decision.files.join(", ")}</span>
          </>
        ) : null}
      </p>
      {decision.assumed ? (
        <label className="inline-flex min-h-9 items-center gap-2 text-[13px] max-sm:min-h-11">
          <input
            type="checkbox"
            className="size-4 accent-primary"
            checked={decision.checked}
            disabled={decision.checked || busy}
            aria-label={`I checked ${decision.id}`}
            onChange={() => onCheck(decision.id)}
          />
          I checked this
        </label>
      ) : null}
    </li>
  )
}

/**
 * What the story asked, the plan, and the choices the agent made with why, where each came
 * from and whether a person checked it. A run with no decision says so and shows what is left.
 */
export function ReasoningTab({
  view,
  note,
  busy,
  onCheck,
  transcriptLines,
}: {
  view: Reasoning | null
  note: string
  busy: boolean
  onCheck: (ids: string[]) => void
  /** Lines of the implement transcript, for the empty state; 0 when not known. */
  transcriptLines: number
}) {
  const [picked, setPicked] = useState<string[]>([])
  if (view === null) {
    return (
      <p className="rounded-[14px] border border-border bg-card p-5 text-sm text-muted-foreground">
        {note === "" ? "Reading the reasoning…" : note}
      </p>
    )
  }
  const open = view.decisions.filter(
    (decision) => decision.assumed && !decision.checked
  )
  const done = view.tasks.filter((task) => task.done).length
  return (
    <div className="flex flex-col gap-4">
      {note !== "" ? (
        <p role="alert" className="text-sm text-destructive">
          {note}
        </p>
      ) : null}
      <Card title="What the story asked" caption="Quoted, not rephrased">
        {view.specs.length === 0 ? (
          <p className="text-sm text-muted-foreground">
            The feature document has no Specs section yet.
          </p>
        ) : (
          <ol className="flex flex-col gap-3">
            {view.specs.map((spec) => (
              <li key={spec.id} className="flex flex-col gap-0.5">
                <span className="text-xs text-muted-foreground">
                  {spec.id} · verbatim
                </span>
                <span className="text-[14px] leading-[22px] text-pretty">
                  {spec.text}
                </span>
              </li>
            ))}
          </ol>
        )}
      </Card>
      {view.tasks.length > 0 ? (
        <Card
          title="Tasks"
          caption={`${done} of ${view.tasks.length} done · from the plan`}
        >
          <ul className="flex flex-col gap-1.5">
            {view.tasks.map((task) => (
              <li
                key={task.text}
                className="flex items-baseline gap-2 text-[13px] leading-5"
              >
                <span
                  aria-hidden="true"
                  className={
                    task.done
                      ? "font-mono text-pass"
                      : "font-mono text-muted-foreground"
                  }
                >
                  {task.done ? "+" : "·"}
                </span>
                <span className="sr-only">
                  {task.done ? "done" : "not done"}
                </span>
                <span className="min-w-0 flex-1 text-pretty">{task.text}</span>
                {task.commit !== "" ? (
                  <span className="font-mono text-xs text-muted-foreground">
                    {task.commit.slice(0, 7)}
                  </span>
                ) : null}
              </li>
            ))}
          </ul>
        </Card>
      ) : null}
      {view.decisions.length === 0 ? (
        <Card title="What the agent decided" caption="Nothing recorded">
          <p className="text-sm font-semibold">
            The agent recorded no decisions for this run
          </p>
          <p className="mt-1 text-[13px] leading-5 text-pretty text-muted-foreground">
            So there is no “because” to read. Span does not guess one.{" "}
            {view.specs.length > 0 || view.tasks.length > 0
              ? "The story’s criteria and the tasks above are still shown."
              : ""}
          </p>
          <dl className="mt-3 flex flex-col gap-1.5 text-[13px] leading-5">
            <div>
              <dt className="inline font-medium">Evidence </dt>
              <dd className="inline text-muted-foreground">
                The diff, the commits and the tests.
              </dd>
            </div>
            {transcriptLines > 0 ? (
              <div>
                <dt className="inline font-medium">Kept </dt>
                <dd className="inline text-muted-foreground">
                  The transcript of implement, {transcriptLines} lines.
                </dd>
              </div>
            ) : null}
            <div>
              <dt className="inline font-medium">Next </dt>
              <dd className="inline text-muted-foreground">
                Read the transcript, or ask the agent to explain before you
                approve.
              </dd>
            </div>
          </dl>
        </Card>
      ) : (
        <Card
          title="What the agent decided"
          caption={`${view.decisions.length} decisions · ${view.decisions.filter((decision) => decision.assumed).length} are assumed`}
        >
          <ol className="flex flex-col gap-3">
            {view.decisions.map((decision) => (
              <Decision
                key={decision.id}
                decision={decision}
                busy={busy}
                onCheck={(id) => {
                  setPicked([])
                  onCheck([id])
                }}
              />
            ))}
          </ol>
        </Card>
      )}
      {open.length > 0 ? (
        <section
          aria-label="Before you approve"
          className="rounded-[14px] border border-border bg-card p-4 shadow-sm sm:p-5"
        >
          <h2 className="mb-2 text-lg leading-[26px] font-semibold">
            Before you approve
          </h2>
          <dl className="flex flex-col gap-1.5 text-[13px] leading-5">
            <div>
              <dt className="inline font-medium">Check </dt>
              <dd className="inline text-muted-foreground">
                {open.map((decision) => decision.id).join(", ")}{" "}
                {open.length === 1 ? "was" : "were"} assumed. The story did not
                say.
              </dd>
            </div>
            <div>
              <dt className="inline font-medium">Changes </dt>
              <dd className="inline text-muted-foreground">
                Marking them checked is saved on the run. It changes no code.
              </dd>
            </div>
            <div>
              <dt className="inline font-medium">Next </dt>
              <dd className="inline text-muted-foreground">
                Then approve on the Ledger tab. Or leave a note to change one.
              </dd>
            </div>
          </dl>
          <button
            type="button"
            disabled={busy || picked.length > 0}
            className="mt-3 inline-flex h-10 items-center rounded-[10px] bg-primary px-4 text-[13px] font-semibold text-primary-foreground disabled:opacity-60 max-sm:h-11"
            onClick={() => {
              setPicked(open.map((decision) => decision.id))
              onCheck(open.map((decision) => decision.id))
            }}
          >
            Mark {open.length}{" "}
            {open.length === 1 ? "assumption" : "assumptions"} as checked
          </button>
        </section>
      ) : null}
    </div>
  )
}
