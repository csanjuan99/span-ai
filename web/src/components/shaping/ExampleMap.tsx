"use client"

import type {
  ShapeQuestion,
  ShapeRule,
  ShapeExample,
  ShapeSource,
} from "@/lib/types"
import { RefId } from "@/components/shaping/RefId"
import { PromptHeading } from "@/components/shell/PromptHeading"

/** The evidence behind a fact: where the agent says it read it. */
export function sourceLabel(source: ShapeSource): string {
  return `${source.kind} ${source.ref}`.trim()
}

function Source({ source }: { source: ShapeSource }) {
  return (
    <span className="text-xs text-muted-foreground">{sourceLabel(source)}</span>
  )
}

function ExampleList({ examples }: { examples: ShapeExample[] }) {
  return (
    <ul className="mt-1 ml-4 flex flex-col gap-1">
      {examples.map((example) => (
        <li key={example.id} className="text-sm">
          <RefId id={example.id} />
          <span>{example.text}</span> <Source source={example.source} />
        </li>
      ))}
    </ul>
  )
}

/** Rules with their examples underneath, and the questions already settled. */
export function ExampleMap({
  rules,
  examples,
  questions,
}: {
  rules: ShapeRule[]
  examples: ShapeExample[]
  questions: ShapeQuestion[]
}) {
  const settled = questions.filter((question) => question.status !== "open")
  const loose = examples.filter(
    (example) => !rules.some((rule) => rule.id === example.rule)
  )
  return (
    <section className="flex flex-col gap-3">
      <PromptHeading>example map</PromptHeading>
      {rules.length === 0 ? (
        <p className="text-pretty text-muted-foreground">No rules yet.</p>
      ) : null}
      <ul className="flex flex-col gap-3">
        {rules.map((rule) => (
          <li key={rule.id}>
            <RefId id={rule.id} />
            <span>{rule.text}</span> <Source source={rule.source} />
            <ExampleList
              examples={examples.filter((example) => example.rule === rule.id)}
            />
          </li>
        ))}
      </ul>
      {loose.length > 0 ? <ExampleList examples={loose} /> : null}
      {settled.length > 0 ? (
        <ul className="flex flex-col gap-1 text-sm">
          {settled.map((question) => (
            <li key={question.id}>
              <RefId id={question.id} />
              {question.text}{" "}
              <span className="text-muted-foreground">
                {question.status}: {question.answer}
              </span>
            </li>
          ))}
        </ul>
      ) : null}
    </section>
  )
}
