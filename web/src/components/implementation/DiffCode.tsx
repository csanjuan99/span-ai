"use client"

import { useState } from "react"

import { cn } from "@/lib/utils"
import { foldContext } from "@/lib/diff-fold"
import type { DiffFile, DiffLine, Hunk } from "@/lib/diff"
import { languageOf, tokenize, type TokenKind } from "@/lib/syntax"

const tokenTone: Record<TokenKind, string> = {
  plain: "",
  keyword: "text-code-keyword",
  string: "text-code-string",
  number: "text-code-number",
  function: "text-code-function",
  comment: "text-code-comment",
}

const rowTone: Record<DiffLine["kind"], string> = {
  add: "bg-diff-add",
  del: "bg-diff-del",
  ctx: "",
  note: "italic text-muted-foreground",
}

const sign: Record<DiffLine["kind"], string> = {
  add: "+",
  del: "−",
  ctx: " ",
  note: " ",
}

function Row({
  line,
  language,
}: {
  line: DiffLine
  language: ReturnType<typeof languageOf>
}) {
  const number = line.kind === "del" ? line.oldNo : line.newNo
  return (
    <div
      className={cn(
        "grid grid-cols-[52px_16px_minmax(0,1fr)] pr-3.5 whitespace-pre",
        rowTone[line.kind]
      )}
    >
      <span
        aria-hidden="true"
        className="pr-2 text-right text-muted-foreground tabular-nums select-none"
      >
        {number ?? ""}
      </span>
      <span aria-hidden="true" className="text-center select-none">
        {sign[line.kind]}
      </span>
      <span className={cn(line.kind === "ctx" && "text-code-comment")}>
        {line.kind === "note"
          ? line.text
          : tokenize(line.text, language).map((token, index) => (
              <span key={index} className={tokenTone[token.kind]}>
                {token.text}
              </span>
            ))}
      </span>
    </div>
  )
}

function HunkView({
  hunk,
  language,
}: {
  hunk: Hunk
  language: ReturnType<typeof languageOf>
}) {
  const [open, setOpen] = useState<Set<number>>(new Set())
  return (
    <>
      <div className="grid grid-cols-[52px_16px_minmax(0,1fr)] bg-muted pr-3.5 whitespace-pre text-muted-foreground">
        <span />
        <span />
        <span>{hunk.header}</span>
      </div>
      {foldContext(hunk.lines).map((segment, index) =>
        segment.kind === "lines" || open.has(index) ? (
          segment.lines.map((line, at) => (
            <Row key={`${index}-${at}`} line={line} language={language} />
          ))
        ) : (
          <div
            key={index}
            className="grid grid-cols-[52px_16px_minmax(0,1fr)] bg-muted pr-3.5"
          >
            <span />
            <span />
            <button
              type="button"
              className="min-h-7 text-left text-primary hover:underline focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none max-md:min-h-11 md:min-h-7"
              onClick={() => setOpen((now) => new Set(now).add(index))}
            >
              ⋯ Show {segment.lines.length} unchanged lines
            </button>
          </div>
        )
      )}
    </>
  )
}

/**
 * The hunks of one file as numbered rows with the code coloured by language. A long run of
 * unchanged lines is folded and opens on a click.
 */
export function DiffCode({ file }: { file: DiffFile }) {
  const language = languageOf(file.path)
  return (
    <div
      role="region"
      aria-label={`Diff of ${file.path}`}
      className="overflow-x-auto rounded-[10px] border border-border bg-surface-code py-2 font-mono text-[13px] leading-5"
    >
      <div className="min-w-max">
        {file.hunks.map((hunk) => (
          <HunkView key={hunk.header} hunk={hunk} language={language} />
        ))}
      </div>
    </div>
  )
}
