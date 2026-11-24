import { isRecord, rows, text } from "@/lib/guards"

/** What the metrics page reads. The reply is untrusted input, so each field is checked. */
export interface MetricRuns {
  total: number
  done: number
  stopped: number
  held: number
  running: number
  firstTry: number
}

export interface StageStat {
  stage: string
  runs: number
  advance: number
  hold: number
  stop: number
  median: number
  p90: number
}

export interface AgentStat {
  agent: string
  model: string
  stages: number
  advance: number
  hold: number
  stop: number
}

export interface RunRow {
  id: string
  label: string
  stage: string
  verdict: string
  iterations: number
  seconds: number
  agents: string[]
}

export interface Metrics {
  runs: MetricRuns
  stages: StageStat[]
  agents: AgentStat[]
  recent: RunRow[]
}

function num(value: unknown): number {
  return typeof value === "number" && Number.isFinite(value) && value >= 0 ? value : 0
}

/** Reads a /metrics reply. Anything that is not the expected shape reads as no runs. */
export function metricsOf(body: unknown): Metrics {
  const root = isRecord(body) ? body : {}
  const runs = isRecord(root.runs) ? root.runs : {}
  return {
    runs: {
      total: num(runs.total),
      done: num(runs.done),
      stopped: num(runs.stopped),
      held: num(runs.held),
      running: num(runs.running),
      firstTry: num(runs.firstTry),
    },
    stages: rows(root.stages).map((row) => ({
      stage: text(row.stage),
      runs: num(row.runs),
      advance: num(row.advance),
      hold: num(row.hold),
      stop: num(row.stop),
      median: num(row.medianSeconds),
      p90: num(row.p90Seconds),
    })),
    agents: rows(root.agents).map((row) => ({
      agent: text(row.agent),
      model: text(row.model),
      stages: num(row.stages),
      advance: num(row.advance),
      hold: num(row.hold),
      stop: num(row.stop),
    })),
    recent: rows(root.recent).map((row) => ({
      id: text(row.id),
      label: text(row.label),
      stage: text(row.stage),
      verdict: text(row.verdict),
      iterations: num(row.iterations),
      seconds: num(row.seconds),
      agents: Array.isArray(row.agents) ? row.agents.filter((item): item is string => typeof item === "string") : [],
    })),
  }
}

/** A time as a person reads it: 45s, 3m 20s, 1h 05m. Nothing measured is a dash. */
export function duration(seconds: number): string {
  if (seconds <= 0) {
    return "—"
  }
  const whole = Math.round(seconds)
  if (whole < 60) {
    return `${whole}s`
  }
  if (whole < 3600) {
    return `${Math.floor(whole / 60)}m ${String(whole % 60).padStart(2, "0")}s`
  }
  return `${Math.floor(whole / 3600)}h ${String(Math.floor((whole % 3600) / 60)).padStart(2, "0")}m`
}

/** part of whole as a whole percent, or a dash when there is no whole. */
export function percent(part: number, whole: number): string {
  if (whole <= 0) {
    return "—"
  }
  return `${Math.round((part / whole) * 100)}%`
}

/** How wide a bar is, 0 to 100, against the largest value shown. */
export function barWidth(value: number, max: number): number {
  if (max <= 0 || value <= 0) {
    return 0
  }
  return Math.max(2, Math.min(100, Math.round((value / max) * 100)))
}

/** What to tell the person when the metrics could not be read. */
export function loadFailure(err: unknown): string {
  return err instanceof Error && err.message !== "" ? err.message : "the metrics could not load"
}
