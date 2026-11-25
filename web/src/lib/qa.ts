import { count, flag, getJSON, shapeRequest } from "@/lib/api"
import { isRecord, oneOf, rows, text } from "@/lib/guards"

/** How a gate or the whole proof reads. A pass needs proof, so `held` is not a pass. */
export type QaVerdict = "pass" | "fail" | "held" | "notrun"

/** The honest states the QA tab can be in, besides a plain reading. */
export type QaState = "ok" | "no_cases" | "never_ran" | "unsealed"

/** What the checks said about the tests they ran. */
export interface QaCounts {
  passed: number
  failed: number
  skipped: number
  seconds: number
}

/** One check of the proof, in the order it ran. */
export interface QaGate {
  name: string
  band: string
  command: string
  verdict: QaVerdict
  ran: boolean
  exitCode: number
  durationMs: number
  lines: number
  counts: { passed: number; failed: number; skipped: number } | null
  preexisting: boolean
  endedAt: string
  detail: string
}

/** A person's recorded decision to go on without proof. */
export interface QaAcceptance {
  reason: string
  time: string
  gate: string
}

/** The QA reading of a run, as GET /spans/{id}/qa returns it. */
export interface QaView {
  run: string
  state: QaState
  verdict: QaVerdict
  headline: string
  command: string
  exitCode: number
  exit: number
  evidenceNote: string
  counts: QaCounts | null
  gates: QaGate[]
  cases: number
  reason: string
  kept: string
  next: string
  accepted: QaAcceptance | null
}

/** One result of a case: its first run and each rerun. */
export interface QaHistory {
  time: string
  kind: string
  result: string
  message: string
}

/** A test case of the proof. */
export interface QaCase {
  name: string
  gate: string
  result: "pass" | "fail" | "skip"
  elapsedMs: number
  path: string
  file: string
  line: number
  message: string
  output: string[]
  isNew: boolean
  history: QaHistory[]
}

/** Whether a gate ran and left a log to open. */
export function hasLog(gate: QaGate): boolean {
  return gate.ran && gate.lines > 0
}

/** The first gate with a log, which the Log tab opens when none is chosen. */
export function firstLoggedGate(gates: QaGate[]): QaGate | undefined {
  return gates.find(hasLog)
}

/** The case list, as GET /spans/{id}/qa/cases returns it. */
export interface QaCasesView {
  cases: QaCase[]
  total: number
  passed: number
  failed: number
  added: number
  known: boolean
  note: string
  rerun: string
}

/** One line of a gate's log, numbered in the whole log. */
export interface LogLine {
  n: number
  text: string
  kind: "ctx" | "err"
}

/** A gate's log, as GET /spans/{id}/logs/{gate} returns it. */
export interface GateLog {
  gate: string
  command: string
  exitCode: number
  durationMs: number
  endedAt: string
  bytes: number
  totalLines: number
  firstLine: number
  kept: number
  truncated: boolean
  windowed: boolean
  firstError: number
  q: string
  matches: number[]
  lines: LogLine[]
}

/** One block of what the agent printed. */
export interface TranscriptTurn {
  n: number
  run: number
  text: string
  cut: boolean
  firstLine: number
  lines: number
}

/** What the agent printed in one stage, as GET /spans/{id}/transcript returns it. */
export interface TranscriptView {
  stage: string
  agent: string
  stages: { stage: string; runs: number; lines: number }[]
  startedAt: string
  endedAt: string
  total: number
  from: number
  lines: number
  redacted: number
  turns: TranscriptTurn[]
  note: string
}

const verdicts: QaVerdict[] = ["pass", "fail", "held", "notrun"]
const states: QaState[] = ["ok", "no_cases", "never_ran", "unsealed"]

function gateOf(row: Record<string, unknown>): QaGate {
  const counts = isRecord(row.counts) ? row.counts : null
  return {
    name: text(row.name),
    band: text(row.band),
    command: text(row.command),
    verdict: oneOf(row.verdict, verdicts, "notrun"),
    ran: flag(row.ran),
    exitCode: count(row.exitCode),
    durationMs: count(row.durationMs),
    lines: count(row.lines),
    counts:
      counts === null
        ? null
        : {
            passed: count(counts.passed),
            failed: count(counts.failed),
            skipped: count(counts.skipped),
          },
    preexisting: flag(row.preexisting),
    endedAt: text(row.endedAt),
    detail: text(row.detail),
  }
}

