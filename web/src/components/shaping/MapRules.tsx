"use client"

import { cn } from "cn"

import { Panel, Pill } from "@/components/shaping/session-ui"
import { sourceIcons } from "@/components/shaping/source-icons"
import { answerSource } from "@/lib/shape-session"
import type {
  ShapeExample,
  ShapeRule,
  ShapeSession,
  ShapeSource,
} from "@/lib/types"
import { titleText } from "@/components/shaping/styles"

function SourceChip({
  session,
  source,
  selected,
  onPick,
}: {
  session: ShapeSession
  source: ShapeSource
  selected: boolean
  onPick: (() => void) | undefined
}) {
  const Icon = sourceIcons[source.kind]
  const label = answerSource(session, source)
  const body = (
    <>
      <Icon className="size-3.5 shrink-0" aria-hidden="true" />
      <span className="truncate">{label}</span>
    </>
  )
  const style =
    "inline-flex min-h-7 max-w-full items-center gap-1.5 rounded-md border border-input bg-card px-2 py-0.5 text-left font-mono text-xs leading-[18px]"
  if (onPick === undefined) {
    return <span className={style}>{body}</span>
  }
  return (
    <button
      type="button"
      aria-expanded={selected}
      aria-label={`Source: ${label}`}
      className={cn(
        style,
        "cursor-pointer hover:border-primary",
        selected && "border-primary bg-accent"
      )}
      onClick={onPick}
    >
      {body}
    </button>
  )
}

function assumedAnswer(session: ShapeSession, example: ShapeExample): boolean {
  return (
    example.source.kind === "answer" &&
    session.questions.some(
      (question) =>
        question.id === example.source.ref && question.status === "assumed"
    )
  )
}

/**
 * The rules with their examples, each with the source it came from. With `onPick` the sources
 * open in the detail panel; without it (a sealed story) they are plain labels.
 */
export function MapRules({
  session,
  picked,
  onPick,
}: {
  session: ShapeSession
  picked?: string
  onPick?: (id: string, source: ShapeSource) => void
}) {
  const pick = (id: string, source: ShapeSource) =>
    onPick === undefined ? undefined : () => onPick(id, source)
  const loose = session.examples.filter(
    (example) => !session.rules.some((rule) => rule.id === example.rule)
  )
  const examples = (list: ShapeExample[]) =>
    list.map((example) => (
      <div
        key={example.id}
        className="grid grid-cols-[36px_minmax(0,1fr)] gap-x-3 gap-y-1.5 border-t border-border py-2.5 pl-2 sm:grid-cols-[36px_minmax(0,1fr)_auto] sm:pl-6"
      >
        <span className="font-mono text-xs leading-5 font-semibold text-muted-foreground">
          {example.id}
        </span>
        <span className="text-[13px] leading-5">
          {example.text}{" "}
          {assumedAnswer(session, example) ? (
            <Pill tone="hold" small>
              assumed
            </Pill>
          ) : null}
        </span>
        <span className="col-start-2 sm:col-start-3">
          <SourceChip
            session={session}
            source={example.source}
            selected={picked === example.id}
            onPick={pick(example.id, example.source)}
          />
        </span>
      </div>
    ))
  return (
    <>
      {session.rules.map((rule: ShapeRule) => (
        <Panel
          key={rule.id}
          aria-label={`Rule ${rule.id}`}
          className="flex flex-col gap-2 p-4 sm:px-5"
        >
          <div className="grid grid-cols-[36px_minmax(0,1fr)] items-start gap-x-3 gap-y-1.5 sm:grid-cols-[36px_minmax(0,1fr)_auto]">
            <span className="font-mono text-[13px] leading-[22px] font-semibold text-primary">
              {rule.id}
            </span>
            <span className={titleText}>{rule.text}</span>
            <span className="col-start-2 sm:col-start-3">
              <SourceChip
                session={session}
                source={rule.source}
                selected={picked === rule.id}
                onPick={pick(rule.id, rule.source)}
              />
            </span>
          </div>
          {examples(
            session.examples.filter((example) => example.rule === rule.id)
          )}
        </Panel>
      ))}
      {loose.length > 0 ? (
        <Panel aria-label="Examples without a rule" className="p-4 sm:px-5">
          {examples(loose)}
        </Panel>
      ) : null}
    </>
  )
}
