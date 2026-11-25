import { isRecord, rows, text } from "@/lib/guards"
import {
  conflictActions,
  conflictKinds,
  conflictRisks,
  conflictStates,
  shapeProposalStates,
  shapeQuestionStates,
  shapeSourceKinds,
  shapeStates,
  type AcceptanceView,
  type Conflict,
  type ConflictGroup,
  type ConflictOption,
  type Cursor,
  type DecisionRequest,
  type Gate,
  type ImplementationView,
  type LiveFrame,
  type ModelList,
  type ModelOption,
  type QACase,
  type Snapshot,
  type Statement,
  type StatementCommand,
  type AgentChoice,
  type AgentState,
  type BranchChoice,
  type AgentStatus,
  type SpanHead,
  type ShapeExample,
  type ShapeJira,
  type ShapePreview,
  type ShapeProposal,
  type ShapeQuestion,
  type ShapeRule,
  type ShapeSession,
  type ShapeSource,
  type ShapeSplit,
} from "@/lib/types"

/** One run a submission started. A submission over several repos starts one per repo. */
export interface StartedRun {
  id: string
  stage: string
  /** The repo the run works in; empty when the server did not say. */
  repo: string
}

/** What the desk gets back from starting a story. */
export interface Started {
  /** The first run, for a caller that follows one. */
  id: string
  stage: string
  /** Every run that started, the first included. */
  runs: StartedRun[]
  /** Some repos started and some did not; `notes` says which. */
  partial: boolean
  notes: string
}

function count(value: unknown): number {
  return typeof value === "number" ? value : 0
}

function flag(value: unknown): boolean {
  return value === true
}

function messageOf(value: unknown): string {
  if (!isRecord(value)) {
    return "request failed"
  }
  const message = value.message
  if (typeof message === "string" && message !== "") {
    return message
  }
  return "request failed"
}

function parseJSON(raw: string): unknown {
  try {
    const value: unknown = JSON.parse(raw)
    return value
  } catch {
    throw new Error("the response was not JSON")
  }
}

const tokenKey = "span.serveToken"

/** The serve token kept for this browser tab. Localhost leaves it empty. */
export function serveToken(): string {
  return sessionStorage.getItem(tokenKey) ?? ""
}

/** Remembers the serve token. An empty value clears it. */
export function rememberServeToken(value: string): void {
  if (value === "") {
    sessionStorage.removeItem(tokenKey)
    return
  }
  sessionStorage.setItem(tokenKey, value)
}

/** The server refused the request because it needs its access token, or the one sent is wrong. */
export class UnauthorizedError extends Error {
  constructor() {
    super("This server needs its access token. Add it under Settings.")
    this.name = "UnauthorizedError"
  }
}

/** Stops before the body is read: the refusal is plain text, not the error JSON. */
function refuseUnauthorized(response: Response): void {
  if (response.status === 401) {
    throw new UnauthorizedError()
  }
}

function withAuth(headers: Headers): Headers {
  const value = serveToken()
  if (value !== "") {
    headers.set("Authorization", `Bearer ${value}`)
  }
  return headers
}

/** The socket protocol coder/websocket expects when SPAN_TOKEN is set. */
export function socketProtocol(): string {
  const value = serveToken()
  if (value === "") {
    return ""
  }
  const bytes = new TextEncoder().encode(value)
  let binary = ""
  for (const byte of bytes) {
    binary += String.fromCharCode(byte)
  }
  const encoded = btoa(binary).replaceAll("+", "-").replaceAll("/", "_").replace(/=+$/u, "")
  return `span.${encoded}`
}

function command(value: unknown): StatementCommand {
  if (!isRecord(value)) {
    return { command: "", exitCode: 0 }
  }
  const error = text(value.error)
  const tests = isRecord(value.tests)
    ? { passed: count(value.tests.passed), failed: count(value.tests.failed), skipped: count(value.tests.skipped) }
    : undefined
  return {
    command: text(value.command),
    exitCode: count(value.exitCode),
    ...(error === "" ? {} : { error }),
    ...(tests === undefined ? {} : { tests }),
  }
}

