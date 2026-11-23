import type { LiveFrame } from "@/lib/types"

const firstDelayMs = 500
const maxDelayMs = 10000
const jitter = 0.2

/**
 * How long to wait before reconnect attempt `attempt` (0 for the first). Doubling keeps a
 * restarting server from being hammered; the cap keeps a laptop that wakes up from
 * waiting minutes; the jitter keeps several tabs from reconnecting in lockstep.
 * Injecting `random` makes the delay deterministic for tests.
 */
export function reconnectDelay(attempt: number, random: () => number = Math.random): number {
  const step = Number.isFinite(attempt) && attempt > 0 ? Math.min(Math.floor(attempt), 30) : 0
  const base = Math.min(firstDelayMs * 2 ** step, maxDelayMs)
  const spread = Math.min(Math.max(random(), 0), 1)
  return Math.round(base * (1 - jitter + 2 * jitter * spread))
}

/**
 * What to do with one live frame given the last seq applied. `reset` re-baselines on a
 * replay, `skip` drops a frame the replay already covered, `gap` means frames were lost
 * and the page must resync, and `apply` covers the next frame or a server that does not
 * number its frames.
 */
export type FrameVerdict = "apply" | "skip" | "gap" | "reset"

/** Decides how a live frame relates to the stream seen so far. See {@link FrameVerdict}. */
export function frameGate(last: number, frame: LiveFrame): FrameVerdict {
  if (frame.type === "replay") {
    return "reset"
  }
  if (frame.seq === undefined) {
    return "apply"
  }
  if (frame.seq <= last) {
    return "skip"
  }
  return frame.seq === last + 1 ? "apply" : "gap"
}

/**
 * Whether a transcript frame only repeats lines the replay tail already holds. The log
 * writes a line before it publishes it, so a line written just before a connect is in
 * the tail and also arrives in the next flush; its offset is not past the replay's.
 */
export function coveredByReplay(offset: number, frame: LiveFrame): boolean {
  if (frame.type !== "transcript" || frame.offset === undefined) {
    return false
  }
  return frame.offset <= offset
}
