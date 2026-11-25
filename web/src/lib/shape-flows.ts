import { shapeFile, shapeRequest } from "@/lib/api"
import { isRecord, rows, strings, text, whole } from "@/lib/guards"

const wsQuery = (workspace: string) =>
  `workspace=${encodeURIComponent(workspace)}`
const idPath = (id: string) => `/shapes/${encodeURIComponent(id)}`

/** One session card of the board. */
export interface BoardCard {
  id: string
  ref: string
  title: string
  status: string
  repo: string
  agent: string
  model: string
  openQuestions: number
  assumed: number
  stories: number
  jiraKey: string
  sealedAt: string
  doing: string
}

export interface BoardColumn {
  key: string
  name: string
  count: number
  hidden: number
  cards: BoardCard[]
}

export interface BoardView {
  inFlight: number
  needYou: number
  columns: BoardColumn[]
  repos: string[]
}

function boardCard(row: Record<string, unknown>): BoardCard {
  return {
    id: text(row.id),
    ref: text(row.ref),
    title: text(row.title),
    status: text(row.status),
    repo: text(row.repo),
    agent: text(row.agent),
    model: text(row.model),
    openQuestions: whole(row.openQuestions),
    assumed: whole(row.assumed),
    stories: whole(row.stories),
    jiraKey: text(row.jiraKey),
    sealedAt: text(row.sealedAt),
    doing: text(row.doing),
  }
}

/** Reads the board reply; it is untrusted, so every part has a safe default. */
export function boardView(value: unknown): BoardView {
  const body = isRecord(value) ? value : {}
  return {
    inFlight: whole(body.inFlight),
    needYou: whole(body.needYou),
    repos: strings(body.repos),
    columns: rows(body.columns).map((row) => ({
      key: text(row.key),
      name: text(row.name),
      count: whole(row.count),
      hidden: whole(row.hidden),
      cards: rows(row.cards).map(boardCard),
    })),
  }
}

/** The sessions in a column per status, optionally of one repo. */
export async function getBoard(
  workspace: string,
  repo: string,
  limit = 0
): Promise<BoardView> {
  const query = new URLSearchParams({ workspace, group: "status" })
  if (repo !== "") {
    query.set("repo", repo)
  }
  if (limit > 0) {
    query.set("limit", String(limit))
  }
  return boardView(await shapeRequest("GET", `/shapes?${query.toString()}`))
}

/** One step an agent or the person took. */
export interface Activity {
  at: string
  session: string
  ref: string
  actor: string
  kind: string
  text: string
}

function activity(row: Record<string, unknown>): Activity {
  return {
    at: text(row.at),
    session: text(row.session),
    ref: text(row.ref),
    actor: text(row.actor),
    kind: text(row.kind),
    text: text(row.text),
  }
}

/** The latest steps of the sessions still open. */
export async function getActivity(
  workspace: string,
  who = "agents",
  limit = 12
): Promise<Activity[]> {
  const query = new URLSearchParams({ workspace, who, limit: String(limit) })
  const body = await shapeRequest("GET", `/shapes/activity?${query.toString()}`)
  return Array.isArray(body) ? rows(body).map(activity) : []
}

export type BulkAction = "assume" | "seal" | "archive" | "unarchive"

export interface BulkItem {
  id: string
  ref: string
  ok: boolean
  skipped: boolean
  changed: string[]
  code: string
  message: string
}

export interface BulkResult {
  action: string
  dryRun: boolean
  items: BulkItem[]
  done: number
  skipped: number
  failed: number
}

/** One action on several sessions; a dry run says what would change and changes nothing. */
export async function bulk(
  workspace: string,
  action: BulkAction,
  ids: string[],
  dryRun: boolean
): Promise<BulkResult> {
  const body = await shapeRequest("POST", "/shapes/bulk", {
    workspace,
    action,
    ids,
    dryRun,
  })
  const reply = isRecord(body) ? body : {}
  return {
    action: text(reply.action),
    dryRun: reply.dryRun === true,
    done: whole(reply.done),
    skipped: whole(reply.skipped),
    failed: whole(reply.failed),
    items: rows(reply.items).map((row) => ({
      id: text(row.id),
      ref: text(row.ref),
      ok: row.ok === true,
      skipped: row.skipped === true,
      changed: strings(row.changed),
      code: text(row.code),
      message: text(row.message),
    })),
  }
}