/** Reads the QA overview payload. */
export function qaView(value: unknown): QaView {
  if (!isRecord(value)) {
    throw new Error("QA payload is missing")
  }
  const counts = isRecord(value.counts) ? value.counts : null
  const evidence = isRecord(value.evidence) ? value.evidence : {}
  const accepted = isRecord(value.accepted) ? value.accepted : null
  return {
    run: text(value.run),
    state: oneOf(value.state, states, "ok"),
    verdict: oneOf(value.verdict, verdicts, "notrun"),
    headline: text(value.headline),
    command: text(value.command),
    exitCode: count(value.exitCode),
    exit: count(value.exit),
    evidenceNote: text(evidence.note),
    counts:
      counts === null
        ? null
        : {
            passed: count(counts.passed),
            failed: count(counts.failed),
            skipped: count(counts.skipped),
            seconds: count(counts.seconds),
          },
    gates: rows(value.gates).map(gateOf),
    cases: count(value.cases),
    reason: text(value.reason),
    kept: text(value.kept),
    next: text(value.next),
    accepted:
      accepted === null
        ? null
        : {
            reason: text(accepted.reason),
            time: text(accepted.time),
            gate: text(accepted.gate),
          },
  }
}

/** Reads one case, with the history of its results. */
export function qaCase(row: Record<string, unknown>): QaCase {
  return {
    name: text(row.name),
    gate: text(row.gate),
    result: oneOf(row.result, ["pass", "fail", "skip"] as const, "pass"),
    elapsedMs: count(row.elapsedMs),
    path: text(row.path),
    file: text(row.file),
    line: count(row.line),
    message: text(row.message),
    output: Array.isArray(row.output)
      ? row.output.filter((line): line is string => typeof line === "string")
      : [],
    isNew: flag(row.new),
    history: rows(row.history).map((item) => ({
      time: text(item.time),
      kind: text(item.kind),
      result: text(item.result),
      message: text(item.message),
    })),
  }
}

/** Reads the case list payload. */
export function qaCasesView(value: unknown): QaCasesView {
  if (!isRecord(value)) {
    throw new Error("QA cases payload is missing")
  }
  return {
    cases: rows(value.cases).map(qaCase),
    total: count(value.total),
    passed: count(value.passed),
    failed: count(value.failed),
    added: count(value.new),
    known: flag(value.known),
    note: text(value.note),
    rerun: text(value.rerun),
  }
}

/** Reads a gate's log payload. */
export function gateLog(value: unknown): GateLog {
  if (!isRecord(value)) {
    throw new Error("log payload is missing")
  }
  return {
    gate: text(value.gate),
    command: text(value.command),
    exitCode: count(value.exitCode),
    durationMs: count(value.durationMs),
    endedAt: text(value.endedAt),
    bytes: count(value.bytes),
    totalLines: count(value.totalLines),
    firstLine: count(value.firstLine),
    kept: count(value.kept),
    truncated: flag(value.truncated),
    windowed: flag(value.windowed),
    firstError: count(value.firstError),
    q: text(value.q),
    matches: Array.isArray(value.matches)
      ? value.matches.filter((n): n is number => typeof n === "number")
      : [],
    lines: rows(value.lines).map((row) => ({
      n: count(row.n),
      text: text(row.text),
      kind: row.kind === "err" ? "err" : "ctx",
    })),
  }
}

/** Reads a transcript payload. */
export function transcriptView(value: unknown): TranscriptView {
  if (!isRecord(value)) {
    throw new Error("transcript payload is missing")
  }
  return {
    stage: text(value.stage),
    agent: text(value.agent),
    stages: rows(value.stages).map((row) => ({
      stage: text(row.stage),
      runs: count(row.runs),
      lines: count(row.lines),
    })),
    startedAt: text(value.startedAt),
    endedAt: text(value.endedAt),
    total: count(value.total),
    from: count(value.from),
    lines: count(value.lines),
    redacted: count(value.redacted),
    note: text(value.note),
    turns: rows(value.turns).map((row) => ({
      n: count(row.n),
      run: count(row.run),
      text: text(row.text),
      cut: flag(row.cut),
      firstLine: count(row.firstLine),
      lines: count(row.lines),
    })),
  }
}

const run = (id: string) => `/spans/${encodeURIComponent(id)}`

/** Reads the QA overview of a run. */
export async function readQa(id: string): Promise<QaView> {
  return qaView(await getJSON(`${run(id)}/qa`))
}

/** Reads the cases of a run; `status` is "", "failed", "passed" or "new". */
export async function readQaCases(
  id: string,
  status: string
): Promise<QaCasesView> {
  const query = new URLSearchParams({ group: "file" })
  if (status !== "") {
    query.set("status", status)
  }
  return qaCasesView(await getJSON(`${run(id)}/qa/cases?${query.toString()}`))
}

/** Reads one case by name. */
export async function readQaCase(id: string, name: string): Promise<QaCase> {
  const body = await getJSON(`${run(id)}/qa/cases/${encodeURIComponent(name)}`)
  if (!isRecord(body)) {
    throw new Error("case payload is missing")
  }
  return qaCase(body)
}