function statement(value: unknown): Statement {
  if (!isRecord(value)) {
    throw new Error("statement is missing")
  }
  const commands = Array.isArray(value.commands) ? value.commands.map(command) : []
  return {
    stage: text(value.stage),
    iteration: count(value.iteration),
    verdict: text(value.verdict),
    reason: text(value.reason),
    time: text(value.time),
    parentDigest: text(value.parentDigest),
    gitSHA: text(value.gitSHA),
    changeURL: text(value.changeURL),
    commands,
    digest: text(value.digest),
  }
}

/** The first allowed value equal to the input, or the fallback. Narrows without a cast. */
function oneOf<T extends string>(value: unknown, allowed: readonly T[], fallback: T): T {
  return allowed.find((item) => item === value) ?? fallback
}

function conflictOption(value: unknown): ConflictOption | undefined {
  if (!isRecord(value) || text(value.id) === "") {
    return undefined
  }
  return {
    id: text(value.id),
    title: text(value.title),
    summary: text(value.summary),
    risk: oneOf(value.risk, conflictRisks, "high"),
    action: oneOf(value.action, conflictActions, "keep"),
    command: text(value.command),
    diff: text(value.diff),
  }
}

function present<T>(item: T | undefined): item is T {
  return item !== undefined
}

function conflictGroup(value: unknown): ConflictGroup | undefined {
  if (!isRecord(value) || text(value.id) === "") {
    return undefined
  }
  const options = Array.isArray(value.options) ? value.options.map(conflictOption).filter(present) : []
  if (options.length === 0) {
    return undefined
  }
  return {
    id: text(value.id),
    kind: oneOf(value.kind, conflictKinds, "other"),
    files: stringList(value.files),
    summary: text(value.summary),
    options,
    chosen: text(value.chosen),
  }
}

/**
 * Reads the conflict the merge stage is resolving. Unknown enums degrade to the safe
 * reading instead of throwing, and a choosing conflict without one usable group is
 * dropped, so the page never offers an empty choice. The other states have no cards yet
 * or anymore, so they keep an empty list.
 */
export function conflictOf(value: unknown): Conflict | undefined {
  if (!isRecord(value)) {
    return undefined
  }
  const state = oneOf(value.state, conflictStates, "proposing")
  const groups = Array.isArray(value.groups) ? value.groups.map(conflictGroup).filter(present) : []
  if (state === "choosing" && groups.length === 0) {
    return undefined
  }
  return { base: text(value.base), baseSHA: text(value.baseSHA), state, note: text(value.note), groups }
}

function cursor(value: unknown): Cursor {
  if (!isRecord(value)) {
    throw new Error("cursor is missing")
  }
  const conflict = conflictOf(value.conflict)
  const decision = decisionOf(value.decision)
  return {
    ...(conflict === undefined ? {} : { conflict }),
    ...(decision === undefined ? {} : { decision }),
    id: text(value.id),
    stage: text(value.stage),
    iteration: count(value.iteration),
    hold: flag(value.hold),
    stop: flag(value.stop),
    done: flag(value.done),
    reason: text(value.reason),
    gitSHA: text(value.gitSHA),
    changeURL: text(value.changeURL),
    branch: text(value.branch),
    base: text(value.base),
    workspace: text(value.workspace),
    issue: text(value.issue),
    questions: stringList(value.questions),
  }
}

/**
 * Reads the decision a run waits on. A request with fewer than two usable options is
 * dropped: the page would show a question nobody can answer.
 */
export function decisionOf(value: unknown): DecisionRequest | undefined {
  if (!isRecord(value)) {
    return undefined
  }
  const options = Array.isArray(value.options)
    ? value.options.filter(isRecord).map((option) => ({
        id: text(option.id),
        label: text(option.label),
        consequence: text(option.consequence),
        reversible: flag(option.reversible),
      }))
    : []
  const usable = options.filter((option) => option.id !== "" && option.label !== "")
  if (usable.length < 2) {
    return undefined
  }
  const evidence = Array.isArray(value.evidence)
    ? value.evidence.filter(isRecord).map((item) => ({ ref: text(item.ref), excerpt: text(item.excerpt) }))
    : []
  return {
    id: text(value.id),
    class: text(value.class),
    stage: text(value.stage),
    title: text(value.title),
    happened: text(value.happened),
    evidence,
    options: usable,
    recommendation: text(value.recommendation),
  }
}

function stringList(value: unknown): string[] {
  if (!Array.isArray(value)) {
    return []
  }
  return value.filter((item): item is string => typeof item === "string" && item !== "")
}

