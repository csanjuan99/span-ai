import { ListChecksIcon } from "lucide-react"

import { Panel } from "@/components/shaping/session-ui"
import { contextCounts, questionCounts } from "@/lib/shape-session"
import type { ShapeSession } from "@/lib/types"

function splitText(session: ShapeSession): string {
  const kids = session.children ?? []
  if (kids.length > 0) {
    return `split into ${kids.length} stories`
  }
  if (session.status === "split") {
    return `${session.split.length} stories proposed`
  }
  if (session.splitKept === true) {
    return "kept as one story"
  }
  return session.status === "interviewing" ? "not proposed" : "not needed"
}

function contextText(session: ShapeSession): string {
  const { accepted, proposed, skipped } = contextCounts(session)
  const parts = [
    accepted > 0 ? `${accepted} accepted` : "",
    proposed > 0 ? `${proposed} proposed` : "",
    skipped > 0 ? `${skipped} skipped` : "",
  ].filter((part) => part !== "")
  if (parts.length > 0) {
    return parts.join(" · ")
  }
  return session.status === "interviewing" ? "none yet" : "none"
}

/** Where the session stands, at a glance: the same five lines on every step. */
export function SessionSummary({ session }: { session: ShapeSession }) {
  const questions = questionCounts(session)
  const rows: [string, string][] = [
    ["Rules", String(session.rules.length)],
    ["Examples", String(session.examples.length)],
    [
      "Questions",
      `${questions.open} open · ${questions.answered} answered · ${questions.assumed} assumed`,
    ],
    ["Split", splitText(session)],
    ["Context", contextText(session)],
  ]
  return (
    <Panel aria-label="Session" className="p-4">
      <div className="mb-1 flex items-center gap-2 text-sm font-semibold">
        <ListChecksIcon className="size-4" aria-hidden="true" />
        Session so far
      </div>
      <dl>
        {rows.map(([label, value]) => (
          <div
            key={label}
            className="grid grid-cols-[64px_minmax(0,1fr)] gap-2.5 border-t border-border py-2"
          >
            <dt className="pt-0.5 text-xs font-medium tracking-wide text-muted-foreground">
              {label}
            </dt>
            <dd className="text-[13px] leading-5">{value}</dd>
          </div>
        ))}
      </dl>
      <p className="mt-2 text-xs text-muted-foreground">
        {session.status === "sealed"
          ? "Sealed. Read-only."
          : "A sealed story cannot change. Seal comes last."}
      </p>
    </Panel>
  )
}
