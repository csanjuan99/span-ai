import { CircleCheckIcon, ClockIcon, GitMergeIcon } from "lucide-react"
import type { ReactNode } from "react"

import { Pill } from "@/components/onboarding/Pill"
import { durationText, type FileLetter, type Summary } from "@/lib/summary"
import { cn } from "@/lib/utils"

const letterTone = {
  A: "bg-pass-soft text-pass",
  M: "bg-accent text-primary",
  D: "bg-fail-soft text-fail",
  R: "bg-hold-soft text-hold",
} as const

const letterName = {
  A: "added",
  M: "modified",
  D: "deleted",
  R: "renamed",
} as const

const levelTone = {
  high: "text-fail",
  medium: "text-hold",
  low: "text-pass",
} as const

function Block({ name, children }: { name: string; children: ReactNode }) {
  return (
    <section
      aria-label={name}
      className="flex flex-col gap-2 rounded-[14px] border border-border bg-card p-4 shadow-sm sm:p-5"
    >
      <h2 className="text-lg font-semibold">{name}</h2>
      {children}
    </section>
  )
}

function Cell({
  label,
  value,
  note,
}: {
  label: string
  value: ReactNode
  note: ReactNode
}) {
  return (
    <div className="flex min-w-0 flex-col gap-1 border-t border-border p-4 first:border-t-0 sm:border-t-0 sm:border-l sm:px-5 sm:first:border-l-0">
      <div className="text-xs font-medium text-muted-foreground">{label}</div>
      <div className="text-lg font-semibold tabular-nums">{value}</div>
      <div className="text-xs text-muted-foreground">{note}</div>
    </div>
  )
}

function letterCounts(files: Summary["files"]): string {
  const counts = new Map<FileLetter, number>()
  for (const file of files) {
    counts.set(file.letter, (counts.get(file.letter) ?? 0) + 1)
  }
  return (["A", "M", "D", "R"] as const)
    .filter((letter) => counts.has(letter))
    .map((letter) => `${letter} ${counts.get(letter)}`)
    .join(" · ")
}

function when(finishedAt: string): string {
  const date = new Date(finishedAt)
  return Number.isNaN(date.getTime())
    ? ""
    : date.toLocaleDateString("en-GB", { day: "numeric", month: "short" })
}

/** The header of a summary: the story, whether it landed, and where and when. */
export function SummaryHead({
  summary,
  done,
}: {
  summary: Summary
  done: boolean
}) {
  const took = durationText(summary.durationMs)
  const day = when(summary.finishedAt)
  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-center gap-3">
        {summary.issue !== "" ? (
          <span className="rounded-md border border-border bg-muted px-1.5 font-mono text-xs">
            {summary.issue}
          </span>
        ) : null}
        <Pill tone={done ? "pass" : "hold"}>
          {done ? (
            <CircleCheckIcon aria-hidden="true" className="size-3.5" />
          ) : null}
          {done ? "Done" : "In progress"}
        </Pill>
      </div>
      <h1 className="text-2xl font-semibold tracking-tight text-balance">
        {summary.title}
      </h1>
      <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-[13px] text-muted-foreground">
        {summary.sha !== "" ? (
          <span className="flex items-center gap-1.5">
            <GitMergeIcon aria-hidden="true" className="size-3.5" />
            {summary.outcome.toLowerCase().includes("merged")
              ? "merged as"
              : "ended on"}{" "}
            <span className="font-mono text-xs text-foreground">
              {summary.sha}
            </span>
            {summary.base !== "" ? (
              <>
                {" "}
                into{" "}
                <span className="font-mono text-xs text-foreground">
                  {summary.base}
                </span>
              </>
            ) : null}
          </span>
        ) : null}
        {day !== "" || took !== "" ? (
          <span className="flex items-center gap-1.5">
            <ClockIcon aria-hidden="true" className="size-3.5" />
            {[day, took].filter((part) => part !== "").join(", ")}
          </span>
        ) : null}
      </div>
    </div>
  )
}