function gate(value: unknown): Gate | null {
  if (value == null) {
    return null
  }
  if (!isRecord(value)) {
    throw new Error("gate is missing")
  }
  return {
    verdict: text(value.verdict),
    command: text(value.command),
    exitCode: count(value.exitCode),
  }
}

/** Reads a `GET /branches` reply. It is untrusted input, so only non-empty names survive. */
export function branchChoice(value: unknown): BranchChoice {
  if (!isRecord(value)) {
    return { branches: [], defaultBase: "dev" }
  }
  const fallback = text(value.default)
  return { branches: stringList(value.branches), defaultBase: fallback === "" ? "dev" : fallback }
}

/** What an empty Base branch field means for the chosen kind: span.yaml's base never applies to a hotfix, which starts from master. */
export function basePlaceholder(kind: string, defaultBase: string): string {
  return kind === "hotfix" ? "master" : defaultBase
}

/** The branch a run works on and the base its change opens into, for the span header. */
export function branchLine(cursor: Pick<Cursor, "branch" | "base">): string {
  const parts = [cursor.branch === "" ? "" : `branch ${cursor.branch}`, cursor.base === "" ? "" : `base ${cursor.base}`]
  return parts.filter((part) => part !== "").join(" · ")
}

/** Reads the span object the socket and the page share. */
export function snapshot(value: unknown): Snapshot {
  if (!isRecord(value)) {
    throw new Error("span payload is not a snapshot")
  }
  const statements = Array.isArray(value.statements) ? value.statements.map(statement) : []
  const acceptance = Array.isArray(value.acceptance)
    ? value.acceptance.filter(isRecord).map((line) => ({ id: text(line.id), text: text(line.text) }))
    : []
  return {
    cursor: cursor(value.cursor),
    statements,
    acceptance,
    gate: gate(value.gate),
  }
}

function qaCase(value: unknown): QACase {
  if (!isRecord(value)) {
    throw new Error("acceptance case is missing")
  }
  return {
    id: text(value.id),
    text: text(value.text),
    result: text(value.result),
    file: text(value.file),
    line: count(value.line),
  }
}

/** Reads the QA tab payload. */
export function acceptanceView(value: unknown): AcceptanceView {
  if (!isRecord(value)) {
    throw new Error("acceptance payload is missing")
  }
  return {
    skipped: flag(value.skipped),
    verdict: text(value.verdict),
    command: text(value.command),
    exitCode: count(value.exitCode),
    lines: Array.isArray(value.lines) ? value.lines.map(qaCase) : [],
  }
}

/** Reads the implementation tab payload. */
export function implementationView(value: unknown): ImplementationView {
  if (!isRecord(value)) {
    throw new Error("implementation payload is missing")
  }
  return {
    stat: text(value.stat),
    patch: text(value.patch),
    truncated: flag(value.truncated),
    commits: count(value.commits),
  }
}

/** Reads one live frame. Throws on a non-object; the socket handler treats that as noise. */
export function liveFrame(value: unknown): LiveFrame {
  if (!isRecord(value)) {
    throw new Error("live frame is missing")
  }
  const frame: LiveFrame = { type: text(value.type) }
  if (value.cursor != null) {
    frame.cursor = cursor(value.cursor)
  }
  if (value.statement != null) {
    frame.statement = statement(value.statement)
  }
  if (value.gate != null) {
    frame.gate = gate(value.gate) ?? undefined
  }
  if (typeof value.stage === "string") {
    frame.stage = value.stage
  }
  if (Array.isArray(value.lines)) {
    frame.lines = value.lines.filter((line): line is string => typeof line === "string")
  }
  if (Array.isArray(value.transcript)) {
    frame.transcript = value.transcript.filter((line): line is string => typeof line === "string")
  }
  if (value.span != null) {
    frame.span = snapshot(value.span)
  }
  const conflict = conflictOf(value.conflict)
  if (conflict !== undefined) {
    frame.conflict = conflict
  }
  if (isWholeCount(value.seq)) {
    frame.seq = value.seq
  }
  if (isWholeCount(value.offset)) {
    frame.offset = value.offset
  }
  return frame
}

function isWholeCount(value: unknown): value is number {
  return typeof value === "number" && Number.isSafeInteger(value) && value >= 0
}

async function readBody(response: Response): Promise<unknown> {
  const raw = await response.text()
  if (raw === "") {
    return null
  }
  return parseJSON(raw)
}

