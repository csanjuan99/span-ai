"use client"

import { useEffect, useState } from "react"

import { getShapeSource } from "@/lib/api"
import { reasonText } from "@/lib/guards"
import { CopyChip } from "@/components/ledger/CopyChip"
import { CodeBlock, Panel } from "@/components/shaping/session-ui"
import { answerSource, parseSourceRef } from "@/lib/shape-session"
import type { ShapeSession, ShapeSnippet, ShapeSource } from "@/lib/types"
import { cn } from "@/lib/utils"
import { errorText, lineNumberCell } from "@/components/shaping/styles"

type Loaded = { ref: string; snippet: ShapeSnippet | null; error: string }

/** The words behind a source that is not code: the idea, the answer, or the context note. */
function describe(session: ShapeSession, source: ShapeSource): string {
  if (source.kind === "prompt") {
    return `Your idea: ${session.idea}`
  }
  if (source.kind === "answer") {
    const question = session.questions.find((item) => item.id === source.ref)
    return question === undefined
      ? `Answer ${source.ref}`
      : `${question.text} → ${question.answer}${question.status === "assumed" ? " (assumed)" : ""}`
  }
  return `From the product context: ${source.ref}`
}

/** What a picked source points at. A code source shows the lines around it, read now. */
export function SourceDetail({
  session,
  id,
  source,
}: {
  session: ShapeSession
  id: string
  source: ShapeSource
}) {
  const code = parseSourceRef(source)
  const ref = code === null ? "" : `${code.path}:${code.line}`
  const [loaded, setLoaded] = useState<Loaded | null>(null)

  useEffect(() => {
    if (ref === "") {
      return
    }
    let live = true
    void getShapeSource(session.id, session.workspace, ref)
      .then((snippet) => {
        if (live) {
          setLoaded({ ref, snippet, error: "" })
        }
      })
      .catch((err: unknown) => {
        if (live) {
          setLoaded({
            ref,
            snippet: null,
            error: reasonText(err, "the source could not be read"),
          })
        }
      })
    return () => {
      live = false
    }
  }, [ref, session.id, session.workspace])

  const current = loaded !== null && loaded.ref === ref ? loaded : null
  return (
    <Panel aria-label="Source detail" className="flex flex-col gap-2 p-4">
      <div className="text-sm font-semibold">{`Source for ${id}`}</div>
      <div className="font-mono text-xs break-all">
        {answerSource(session, source)}
      </div>
      {code === null ? (
        <p className="text-sm text-pretty">{describe(session, source)}</p>
      ) : current === null ? (
        <p className="text-sm text-muted-foreground">Reading…</p>
      ) : current.snippet === null ? (
        <p role="alert" className={errorText}>
          {current.error}
        </p>
      ) : (
        <>
          <CodeBlock label="Code">
            {current.snippet.lines.map((line) => (
              <div
                key={line.n}
                className={cn(
                  "grid grid-cols-[36px_minmax(0,1fr)] pr-3.5 whitespace-pre",
                  line.hit && "bg-accent"
                )}
              >
                <span className={lineNumberCell}>{line.n}</span>
                <span>{line.text}</span>
              </div>
            ))}
          </CodeBlock>
          <CopyChip label="source" value={ref}>
            {ref}
          </CopyChip>
          <p className="text-xs text-muted-foreground">
            Shown as the file is now, not as the agent read it.
          </p>
        </>
      )}
    </Panel>
  )
}
