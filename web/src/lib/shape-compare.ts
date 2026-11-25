import { shapeRequest, shapeSession } from "@/lib/api"
import { isRecord, rows, strings, text, whole } from "@/lib/guards"
import type { ShapeSession } from "@/lib/types"

/** One session of a comparison. */
export interface CompareSession {
  id: string
  ref: string
  title: string
  status: string
  sealed: boolean
  items: number
  unique: number
  mergedInto: string
}

/** One statement of one session. Origin says who settled it: you, the agent's assumption, or a seal. */
export interface Member {
  session: string
  ref: string
  item: string
  kind: string
  text: string
  origin: string
  sealed: boolean
}

export interface SameGroup {
  kind: string
  text: string
  members: Member[]
}

export interface ConflictOption {
  text: string
  members: Member[]
}

export interface Conflict {
  id: string
  title: string
  options: ConflictOption[]
}

export interface Duplicate {
  a: string
  b: string
  aRef: string
  bRef: string
  percent: number
  aItems: string[]
  bItems: string[]
  dismissed: boolean
}

export interface Comparison {
  repo: string
  sessions: CompareSession[]
  same: SameGroup[]
  conflicts: Conflict[]
  duplicates: Duplicate[]
  unique: number
  /** Always true: matching is by shared words, so the page says so. */
  estimate: boolean
}

function member(row: Record<string, unknown>): Member {
  return {
    session: text(row.session),
    ref: text(row.ref),
    item: text(row.item),
    kind: text(row.kind),
    text: text(row.text),
    origin: text(row.origin),
    sealed: row.sealed === true,
  }
}

/** Reads a comparison. The reply is untrusted input, so every part has a safe default. */
export function comparison(value: unknown): Comparison {
  const body = isRecord(value) ? value : {}
  return {
    repo: text(body.repo),
    sessions: rows(body.sessions).map((row) => ({
      id: text(row.id),
      ref: text(row.ref),
      title: text(row.title),
      status: text(row.status),
      sealed: row.sealed === true,
      items: whole(row.items),
      unique: whole(row.unique),
      mergedInto: text(row.mergedInto),
    })),
    same: rows(body.same).map((row) => ({
      kind: text(row.kind),
      text: text(row.text),
      members: rows(row.members).map(member),
    })),
    conflicts: rows(body.conflicts).map((row) => ({
      id: text(row.id),
      title: text(row.title),
      options: rows(row.options).map((option) => ({
        text: text(option.text),
        members: rows(option.members).map(member),
      })),
    })),
    duplicates: rows(body.duplicates).map((row) => ({
      a: text(row.a),
      b: text(row.b),
      aRef: text(row.aRef),
      bRef: text(row.bRef),
      percent: whole(row.percent),
      aItems: strings(row.aItems),
      bItems: strings(row.bItems),
      dismissed: row.dismissed === true,
    })),
    unique: whole(body.unique),
    estimate: body.estimate !== false,
  }
}

/** What two to five sessions of one repo share and where they disagree. Nothing is written. */
export async function compareShapes(
  workspace: string,
  ids: string[]
): Promise<Comparison> {
  const query = new URLSearchParams({ workspace, ids: ids.join(",") })
  return comparison(
    await shapeRequest("GET", `/shapes/compare?${query.toString()}`)
  )
}

/** Records that these sessions stay separate. */
export async function keepSeparate(
  workspace: string,
  ids: string[]
): Promise<Comparison> {
  return comparison(
    await shapeRequest("POST", "/shapes/compare/dismiss", { workspace, ids })
  )
}

/** What a merge did. */
export interface MergeOutcome {
  into: ShapeSession
  moved: number
  answers: number
  replaced: number
  merged: string[]
}

/** Moves the unique items of the `from` sessions into `into`. Every conflict needs its decision in `resolve`. */
export async function mergeShapes(
  workspace: string,
  input: {
    into: string
    from: string[]
    against: string[]
    resolve: Record<string, string>
  }
): Promise<MergeOutcome> {
  const body = await shapeRequest("POST", "/shapes/merge", {
    workspace,
    ...input,
  })
  const reply = isRecord(body) ? body : {}
  return {
    into: shapeSession(reply.into),
    moved: whole(reply.moved),
    answers: whole(reply.answers),
    replaced: whole(reply.replaced),
    merged: strings(reply.merged),
  }
}

/** The session a conflict option belongs to, by the first member that says it. */
export function optionOwner(option: ConflictOption): string {
  return option.members[0]?.session ?? ""
}

/** The conflicts that still have no decision. */
export function undecided(
  conflicts: Conflict[],
  picks: Record<string, string>
): string[] {
  return conflicts
    .filter((conflict) => picks[conflict.id] === undefined)
    .map((conflict) => conflict.id)
}

/** The command that runs a merge with the decisions made so far. */
export function mergeCommand(
  into: string,
  from: string[],
  against: string[],
  picks: Record<string, string>
): string {
  const parts = ["span shape merge", into, ...from]
  if (against.length > 0) {
    parts.push(`--against ${against.join(",")}`)
  }
  const resolve = Object.entries(picks).map(
    ([id, session]) => `${id}=${session}`
  )
  if (resolve.length > 0) {
    parts.push(`--resolve ${resolve.join(",")}`)
  }
  parts.push("--yes")
  return parts.join(" ")
}

/**
 * The session the merge goes into: the one that is ready, else the first that is not sealed.
 * A sealed session is only compared, never changed.
 */
export function suggestedTarget(
  sessions: CompareSession[],
  chosen: string[]
): string {
  const open = sessions.filter(
    (session) =>
      chosen.includes(session.id) &&
      !session.sealed &&
      session.mergedInto === ""
  )
  return (
    (open.find((session) => session.status === "ready") ?? open[0])?.id ?? ""
  )
}