/** Fetches JSON and surfaces the server message beside the action that sent it. */
export async function getJSON(url: string): Promise<unknown> {
  let response: Response
  try {
    response = await fetch(url, { headers: withAuth(new Headers({ Accept: "application/json" })) })
  } catch (err: unknown) {
    // The request never got an answer: the server is down or on another port.
    throw new ApiError("unreachable", err instanceof Error ? err.message : "the server did not answer")
  }
  refuseUnauthorized(response)
  const body = await readBody(response)
  if (!response.ok) {
    throw new ApiError(isRecord(body) ? text(body.error) : "", messageOf(body))
  }
  return body
}

/** Keeps the entries of a `runs` reply that are runs. The reply is untrusted input. */
function startedRuns(value: unknown): StartedRun[] {
  if (!Array.isArray(value)) {
    return []
  }
  const items: unknown[] = value
  const runs: StartedRun[] = []
  for (const item of items) {
    if (isRecord(item) && typeof item.id === "string" && typeof item.stage === "string") {
      runs.push({ id: item.id, stage: item.stage, repo: text(item.repo) })
    }
  }
  return runs
}

/** Posts the desk form. A 202 returns the new span. Other success is configuration text. */
export async function startRun(form: FormData): Promise<Started | string> {
  const response = await fetch("/quickstart", { method: "POST", body: form, headers: withAuth(new Headers()) })
  refuseUnauthorized(response)
  const raw = await response.text()
  if (response.status === 202) {
    const body = parseJSON(raw)
    if (!isRecord(body) || typeof body.id !== "string" || typeof body.stage !== "string") {
      throw new Error("quickstart did not return a span id")
    }
    const listed = startedRuns(body.runs)
    const runs = listed.length > 0 ? listed : [{ id: body.id, stage: body.stage, repo: "" }]
    return { id: body.id, stage: body.stage, runs, partial: body.partial === true, notes: text(body.notes) }
  }
  if (!response.ok) {
    throw new Error(failureText(raw))
  }
  return raw
}

/** The error JSON, or a body that was not JSON read as its own message. */
function errorBody(raw: string): unknown {
  try {
    return parseJSON(raw)
  } catch {
    return { message: raw || "request failed" }
  }
}

/** The message a failed response carries, or the raw body when it is not the error JSON. */
function failureText(raw: string): string {
  return messageOf(errorBody(raw))
}

async function postJSON(url: string, body: unknown): Promise<void> {
  const response = await fetch(url, {
    method: "POST",
    headers: withAuth(new Headers({ "Content-Type": "application/json", Accept: "application/json" })),
    body: body === undefined ? undefined : JSON.stringify(body),
  })
  refuseUnauthorized(response)
  if (response.status === 202 || response.ok) {
    return
  }
  const raw = await response.text()
  const message = failureText(raw)
  throw new Error(message)
}

/** Sends the answers and continues the waiting run. */
export async function answerRun(id: string, answers: { question: string; body: string }[]): Promise<void> {
  await postJSON(`/spans/${id}/answer`, { answers })
}

/** Answers the decision a run waits on and continues it where the answer says. */
export async function decideRun(id: string, decision: string, option: string, text: string): Promise<void> {
  await postForm(`/spans/${encodeURIComponent(id)}/decide`, new URLSearchParams({ decision, option, text }))
}

/** Reads the Jira story again and continues the waiting run. */
export async function readStory(id: string): Promise<void> {
  await postJSON(`/spans/${id}/story`, undefined)
}

/** Continues a paused run that is not waiting for an answer. */
export async function resumeRun(id: string): Promise<void> {
  await postJSON(`/spans/${id}/resume`, undefined)
}

/** Approves a stage held by its mode and continues the run; the server binds it to the commit shown. */
export async function approveStage(id: string, stage: string, environment: string): Promise<void> {
  const form = new URLSearchParams({ stage })
  if (environment !== "") {
    form.set("environment", environment)
  }
  await postForm(`/spans/${id}/approve`, form)
}

/** Starts a run again when this process no longer has its worker. */
export async function relaunchRun(id: string): Promise<void> {
  await postJSON(`/spans/${id}/relaunch`, undefined)
}