/** One step of a session's history. */
export interface HistoryEvent {
  id: string
  at: string
  actor: string
  kind: string
  summary: string
  ref: string
  undoable: boolean
  why: string
  dependents: string[]
  undoneBy: string
}

export interface HistoryView {
  id: string
  ref: string
  title: string
  status: string
  sealed: boolean
  counts: { everyone: number; you: number; agents: number }
  events: HistoryEvent[]
}

function historyEvent(row: Record<string, unknown>): HistoryEvent {
  return {
    id: text(row.id),
    at: text(row.at),
    actor: text(row.actor),
    kind: text(row.kind),
    summary: text(row.summary),
    ref: text(row.ref),
    undoable: row.undoable === true,
    why: text(row.why),
    dependents: strings(row.dependents),
    undoneBy: text(row.undoneBy),
  }
}

export function historyView(value: unknown): HistoryView {
  const body = isRecord(value) ? value : {}
  const counts = isRecord(body.counts) ? body.counts : {}
  return {
    id: text(body.id),
    ref: text(body.ref),
    title: text(body.title),
    status: text(body.status),
    sealed: body.sealed === true,
    counts: {
      everyone: whole(counts.everyone),
      you: whole(counts.you),
      agents: whole(counts.agents),
    },
    events: rows(body.events).map(historyEvent),
  }
}

export async function getHistory(
  id: string,
  workspace: string,
  actor: string,
  kind: string
): Promise<HistoryView> {
  const query = new URLSearchParams({ workspace })
  if (actor !== "") {
    query.set("actor", actor)
  }
  if (kind !== "") {
    query.set("kind", kind)
  }
  return historyView(
    await shapeRequest("GET", `${idPath(id)}/history?${query.toString()}`)
  )
}

/** What undoing one step changes. */
export interface UndoPlan {
  target: HistoryEvent
  dependents: HistoryEvent[]
  now: string
  after: string
  changes: string[]
  kept: string
  next: string
  sealed: boolean
  undoOnlyThis: boolean
}

export async function getUndoPlan(
  id: string,
  workspace: string,
  step: string
): Promise<UndoPlan> {
  const body = await shapeRequest(
    "GET",
    `${idPath(id)}/history/${encodeURIComponent(step)}?${wsQuery(workspace)}`
  )
  const reply = isRecord(body) ? body : {}
  return {
    target: historyEvent(isRecord(reply.target) ? reply.target : {}),
    dependents: rows(reply.dependents).map(historyEvent),
    now: text(reply.now),
    after: text(reply.after),
    changes: strings(reply.changes),
    kept: text(reply.kept),
    next: text(reply.next),
    sealed: reply.sealed === true,
    undoOnlyThis: reply.undoOnlyThis === true,
  }
}

export async function undoStep(
  id: string,
  workspace: string,
  step: string,
  cascade: boolean
): Promise<string[]> {
  const body = await shapeRequest(
    "POST",
    `${idPath(id)}/history/${encodeURIComponent(step)}/undo`,
    { workspace, cascade }
  )
  return isRecord(body) ? strings(body.undone) : []
}

/** What Span understood of a markdown file or a Jira story, before anything starts. */
export interface ImportRead {
  lines: string[]
  title: string
  bytes: number
}

export async function readMarkdown(
  workspace: string,
  name: string,
  markdown: string
): Promise<ImportRead> {
  const body = await shapeRequest("POST", "/shapes/import", {
    workspace,
    markdown,
    name,
    dryRun: true,
  })
  const reply = isRecord(body) ? body : {}
  return {
    lines: strings(reply.reads),
    title: text(reply.title),
    bytes: whole(reply.bytes),
  }
}

