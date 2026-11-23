import { describe, expect, it } from "vitest"

import { ranWith } from "@/lib/ranwith"
import type { JournalEvent } from "@/hooks/use-events"

const run = (seq: number, stage: string, agent: string, model: string): JournalEvent => ({
  seq,
  time: "",
  actor: "span",
  type: "agent.run",
  detail: { stage, agent, model },
})

describe("ranWith", () => {
  it("keeps the last agent and model of each stage, in the order of the run", () => {
    const rows = ranWith([run(1, "build", "claude", "a"), run(2, "frame", "pi", "m"), run(3, "build", "pi", "b")])
    expect(rows).toEqual([
      { stage: "frame", agent: "pi", model: "m" },
      { stage: "build", agent: "pi", model: "b" },
    ])
  })

  it("keeps an empty model empty instead of inventing one", () => {
    expect(ranWith([run(1, "frame", "cursor-agent", "")])[0]?.model).toBe("")
  })

  it("ignores events that are not agent runs", () => {
    expect(ranWith([{ seq: 1, time: "", actor: "span", type: "stage", detail: { stage: "frame" } }])).toEqual([])
  })
})