/** Reads one agent review. */
export function agentStatus(value: unknown): AgentStatus {
  if (!isRecord(value)) {
    throw new Error("agent status is missing")
  }
  return {
    id: text(value.id),
    running: flag(value.running),
    relaunch: flag(value.relaunch),
    name: text(value.name),
    pid: count(value.pid),
    stage: text(value.stage),
    elapsed: text(value.elapsed),
    summary: text(value.summary),
  }
}

/** Reads the agents running in this process. */
export function agentList(value: unknown): AgentStatus[] {
  if (!isRecord(value) || !Array.isArray(value.agents)) {
    return []
  }
  return value.agents.map((item) => agentStatus(item))
}

/** Posts a stage instruction and leaves the span page in place. */
export async function saveInstruction(id: string, stage: string, body: string): Promise<void> {
  const data = new URLSearchParams()
  data.set("stage", stage)
  data.set("body", body)
  const response = await fetch(`/spans/${id}/instructions`, {
    method: "POST",
    headers: withAuth(new Headers({ "Content-Type": "application/x-www-form-urlencoded" })),
    body: data,
    redirect: "manual",
  })
  if (response.status === 303 || response.status === 0 || response.ok) {
    return
  }
  const raw = await response.text()
  throw new Error(raw || "the instruction was not saved")
}

/** Posts a form body and returns the JSON payload. */
export async function postForm(url: string, body: URLSearchParams): Promise<unknown> {
  const response = await fetch(url, {
    method: "POST",
    headers: withAuth(new Headers({ "Content-Type": "application/x-www-form-urlencoded" })),
    body,
  })
  refuseUnauthorized(response)
  const payload = await readBody(response)
  if (!response.ok) {
    throw new Error(messageOf(payload))
  }
  return payload
}

function agentStateOf(value: unknown): AgentState {
  const row = isRecord(value) ? value : {}
  return {
    id: text(row.id),
    label: text(row.label),
    binary: text(row.binary),
    installed: flag(row.installed),
    gentle: flag(row.gentle),
    ready: flag(row.ready),
    reason: text(row.reason),
  }
}

/** Every CLI Span knows, which one the project asked for, and which would run. */
export function agentChoice(value: unknown): AgentChoice {
  const body = isRecord(value) ? value : {}
  const rows = Array.isArray(body.agents) ? body.agents : []
  return {
    agents: rows.map(agentStateOf).filter((row) => row.id !== ""),
    selected: text(body.selected) === "" ? "any" : text(body.selected),
    chosen: text(body.chosen),
    exists: flag(body.exists),
    reason: text(body.reason),
    models: modelMap(body.models),
  }
}

function modelMap(value: unknown): Record<string, string> {
  const out: Record<string, string> = {}
  if (!isRecord(value)) {
    return out
  }
  for (const [id, model] of Object.entries(value)) {
    if (typeof model === "string" && model !== "") {
      out[id] = model
    }
  }
  return out
}

/** The models one CLI can offer. A reply that is not a list reads as an empty free-text one. */
export function modelList(value: unknown): ModelList {
  const body = isRecord(value) ? value : {}
  const rows = Array.isArray(body.models) ? body.models : []
  const options: ModelOption[] = []
  for (const row of rows) {
    const item = isRecord(row) ? row : {}
    if (text(item.id) !== "") {
      options.push({ id: text(item.id), label: text(item.label) })
    }
  }
  const source = body.source === "cli" || body.source === "fixed" ? body.source : "none"
  return { models: options, free: body.free !== false, source, error: text(body.error) }
}

/** Save the model one CLI runs on. An empty model gives the choice back to Gentle AI. */
export async function saveModel(project: string, cli: string, model: string): Promise<unknown> {
  return postForm("/agent/model", new URLSearchParams({ project, cli, model }))
}

/** Delete one run's record. The server refuses a run that is being driven. */
export async function deleteRun(id: string): Promise<void> {
  await postForm(`/spans/${encodeURIComponent(id)}/delete`, new URLSearchParams())
}

/** Save the agent the project runs. The server rewrites one line of span.yaml. */
export async function saveAgent(project: string, id: string): Promise<unknown> {
  return postForm("/agent", new URLSearchParams({ project, agent: id }))
}

/** Save which CLI one stage of this run uses. An empty id clears the exception. */
export async function setStageAgent(id: string, stage: string, agentId: string): Promise<unknown> {
  return postForm(`/spans/${encodeURIComponent(id)}/agent`, new URLSearchParams({ stage, agent: agentId }))
}