/** The summary of a finished run: what landed, how risky, what changed, what the checks said. */
export function SummaryView({ summary }: { summary: Summary }) {
  return (
    <div className="flex flex-col gap-6">
      <section
        aria-label="Outcome"
        className="flex flex-col gap-2 rounded-[14px] border border-border bg-card p-4 shadow-sm sm:p-5"
      >
        <div className="text-xs font-medium text-muted-foreground">Verdict</div>
        <p className="text-lg font-semibold text-pretty">{summary.outcome}</p>
        {summary.verdict !== null ? (
          <p className={cn("text-[13px]", levelTone[summary.verdict.level])}>
            {summary.verdict.headline} · {summary.verdict.level} risk
          </p>
        ) : (
          <p className="text-[13px] text-muted-foreground">
            Not assessed: there is no diff to read for this run.
          </p>
        )}
      </section>
      {summary.assessed ? (
        <section
          aria-label="Proof"
          className="grid overflow-hidden rounded-[14px] border border-border bg-card shadow-sm sm:grid-cols-3"
        >
          <Cell
            label="Diff"
            value={
              <>
                {summary.totals.files}{" "}
                {summary.totals.files === 1 ? "file" : "files"} ·{" "}
                <span className="text-pass">+{summary.totals.added}</span>{" "}
                <span className="text-fail">−{summary.totals.removed}</span>
              </>
            }
            note={letterCounts(summary.files)}
          />
          <Cell
            label="Tests"
            value={
              summary.tests === null
                ? "No count"
                : `${summary.tests.passed} passed · ${summary.tests.failed} failed`
            }
            note={
              summary.tests === null
                ? "The checks printed no test count."
                : "From what the checks printed."
            }
          />
          <Cell
            label="Risks"
            value={`${summary.risks.length} noted`}
            note={
              summary.risks.length === 0
                ? "None found in the diff."
                : "Read from the diff."
            }
          />
        </section>
      ) : null}
      {summary.assessed ? (
        <Block name={`Files changed · ${summary.totals.files}`}>
          <p className="text-xs text-muted-foreground">
            A added · M modified · D deleted · R renamed
          </p>
          <ul className="flex flex-col divide-y divide-border">
            {summary.files.map((file) => (
              <li
                key={file.path}
                className="grid grid-cols-[24px_minmax(0,1fr)] items-center gap-x-3 py-2.5 text-sm sm:grid-cols-[24px_minmax(0,1fr)_auto]"
              >
                <span
                  aria-hidden="true"
                  className={cn(
                    "grid size-6 place-items-center rounded-md font-mono text-xs font-semibold",
                    letterTone[file.letter]
                  )}
                >
                  {file.letter}
                </span>
                <span className="sr-only">{letterName[file.letter]}</span>
                <span className="min-w-0 truncate font-mono text-xs">
                  {file.path}
                </span>
                <span className="col-start-2 text-xs text-muted-foreground tabular-nums sm:col-start-3">
                  <span className="text-pass">+{file.added}</span>{" "}
                  <span className="text-fail">−{file.removed}</span>
                </span>
              </li>
            ))}
          </ul>
        </Block>
      ) : null}
      {summary.risks.length > 0 ? (
        <Block name={`Risks noted · ${summary.risks.length}`}>
          <ul className="flex flex-col gap-1.5 text-[13px]">
            {summary.risks.map((risk) => (
              <li key={risk.title} className="flex gap-2">
                <span
                  aria-hidden="true"
                  className={cn(
                    "font-mono font-semibold",
                    levelTone[risk.level]
                  )}
                >
                  !
                </span>
                {risk.title}
              </li>
            ))}
          </ul>
        </Block>
      ) : null}
      {summary.links.length > 0 ? (
        <Block name="Links">
          <ul className="flex flex-wrap gap-3 text-[13px]">
            {summary.links.map((link) => (
              <li key={link.href}>
                <a
                  className="text-primary underline-offset-4 hover:underline"
                  href={link.href}
                >
                  {link.label}
                </a>
              </li>
            ))}
          </ul>
        </Block>
      ) : null}
    </div>
  )
}
