import { Link } from "react-router"
import {
  ClockIcon,
  DownloadIcon,
  FileDiffIcon,
  FolderIcon,
  HistoryIcon,
  LockIcon,
  UserIcon,
} from "lucide-react"

import { Button } from "@/components/ui/button"
import { IdChip } from "@/components/list-controls/IdChip"
import { Pill, type Tone } from "@/components/shaping/session-ui"
import { whenLabel } from "@/lib/clock"
import type { ShapeSession } from "@/lib/types"
import { wrapRowWide } from "@/components/shaping/styles"
import { touchButtonSm } from "@/lib/button-class"

const badge: Record<ShapeSession["status"], { tone: Tone; label: string }> = {
  interviewing: { tone: "acc", label: "Interviewing" },
  ready: { tone: "hold", label: "Ready" },
  split: { tone: "hold", label: "Split" },
  sealed: { tone: "pass", label: "Sealed" },
}

/** The last folder of a path: what a person calls the project. */
function folderName(path: string): string {
  return path.replace(/\/+$/, "").split("/").pop() || path
}

/** Which agent and model the interview uses. */
export function agentLabel(session: ShapeSession): string {
  const agent =
    session.agent === undefined || session.agent === "" ? "Auto" : session.agent
  const model =
    session.model === undefined || session.model === "" ? "Auto" : session.model
  return `${agent} · ${model}`
}

/** What the session is about, how it stands and where it runs. */
const sessionLinks = [
  { to: "history", label: "History", icon: HistoryIcon },
  { to: "versions", label: "Compare", icon: FileDiffIcon },
  { to: "transfer", label: "Export", icon: DownloadIcon },
]

export function SessionHeader({ session }: { session: ShapeSession }) {
  const state = badge[session.status]
  const sealed = session.status === "sealed"
  const title =
    sealed && session.title !== undefined && session.title !== ""
      ? session.title
      : session.idea
  const when = whenLabel(sealed ? session.sealedAt : session.startedAt)
  return (
    <header className="flex flex-col gap-2">
      <div className={wrapRowWide}>
        <IdChip>
          {session.ref !== undefined && session.ref !== ""
            ? session.ref
            : session.id}
        </IdChip>
        <Pill tone={state.tone}>
          {sealed ? <LockIcon aria-hidden="true" className="size-3" /> : null}
          {state.label}
        </Pill>
        {sealed ? (
          <Pill tone="mut">
            <LockIcon aria-hidden="true" className="size-3" />
            Read-only
          </Pill>
        ) : null}
        <span className="ml-auto flex flex-wrap gap-1">
          {sessionLinks.map((link) => (
            <Button
              key={link.to}
              asChild
              variant="ghost"
              size="sm"
              className={touchButtonSm}
            >
              <Link
                to={`/shaping/${encodeURIComponent(session.id)}/${link.to}?workspace=${encodeURIComponent(session.workspace)}`}
              >
                <link.icon aria-hidden="true" className="size-4" />
                {link.label}
              </Link>
            </Button>
          ))}
        </span>
      </div>
      <h1 className="text-2xl leading-8 font-semibold tracking-[-0.01em] text-balance whitespace-pre-line">
        {title}
      </h1>
      <div className="flex flex-wrap items-center gap-x-4 gap-y-1.5 text-[13px] leading-5 text-muted-foreground">
        <span className="inline-flex items-center gap-1.5">
          <FolderIcon className="size-3.5" aria-hidden="true" />
          <span className="font-mono text-xs break-all">
            {session.repo === "" ? folderName(session.workspace) : session.repo}
          </span>
        </span>
        <span className="inline-flex items-center gap-1.5">
          <UserIcon className="size-3.5" aria-hidden="true" />
          <span className="font-mono text-xs">{agentLabel(session)}</span>
        </span>
        {when !== "" ? (
          <span className="inline-flex items-center gap-1.5 text-xs tabular-nums">
            <ClockIcon className="size-3.5" aria-hidden="true" />
            {sealed ? "sealed" : "started"} {when}
          </span>
        ) : null}
      </div>
    </header>
  )
}