/** Ask the run to hold before the next stage starts. Answers why it did or did not. */
export async function pauseRun(id: string): Promise<string> {
  const body = await postForm(`/spans/${encodeURIComponent(id)}/pause`, new URLSearchParams())
  const record = isRecord(body) ? body : {}
  return text(record.reason)
}

/** Restart the run at a stage. The note is what that stage reads when it starts. */
export async function rewindRun(id: string, stage: string, note: string): Promise<unknown> {
  return postForm(`/spans/${encodeURIComponent(id)}/rewind`, new URLSearchParams({ stage, note }))
}

/** Cancel the stage that is running now. Answers why it did or did not. */
export async function stopRun(id: string): Promise<string> {
  const body = await postForm(`/spans/${encodeURIComponent(id)}/stop`, new URLSearchParams())
  const record = isRecord(body) ? body : {}
  return text(record.reason)
}

/** Applies the person's choice: one `group`/`option` pair per group, in group order. */
export async function chooseConflict(id: string, pairs: { group: string; option: string }[]): Promise<unknown> {
  const body = new URLSearchParams()
  for (const pair of pairs) {
    body.append("group", pair.group)
    body.append("option", pair.option)
  }
  return postForm(`/spans/${encodeURIComponent(id)}/conflict/choose`, body)
}

/** Sends the person's hint so the agent proposes the options again. */
export async function hintConflict(id: string, hint: string): Promise<unknown> {
  return postForm(`/spans/${encodeURIComponent(id)}/conflict/hint`, new URLSearchParams({ text: hint }))
}

/** Runs `git merge --abort`: the branch is left exactly as it was. */
export async function abortConflict(id: string): Promise<unknown> {
  return postForm(`/spans/${encodeURIComponent(id)}/conflict/abort`, new URLSearchParams())
}

function spanHeadOf(value: unknown): SpanHead {
  const row = isRecord(value) ? value : {}
  return {
    id: text(row.id),
    label: text(row.label),
    issue: text(row.issue),
    repo: text(row.repo),
    stage: text(row.stage),
    verdict: text(row.verdict),
    title: text(row.title),
    branch: text(row.branch),
    reason: text(row.reason),
    updatedAt: text(row.updatedAt),
  }
}

/** The runs the desk lists, with the label a person reads instead of the bare id. */
export function spanHeadsOf(value: unknown): SpanHead[] {
  const body = isRecord(value) ? value : {}
  const rows = Array.isArray(body.spans) ? body.spans : []
  return rows.map(spanHeadOf).filter((row) => row.id !== "")
}

/** A refusal that carries the server's error code beside its message. */
export class ApiError extends Error {
  readonly code: string

  constructor(code: string, message: string) {
    super(message)
    this.name = "ApiError"
    this.code = code
  }
}

async function shapeRequest(method: "GET" | "POST", url: string, body?: unknown): Promise<unknown> {
  const headers = new Headers({ Accept: "application/json" })
  if (body !== undefined) {
    headers.set("Content-Type", "application/json")
  }
  const response = await fetch(url, { method, headers: withAuth(headers), body: body === undefined ? undefined : JSON.stringify(body) })
  refuseUnauthorized(response)
  const raw = await response.text()
  if (response.ok) {
    return raw === "" ? null : parseJSON(raw)
  }
  const payload = errorBody(raw)
  throw new ApiError(isRecord(payload) ? text(payload.error) : "", messageOf(payload))
}

function shapeSource(value: unknown): ShapeSource {
  const row = isRecord(value) ? value : {}
  return { kind: oneOf(row.kind, shapeSourceKinds, "prompt"), ref: text(row.ref) }
}

function shapeJira(value: unknown): ShapeJira | null {
  if (!isRecord(value) || text(value.key) === "") {
    return null
  }
  return { key: text(value.key), url: text(value.url), status: text(value.status) }
}

