import { cn } from "cn"

import { Panel, toneText, type Tone } from "@/components/shaping/session-ui"
import type { ChangeKind, ItemChange, ItemKind } from "@/lib/shape-versions"
import {
  cardBody,
  labelText,
  lineNumberCell,
  proseText,
  wrapRow,
} from "@/components/shaping/styles"

const marks: Record<ChangeKind, { mark: string; tone: Tone; word: string }> = {
  added: { mark: "+", tone: "pass", word: "Added" },
  removed: { mark: "−", tone: "fail", word: "Removed" },
  changed: { mark: "~", tone: "hold", word: "Changed" },
}

const kindWord: Record<ItemKind, string> = {
  rule: "Rule",
  example: "Example",
  question: "Question",
}

/** The rows of a small before and after, as the story diff draws them. */
export function DiffRows({
  rows,
}: {
  rows: { sign: "+" | "-" | " "; label: string; text: string }[]
}) {
  return (
    <div
      role="region"
      aria-label="Code"
      tabIndex={0}
      className="overflow-x-auto rounded-[10px] border border-border bg-surface-code py-2.5 font-mono text-[13px] leading-5 focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none"
    >
      {rows.map((row, at) => (
        <div
          key={at}
          className={cn(
            "grid min-w-max grid-cols-[36px_16px_minmax(0,1fr)] pr-3.5 whitespace-pre",
            row.sign === "+" && "bg-diff-add",
            row.sign === "-" && "bg-diff-del",
            row.sign === " " && "text-muted-foreground"
          )}
        >
          <span className={lineNumberCell}>{row.label}</span>
          <span aria-hidden="false">{row.sign}</span>
          <span>{row.text}</span>
        </div>
      ))}
    </div>
  )
}

function Side({
  title,
  text,
  emphasis,
}: {
  title: string
  text: string
  emphasis: boolean
}) {
  return (
    <div
      className={cn(
        "flex flex-col gap-1 rounded-[10px] border bg-muted px-3 py-2.5",
        emphasis ? "border-primary" : "border-border"
      )}
    >
      <div
        className={cn(
          labelText,
          emphasis ? "text-primary" : "text-muted-foreground"
        )}
      >
        {title}
      </div>
      <div className={proseText}>{text}</div>
    </div>
  )
}

/** One rule, example or question that changed: before and after side by side, or as a two-line diff. */
export function ChangeCard({
  item,
  from,
  to,
  view,
}: {
  item: ItemChange
  from: string
  to: string
  view: "side" | "inline"
}) {
  const state = marks[item.change]
  const label = `${kindWord[item.kind]} ${item.id}`
  return (
    <Panel aria-label={label} className={cardBody}>
      <div className={wrapRow}>
        <span
          aria-hidden="true"
          className={cn("font-mono font-semibold", toneText[state.tone])}
        >
          {state.mark}
        </span>
        <span className="text-sm font-semibold">
          {state.word}: {label}
        </span>
        {item.source !== "" ? (
          <span className="ml-auto inline-flex min-h-7 items-center rounded-md border border-input bg-card px-2 font-mono text-xs">
            {item.source}
          </span>
        ) : null}
      </div>
      {view === "side" ? (
        <div className="grid gap-3 sm:grid-cols-2">
          <Side
            title={`${from} · before`}
            text={item.before === "" ? `Not in ${from}.` : item.before}
            emphasis={false}
          />
          <Side
            title={`${to} · after`}
            text={item.after === "" ? `Not in ${to}.` : item.after}
            emphasis
          />
        </div>
      ) : (
        <DiffRows
          rows={[
            ...(item.before === ""
              ? []
              : [{ sign: "-" as const, label: from, text: item.before }]),
            ...(item.after === ""
              ? []
              : [{ sign: "+" as const, label: to, text: item.after }]),
          ]}
        />
      )}
    </Panel>
  )
}