/** Runs the named cases again; the server returns where each one stands now. */
export async function rerunQa(id: string, names: string[]): Promise<QaCase[]> {
  const body = await shapeRequest("POST", `${run(id)}/qa/rerun`, {
    cases: names,
  })
  return isRecord(body) ? rows(body.cases).map(qaCase) : []
}

/** Records the risk of going on without proof, with the person's reason. */
export async function acceptQa(id: string, reason: string): Promise<void> {
  await shapeRequest("POST", `${run(id)}/qa/accept`, { reason })
}

/** Reads a gate's log. `q` searches the whole log; `firstError` asks for the lines around it. */
export async function readGateLog(
  id: string,
  gate: string,
  q: string
): Promise<GateLog> {
  const query = new URLSearchParams()
  if (q !== "") {
    query.set("q", q)
  }
  const suffix = query.toString() === "" ? "" : `?${query.toString()}`
  return gateLog(
    await getJSON(`${run(id)}/logs/${encodeURIComponent(gate)}${suffix}`)
  )
}

/** Reads the transcript of a stage; an empty stage reads the last one that printed. */
export async function readTranscript(
  id: string,
  stage: string,
  from: number
): Promise<TranscriptView> {
  const query = new URLSearchParams({ from: String(from) })
  if (stage !== "") {
    query.set("stage", stage)
  }
  return transcriptView(
    await getJSON(`${run(id)}/transcript?${query.toString()}`)
  )
}

/** Seconds as the design writes them: "4.1 s", "11 s", and a dash for a command with no time. */
export function seconds(ms: number): string {
  if (ms <= 0) {
    return "–"
  }
  const value = ms / 1000
  return value >= 10 ? `${Math.round(value)} s` : `${value.toFixed(1)} s`
}

/** A case's time as "0.02s". */
export function caseTime(ms: number): string {
  return `${(ms / 1000).toFixed(2)}s`
}

/** The cases of one file, failures first, with the numbers the group header shows. */
export interface CaseGroup {
  path: string
  cases: QaCase[]
  passed: number
  failed: number
}

/** Groups cases by file, keeping the order the server sent (failures first, unplaced last). */
export function groupByFile(cases: QaCase[]): CaseGroup[] {
  const groups: CaseGroup[] = []
  for (const item of cases) {
    let group = groups.find((entry) => entry.path === item.path)
    if (group === undefined) {
      group = { path: item.path, cases: [], passed: 0, failed: 0 }
      groups.push(group)
    }
    group.cases.push(item)
    if (item.result === "pass") {
      group.passed += 1
    }
    if (item.result === "fail") {
      group.failed += 1
    }
  }
  return groups
}

/** What the log viewer draws: a line, or a stretch of passing lines folded away. */
export type LogItem =
  | { kind: "line"; line: LogLine }
  | { kind: "fold"; from: number; to: number; count: number; id: string }

/**
 * Folds the stretches of a log that nobody needs to read: runs of plain lines that are at least
 * `context` lines away from an error or a search match, the first `head` lines and the last
 * `tail` excepted. A fold the person opened stays open, so what they asked to see is never hidden.
 */
export function foldLog(
  lines: LogLine[],
  matches: number[],
  opened: ReadonlySet<string>,
  context = 2,
  head = 5,
  tail = 3
): LogItem[] {
  const interest = new Set<number>(matches)
  for (const line of lines) {
    if (line.kind === "err") {
      interest.add(line.n)
    }
  }
  const keep = lines.map((line, index) => {
    if (index < head || index >= lines.length - tail) {
      return true
    }
    for (let near = line.n - context; near <= line.n + context; near += 1) {
      if (interest.has(near)) {
        return true
      }
    }
    return false
  })
  const items: LogItem[] = []
  let index = 0
  while (index < lines.length) {
    if (keep[index]) {
      items.push({ kind: "line", line: lines[index] as LogLine })
      index += 1
      continue
    }
    let end = index
    while (end < lines.length && !keep[end]) {
      end += 1
    }
    const from = (lines[index] as LogLine).n
    const to = (lines[end - 1] as LogLine).n
    const id = `${from}-${to}`
    if (end - index < 3 || opened.has(id)) {
      for (let at = index; at < end; at += 1) {
        items.push({ kind: "line", line: lines[at] as LogLine })
      }
    } else {
      items.push({ kind: "fold", from, to, count: end - index, id })
    }
    index = end
  }
  return items
}

/** The exit code the terminal view of a verdict answers with, so the page can say the same. */
export function exitLabel(exit: number): string {
  switch (exit) {
    case 0:
      return "ok"
    case 10:
      return "a gate failed"
    case 11:
      return "no proof"
    default:
      return `exit ${exit}`
  }
}
