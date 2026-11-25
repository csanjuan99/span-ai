import { sinceLine } from "@/lib/since"

/** How the run stands, in the words the header badge uses. */
export type RunState = "running" | "waiting" | "hold" | "stop" | "done"

export const stateBadge: Record<
  RunState,
  { label: string; tone: "acc" | "hold" | "fail" | "pass" }
> = {
  running: { label: "Running", tone: "acc" },
  waiting: { label: "Waiting for you", tone: "hold" },
  hold: { label: "On hold", tone: "hold" },
  stop: { label: "Stopped", tone: "fail" },
  done: { label: "Done", tone: "pass" },
}

/**
 * The time line under the title: when the run began while it works, when it was held or stopped
 * while it waits, when it finished once it is done. Empty when the stamp is not known.
 */
export function metaTime(input: {
  state: RunState
  startedAt: string
  lastStamp: string
  now: Date
}): string {
  switch (input.state) {
    case "running":
      return sinceLine("started", input.startedAt, input.now).replace(
        / ago$/,
        ""
      )
    case "waiting":
      return sinceLine("waiting since", input.lastStamp, input.now)
    case "hold":
      return sinceLine("held", input.lastStamp, input.now)
    case "stop":
      return sinceLine("stopped", input.lastStamp, input.now)
    case "done": {
      const clock = sinceLine("finished", input.lastStamp, input.now).replace(
        / · .* ago$/,
        ""
      )
      const took = tookLine(input.startedAt, input.lastStamp)
      return clock === "" || took === "" ? clock : `${clock} · took ${took}`
    }
  }
}

/** How long a run took, from its first stamp to its last: "1 h 33 min", "41 min", or empty when unknown. */
function tookLine(start: string, end: string): string {
  const from = Date.parse(start)
  const to = Date.parse(end)
  if (Number.isNaN(from) || Number.isNaN(to) || to <= from) {
    return ""
  }
  const minutes = Math.round((to - from) / 60_000)
  const hours = Math.floor(minutes / 60)
  return hours < 1
    ? `${minutes} min`
    : `${hours} h${minutes % 60 === 0 ? "" : ` ${minutes % 60} min`}`
}
