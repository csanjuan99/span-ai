import { shapeRequest, shapeSession } from "@/lib/api"
import { isRecord, oneOf, rows, text, whole } from "@/lib/guards"
import type { ShapeSession } from "@/lib/types"

/** One saved state of a session, as the list shows it. */
export interface Revision {
  n: number
  id: string
  at: string
  actor: string
  label: string
  digest: string
  status: string
  current: boolean
}

export type ChangeKind = "added" | "removed" | "changed"
export type ItemKind = "rule" | "example" | "question"

/** One rule, example or question that differs between two revisions. */
export interface ItemChange {
  kind: ItemKind
  id: string
  change: ChangeKind
  before: string
  after: string
  /** Where the later version comes from, such as "answer · A2". Empty for a question. */
  source: string
}

export interface Tally {
  added: number
  removed: number
  changed: number
}

/** One line of the story diff. Old and new are line numbers; 0 where the line is absent. */
export interface StoryLine {
  kind: "ctx" | "add" | "del"
  old: number
  new: number
  text: string
}

export interface RevisionDiff {
  from: Revision
  to: Revision
  rules: Tally
  examples: Tally
  questions: Tally
  storyAdded: number
  storyRemoved: number
  items: ItemChange[]
  story: StoryLine[]
  sealed: boolean
}

const changes: readonly ChangeKind[] = ["added", "removed", "changed"]
const itemKinds: readonly ItemKind[] = ["rule", "example", "question"]
const lineKinds: readonly StoryLine["kind"][] = ["ctx", "add", "del"]

/** Reads one revision. The reply is untrusted input, so every part has a safe default. */
export function revision(value: unknown): Revision {
  const body = isRecord(value) ? value : {}
  return {
    n: whole(body.n),
    id: text(body.id),
    at: text(body.at),
    actor: text(body.actor),
    label: text(body.label),
    digest: text(body.digest),
    status: text(body.status),
    current: body.current === true,
  }
}

function tally(value: unknown): Tally {
  const body = isRecord(value) ? value : {}
  return {
    added: whole(body.added),
    removed: whole(body.removed),
    changed: whole(body.changed),
  }
}

/** Reads the diff of two revisions. */
export function revisionDiff(value: unknown): RevisionDiff {
  const body = isRecord(value) ? value : {}
  return {
    from: revision(body.from),
    to: revision(body.to),
    rules: tally(body.rules),
    examples: tally(body.examples),
    questions: tally(body.questions),
    storyAdded: whole(body.storyAdded),
    storyRemoved: whole(body.storyRemoved),
    items: rows(body.items).map((row) => {
      const source = isRecord(row.source) ? row.source : {}
      const kind = text(source.kind)
      const ref = text(source.ref)
      return {
        kind: oneOf(row.kind, itemKinds, "rule"),
        id: text(row.id),
        change: oneOf(row.change, changes, "changed"),
        before: text(row.before),
        after: text(row.after),
        source: kind === "" ? "" : ref === "" ? kind : `${kind} · ${ref}`,
      }
    }),
    story: rows(body.story).map((row) => ({
      kind: oneOf(row.kind, lineKinds, "ctx"),
      old: whole(row.old),
      new: whole(row.new),
      text: text(row.text),
    })),
    sealed: body.sealed === true,
  }
}

/** The revisions of a session, oldest first. */
export async function listRevisions(
  id: string,
  workspace: string
): Promise<Revision[]> {
  const query = new URLSearchParams({ workspace })
  const body = await shapeRequest(
    "GET",
    `/shapes/${encodeURIComponent(id)}/revisions?${query.toString()}`
  )
  return isRecord(body) ? rows(body.revisions).map(revision) : []
}

/** What changed between two revisions of a session. */
export async function getRevisionDiff(
  id: string,
  workspace: string,
  from: string,
  to: string
): Promise<RevisionDiff> {
  const query = new URLSearchParams({ workspace, from, to })
  return revisionDiff(
    await shapeRequest(
      "GET",
      `/shapes/${encodeURIComponent(id)}/diff?${query.toString()}`
    )
  )
}

/** Adds a revision with the content of an earlier one. Nothing is deleted. */
export async function restoreRevision(
  id: string,
  workspace: string,
  rev: string
): Promise<ShapeSession> {
  return shapeSession(
    await shapeRequest(
      "POST",
      `/shapes/${encodeURIComponent(id)}/revisions/${encodeURIComponent(rev)}/restore`,
      {
        workspace,
      }
    )
  )
}

/** Starts a new session from a revision. The session it comes from stays. */
export async function forkRevision(
  id: string,
  workspace: string,
  rev: string
): Promise<ShapeSession> {
  return shapeSession(
    await shapeRequest(
      "POST",
      `/shapes/${encodeURIComponent(id)}/revisions/${encodeURIComponent(rev)}/fork`,
      {
        workspace,
      }
    )
  )
}

/** "1 changed", "2 added · 1 removed", or "no change": how many of one kind of item differ. */
export function tallyText(value: Tally): string {
  const parts: string[] = []
  if (value.added > 0) {
    parts.push(`${value.added} added`)
  }
  if (value.removed > 0) {
    parts.push(`${value.removed} removed`)
  }
  if (value.changed > 0) {
    parts.push(`${value.changed} changed`)
  }
  return parts.length === 0 ? "no change" : parts.join(" · ")
}

/** The two revisions a page shows by default: the last two. */
export function defaultPair(revisions: Revision[]): {
  before: string
  after: string
} {
  const last = revisions[revisions.length - 1]
  const before = revisions[revisions.length - 2]
  return { before: before?.id ?? last?.id ?? "", after: last?.id ?? "" }
}

/** One row of the side-by-side story diff: the earlier line, the later line, or both. */
export interface SideRow {
  left: StoryLine | null
  right: StoryLine | null
}

/**
 * Lines the way two columns show them: unchanged lines sit on both sides, and removed and
 * added lines that follow each other are paired so a rewritten line reads across.
 */
export function sideRows(lines: StoryLine[]): SideRow[] {
  const out: SideRow[] = []
  let at = 0
  while (at < lines.length) {
    const line = lines[at]
    if (line.kind === "ctx") {
      out.push({ left: line, right: line })
      at++
      continue
    }
    const removed: StoryLine[] = []
    const added: StoryLine[] = []
    while (at < lines.length && lines[at].kind !== "ctx") {
      ;(lines[at].kind === "del" ? removed : added).push(lines[at])
      at++
    }
    for (let i = 0; i < Math.max(removed.length, added.length); i++) {
      out.push({ left: removed[i] ?? null, right: added[i] ?? null })
    }
  }
  return out
}

/** Whether a restore is possible: not for a sealed story, and not for the revision already current. */
export function canRestore(diff: RevisionDiff, target: Revision): boolean {
  return !diff.sealed && !target.current
}