export interface JiraRead {
  summary: string
  status: string
  paragraphs: number
  criteria: number
  comments: string
}

export async function readJira(
  workspace: string,
  key: string
): Promise<JiraRead> {
  const body = await shapeRequest("POST", "/shapes/import", {
    workspace,
    jira: key,
    dryRun: true,
  })
  const reply = isRecord(body) ? body : {}
  return {
    summary: text(reply.summary),
    status: text(reply.status),
    paragraphs: whole(reply.paragraphs),
    criteria: whole(reply.criteria),
    comments: text(reply.comments),
  }
}

/** Starts a session from a Jira story or a markdown file; returns its id. */
export async function importStory(input: {
  workspace: string
  repo: string
  jira?: string
  markdown?: string
  name?: string
}): Promise<string> {
  const body = await shapeRequest("POST", "/shapes/import", input)
  const reply = isRecord(body) ? body : {}
  const session = isRecord(reply.session) ? reply.session : reply
  return text(session.id)
}

export interface ExportChoice {
  format: "md" | "json"
  sources: boolean
  assumptions: boolean
  digest: boolean
}

/** The route that downloads the export, with the options as the server reads them. */
export function exportPath(
  id: string,
  workspace: string,
  choice: ExportChoice
): string {
  const query = new URLSearchParams({ workspace, format: choice.format })
  for (const [name, on] of [
    ["sources", choice.sources],
    ["assumptions", choice.assumptions],
    ["digest", choice.digest],
  ] as const) {
    if (on) {
      query.set(name, "1")
    }
  }
  return `${idPath(id)}/export?${query.toString()}`
}

/** The text of an export, to show before it is saved. */
export async function exportText(
  id: string,
  workspace: string,
  choice: ExportChoice
): Promise<string> {
  return shapeFile(exportPath(id, workspace, choice))
}

export interface ShareLink {
  token: string
  createdAt: string
  expiresAt: string
  opens: number
  revoked: boolean
}

function shareLink(row: Record<string, unknown>): ShareLink {
  return {
    token: text(row.token),
    createdAt: text(row.createdAt),
    expiresAt: text(row.expiresAt),
    opens: whole(row.opens),
    revoked: row.revoked === true,
  }
}

export async function listShares(
  id: string,
  workspace: string
): Promise<ShareLink[]> {
  const body = await shapeRequest(
    "GET",
    `${idPath(id)}/shares?${wsQuery(workspace)}`
  )
  return rows(body).map(shareLink)
}

export async function createShare(
  id: string,
  workspace: string,
  days: number
): Promise<ShareLink> {
  const body = await shapeRequest("POST", `${idPath(id)}/shares`, {
    workspace,
    days,
  })
  return shareLink(isRecord(body) && isRecord(body.link) ? body.link : {})
}

export async function revokeShare(
  id: string,
  workspace: string,
  token: string
): Promise<void> {
  await shapeRequest(
    "DELETE",
    `${idPath(id)}/shares/${encodeURIComponent(token)}?${wsQuery(workspace)}`
  )
}

/** What a read-only link shows. */
export interface SharedStory {
  ref: string
  title: string
  repo: string
  story: string
  digest: string
  sealedAt: string
  rules: { id: string; text: string }[]
  examples: { id: string; rule: string; text: string }[]
}

export async function openShared(
  id: string,
  workspace: string,
  token: string
): Promise<SharedStory> {
  const body = await shapeRequest(
    "GET",
    `${idPath(id)}/shared/${encodeURIComponent(token)}?${wsQuery(workspace)}`
  )
  const reply = isRecord(body) ? body : {}
  return {
    ref: text(reply.ref),
    title: text(reply.title),
    repo: text(reply.repo),
    story: text(reply.story),
    digest: text(reply.digest),
    sealedAt: text(reply.sealedAt),
    rules: rows(reply.rules).map((row) => ({
      id: text(row.id),
      text: text(row.text),
    })),
    examples: rows(reply.examples).map((row) => ({
      id: text(row.id),
      rule: text(row.rule),
      text: text(row.text),
    })),
  }
}
