import { hhmmss } from "@/lib/clock"
import type {
  AgentStatus,
  Cursor,
  Gate,
  LiveFrame,
  Statement,
} from "@/lib/types"

/** How often the page asks whether an agent is still running. */
export const agentReviewMs = 2000

/** One decision on the terminal band. */
export interface TimelineLine {
  clock: string
  stage: string
  mark: string
  detail: string
  tone: "primary" | "pass" | "hold" | "stop" | "muted"
  key: string
}

/** A live frame stamped when it arrived. */
export interface TimelineFrame {
  at: string
  frame: LiveFrame
}

/** A local page action. The mark is what the person did. */
export interface TimelineAction {
  at: string
  text: string
}

/** Hours, minutes, and seconds from an RFC3339 stamp. An empty stamp stays empty. */
export function clockOf(at: string): string {
  if (at === "") {
    return ""
  }
  const date = new Date(at)
  if (Number.isNaN(date.getTime())) {
    return at
  }
  return hhmmss(date)
}

function toneOf(mark: string): TimelineLine["tone"] {
  if (mark === "advance" || mark === "pass" || mark === "done") {
    return "pass"
  }
  if (mark === "hold" || mark === "unverified") {
    return "hold"
  }
  if (mark === "stop" || mark === "fail") {
    return "stop"
  }
  if (mark === "running") {
    return "primary"
  }
  return "muted"
}

function cursorMark(cursor: Cursor): string {
  if (cursor.done) {
    return "done"
  }
  if (cursor.stop) {
    return "stop"
  }
  if (cursor.hold) {
    return "hold"
  }
  return "running"
}

function decision(
  clock: string,
  stage: string,
  mark: string,
  detail: string
): TimelineLine {
  return {
    clock,
    stage,
    mark,
    detail,
    tone: toneOf(mark),
    key: `${stage}|${mark}|${detail}`,
  }
}

function statementLine(item: Statement, fallback: string): TimelineLine {
  return decision(
    clockOf(item.time && item.time !== "" ? item.time : fallback),
    item.stage,
    item.verdict,
    item.reason
  )
}

function gateLine(clock: string, gate: Gate): TimelineLine {
  return decision(clock, "qa", gate.verdict, gate.command)
}

/** Turns sealed statements, live frames, and local actions into one band. Same stage, verdict, and reason collapse. */
export function timelineLines(input: {
  statements: Statement[]
  gate?: Gate | null
  frames: TimelineFrame[]
  actions: TimelineAction[]
}): TimelineLine[] {
  const lines: TimelineLine[] = []
  const seen = new Set<string>()
  const push = (line: TimelineLine) => {
    if (seen.has(line.key)) {
      return
    }
    seen.add(line.key)
    lines.push(line)
  }

  const statements = [...input.statements].sort((left, right) =>
    (left.time ?? "").localeCompare(right.time ?? "")
  )
  for (const item of statements) {
    push(statementLine(item, ""))
  }
  if (input.gate) {
    push(gateLine(clockOf(statements.at(-1)?.time ?? ""), input.gate))
  }

  let signature = ""
  let transcriptIndex = 0
  for (const stamped of input.frames) {
    const frame = stamped.frame
    const clock = clockOf(stamped.at)
    if (frame.type === "replay" && frame.span) {
      for (const item of frame.span.statements) {
        push(statementLine(item, stamped.at))
      }
      if (frame.span.gate) {
        push(gateLine(clock, frame.span.gate))
      }
      continue
    }
    if (frame.type === "cursor" && frame.cursor) {
      const cursor = frame.cursor
      const next = `${cursor.stage}|${cursor.hold}|${cursor.stop}|${cursor.reason}`
      if (next === signature) {
        continue
      }
      signature = next
      push(decision(clock, cursor.stage, cursorMark(cursor), cursor.reason))
    }
    if (frame.type === "statement" && frame.statement) {
      push(statementLine(frame.statement, stamped.at))
    }
    if (frame.type === "transcript" && frame.lines && frame.lines.length > 0) {
      transcriptIndex += 1
      const stage = frame.stage && frame.stage !== "" ? frame.stage : "build"
      const detail = `+${frame.lines.length} lines`
      const line = decision(clock, stage, "transcript", detail)
      line.key = `${stage}|transcript|${detail}|${transcriptIndex}`
      push(line)
    }
    if (frame.type === "gate" && frame.gate) {
      push(gateLine(clock, frame.gate))
    }
  }

  for (const action of input.actions) {
    push({
      clock: clockOf(action.at),
      stage: ">",
      mark: action.text,
      detail: "",
      tone: "primary",
      key: `>|${action.at}|${action.text}`,
    })
  }
  return lines
}

/** One agent review. A later review replaces the previous one. */
export function agentLine(
  status: AgentStatus | null,
  at: string
): TimelineLine | null {
  if (status === null || status.summary === "") {
    return null
  }
  let tone: TimelineLine["tone"] = "muted"
  if (status.running) {
    tone = "primary"
  } else if (status.summary === "not running") {
    tone = "hold"
  } else if (status.summary === "stopped") {
    tone = "stop"
  }
  return {
    clock: clockOf(at),
    stage: status.stage !== "" ? status.stage : "agent",
    mark: "agent",
    detail: status.summary,
    tone,
    key: "agent",
  }
}

/** Replaces the previous agent summary and keeps it at the end of the band. */
export function withAgent(
  lines: TimelineLine[],
  agent: TimelineLine | null
): TimelineLine[] {
  const rest = lines.filter((line) => line.key !== "agent")
  if (agent === null) {
    return rest
  }
  return [...rest, agent]
}

/** One desk line for the agents this process is running. Nobody running leaves the band alone. */
export function deskAgentLine(
  agents: AgentStatus[],
  at: string
): TimelineLine | null {
  const first = agents[0]
  if (!first) {
    return null
  }
  const more = agents.length > 1 ? ` +${agents.length - 1}` : ""
  return {
    clock: clockOf(at),
    stage: "agent",
    mark: "running",
    detail: `${first.stage} ${first.summary}${more}`.trim(),
    tone: "primary",
    key: "agent",
  }
}
