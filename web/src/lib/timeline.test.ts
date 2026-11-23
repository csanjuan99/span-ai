import { describe, expect, it } from "vitest"

import { clockOf, timelineLines, withAgent, agentLine, type TimelineFrame } from "@/lib/timeline"
import type { Cursor, LiveFrame, Statement } from "@/lib/types"

const at = "2026-11-19T03:44:01.000Z"

function statement(stage: string, verdict: string, reason: string, time: string): Statement {
  return { stage, iteration: 1, verdict, reason, time, commands: [] }
}

function cursor(stage: string, reason: string, hold: boolean, stop = false): Cursor {
  return {
    id: "run",
    stage,
    iteration: 1,
    hold,
    stop,
    done: false,
    reason,
    gitSHA: "",
    changeURL: "",
    branch: "",
    base: "",
    workspace: "",
    issue: "",
    questions: [],
  }
}

function frame(atStamp: string, body: LiveFrame): TimelineFrame {
  return { at: atStamp, frame: body }
}

describe("timelineLines", () => {
  it("seeds sealed statements in time order", () => {
    const lines = timelineLines({
      statements: [
        statement("qa", "advance", "acceptance gate passed", "2026-11-19T03:44:18.000Z"),
        statement("frame", "hold", "open questions: Which permission atom?", "2026-11-19T03:44:01.000Z"),
      ],
      frames: [],
      actions: [],
    })
    expect(lines.map((line) => `${line.clock} ${line.stage} ${line.mark} ${line.detail}`.trim())).toEqual([
      `${clockOf("2026-11-19T03:44:01.000Z")} frame hold open questions: Which permission atom?`,
      `${clockOf("2026-11-19T03:44:18.000Z")} qa advance acceptance gate passed`,
    ])
    expect(lines[0]?.tone).toBe("hold")
    expect(lines[1]?.tone).toBe("pass")
  })

  it("collapses the same stage, verdict, and reason", () => {
    const held = statement("frame", "hold", "open questions: Which permission atom?", at)
    const lines = timelineLines({
      statements: [held],
      frames: [frame(at, { type: "statement", statement: held })],
      actions: [],
    })
    expect(lines).toHaveLength(1)
  })

  it("adds a cursor line only when stage, hold, stop, or reason changes", () => {
    const lines = timelineLines({
      statements: [],
      frames: [
        frame(at, { type: "cursor", cursor: cursor("frame", "brief accepted", false) }),
        frame("2026-11-19T03:44:02.000Z", { type: "cursor", cursor: cursor("frame", "brief accepted", false) }),
        frame("2026-11-19T03:44:12.000Z", { type: "cursor", cursor: cursor("frame", "open questions: Which permission atom?", true) }),
      ],
      actions: [],
    })
    expect(lines.map((line) => `${line.stage} ${line.mark}`)).toEqual(["frame running", "frame hold"])
    expect(lines[1]?.tone).toBe("hold")
    expect(lines[0]?.tone).toBe("primary")
  })

  it("summarizes a transcript flush as one line", () => {
    const lines = timelineLines({
      statements: [],
      frames: [frame(at, { type: "transcript", stage: "build", lines: ["a", "b", "c", "d"] })],
      actions: [],
    })
    expect(lines).toHaveLength(1)
    expect(lines[0]?.stage).toBe("build")
    expect(lines[0]?.mark).toBe("transcript")
    expect(lines[0]?.detail).toBe("+4 lines")
  })

  it("records a gate and a local action after the decisions", () => {
    const lines = timelineLines({
      statements: [statement("build", "advance", "feature document present", at)],
      frames: [frame("2026-11-19T03:44:18.000Z", { type: "gate", gate: { verdict: "pass", command: "", exitCode: 0 } })],
      actions: [{ at: "2026-11-19T03:44:12.000Z", text: "answer saved" }],
    })
    expect(lines.map((line) => line.stage)).toEqual(["build", "qa", ">"])
    expect(lines[1]?.mark).toBe("pass")
    expect(lines[1]?.tone).toBe("pass")
    expect(lines[2]?.mark).toBe("answer saved")
    expect(lines[2]?.tone).toBe("primary")
  })

  it("seeds a replay from its statements and does not copy the transcript tail", () => {
    const lines = timelineLines({
      statements: [],
      frames: [
        frame(at, {
          type: "replay",
          transcript: ["one", "two"],
          span: {
            cursor: cursor("frame", "brief accepted", false),
            statements: [statement("intake", "advance", "brief accepted", at)],
            acceptance: [],
            gate: null,
          },
        }),
      ],
      actions: [],
    })
    expect(lines).toHaveLength(1)
    expect(lines[0]?.stage).toBe("intake")
  })

  it("keeps one agent summary and replaces it on the next review", () => {
    const seeded = timelineLines({
      statements: [statement("frame", "hold", "agent reply has no JSON object", at)],
      frames: [],
      actions: [],
    })
    const first = withAgent(
      seeded,
      agentLine({ id: "", running: false, relaunch: true, name: "", pid: 0, stage: "frame", elapsed: "", summary: "not running" }, at),
    )
    const second = withAgent(
      first,
      agentLine(
        { id: "", running: true, relaunch: false, name: "claude", pid: 7, stage: "frame", elapsed: "14s", summary: "claude 14s" },
        "2026-11-19T03:44:12.000Z",
      ),
    )
    const agent = second.filter((line) => line.mark === "agent")
    expect(agent).toHaveLength(1)
    expect(agent[0]?.detail).toBe("claude 14s")
    expect(agent[0]?.tone).toBe("primary")
    expect(second.at(-1)?.mark).toBe("agent")
  })
})