/** Reads one shaping session. The reply is untrusted input, so every part has a safe default. */
export function shapeSession(value: unknown): ShapeSession {
  const body = isRecord(value) ? value : {}
  const map = isRecord(body.map) ? body.map : {}
  const rules: ShapeRule[] = rows(map.rules).map((row) => ({ id: text(row.id), text: text(row.text), source: shapeSource(row.source) }))
  const examples: ShapeExample[] = rows(map.examples).map((row) => ({
    id: text(row.id),
    rule: text(row.rule),
    text: text(row.text),
    source: shapeSource(row.source),
  }))
  const questions: ShapeQuestion[] = rows(map.questions).map((row) => ({
    id: text(row.id),
    text: text(row.text),
    blocks: text(row.blocks),
    assume: text(row.assume),
    status: oneOf(row.status, shapeQuestionStates, "open"),
    answer: text(row.answer),
  }))
  const split: ShapeSplit[] = rows(body.split).map((row) => ({ story: text(row.story), leftOut: text(row.leftOut) }))
  const proposals: ShapeProposal[] = rows(body.proposals).map((row) => ({
    id: text(row.id),
    section: text(row.section),
    text: text(row.text),
    status: oneOf(row.status, shapeProposalStates, "proposed"),
  }))
  return {
    id: text(body.id),
    workspace: text(body.workspace),
    repo: text(body.repo),
    idea: text(body.idea),
    status: oneOf(body.status, shapeStates, "interviewing"),
    rules,
    examples,
    questions,
    split,
    proposals,
    story: text(body.story),
    digest: text(body.digest),
    jira: shapeJira(body.jira),
  }
}

/** Reads the Jira preview. */
export function shapePreview(value: unknown): ShapePreview {
  const body = isRecord(value) ? value : {}
  const board = isRecord(body.board) ? body.board : {}
  return {
    project: text(body.project),
    board: { id: count(board.id), name: text(board.name) },
    backlogStatuses: stringList(body.backlogStatuses),
    summary: text(body.summary),
    description: text(body.description),
  }
}

/** The sessions of one workspace, newest as the server orders them. */
export async function listShapes(workspace: string): Promise<ShapeSession[]> {
  const body = await shapeRequest("GET", `/shapes?workspace=${encodeURIComponent(workspace)}`)
  return Array.isArray(body) ? body.map(shapeSession) : []
}

/** One session. */
export async function getShape(id: string, workspace: string): Promise<ShapeSession> {
  return shapeSession(await shapeRequest("GET", `/shapes/${encodeURIComponent(id)}?workspace=${encodeURIComponent(workspace)}`))
}

/** Starts a session; the server runs the first turn before it answers. */
export async function createShape(input: { workspace: string; repo?: string; idea: string; agent?: string; model?: string }): Promise<ShapeSession> {
  return shapeSession(await shapeRequest("POST", "/shapes", input))
}

/** Sends every answer and accepted assumption of one round; the server runs the next turn. */
export async function answerShape(id: string, workspace: string, answers: { question: string; text: string }[], assume: string[]): Promise<ShapeSession> {
  return shapeSession(await shapeRequest("POST", `/shapes/${encodeURIComponent(id)}/answers`, { workspace, answers, assume }))
}

/** Accepts the product-context proposals the person ticked. */
export async function acceptShapeContext(id: string, workspace: string, accept: string[]): Promise<ShapeSession> {
  return shapeSession(await shapeRequest("POST", `/shapes/${encodeURIComponent(id)}/context`, { workspace, accept }))
}

/** Seals the story. Refuses with `shape_not_ready` while a question is open. */
export async function sealShape(id: string, workspace: string): Promise<ShapeSession> {
  return shapeSession(await shapeRequest("POST", `/shapes/${encodeURIComponent(id)}/seal`, { workspace }))
}

/** What the Jira issue would look like, with nothing created. */
export async function previewShapeIssue(id: string, workspace: string, project: string, board: string): Promise<ShapePreview> {
  const query = new URLSearchParams({ workspace, project })
  if (board.trim() !== "") {
    query.set("board", board.trim())
  }
  return shapePreview(await shapeRequest("GET", `/shapes/${encodeURIComponent(id)}/jira/preview?${query.toString()}`))
}

/** Creates the issue in the backlog. Only the confirm button calls this, so `confirm` is always true. */
export async function createShapeIssue(id: string, workspace: string, project: string, board: string): Promise<ShapeSession> {
  const request: { workspace: string; project: string; board?: number; confirm: true } = { workspace, project, confirm: true }
  if (board.trim() !== "") {
    request.board = Number(board.trim())
  }
  return shapeSession(await shapeRequest("POST", `/shapes/${encodeURIComponent(id)}/jira`, request))
}
