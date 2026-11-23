import type { Statement, StatementCommand } from "@/lib/types"

/** How a statement reads at a glance: it moved on, waits for a person, or stopped the run. */
export type Tone = "pass" | "hold" | "stop"

/** The tone of a verdict. Anything that is not a hold or a stop moved the run on. */
export function toneOf(verdict: string): Tone {
  if (verdict === "stop") {
    return "stop"
  }
  return verdict === "hold" ? "hold" : "pass"
}

/** The first seven characters of a commit or digest, without a `sha256:` prefix. */
export function shortHash(value: string): string {
  return value.replace(/^sha256:/, "").slice(0, 7)
}

/** `PR #8` for a GitHub pull request, `MR !3` for a GitLab merge request, else host and path. */
export function changeLabel(url: string): string {
  const clean = url.trim()
  const pull = /\/pull\/(\d+)/.exec(clean)
  if (pull !== null) {
    return `PR #${pull[1]}`
  }
  const merge = /\/merge_requests\/(\d+)/.exec(clean)
  if (merge !== null) {
    return `MR !${merge[1]}`
  }
  return clean.replace(/^https?:\/\//, "")
}

/** A command that did not succeed: a non-zero exit, or an error the runner recorded. */
export function failed(command: StatementCommand): boolean {
  return command.exitCode !== 0 || (command.error ?? "") !== ""
}

/** The statements of one pass through the stages, in the order they were sealed. */
export interface Round {
  /** 1-based position of the pass in the run. */
  number: number
  items: Statement[]
}

/**
 * Groups statements into passes. A new pass starts when a stage shows up again, which is what
 * happens when inspect or a failing check sends the work back to build. The iteration counter
 * is not used: inspect returns do not raise it.
 */
export function rounds(statements: Statement[]): Round[] {
  const out: Round[] = []
  for (const item of statements) {
    const last = out[out.length - 1]
    if (last !== undefined && !last.items.some((seen) => seen.stage === item.stage)) {
      last.items.push(item)
    } else {
      out.push({ number: out.length + 1, items: [item] })
    }
  }
  return out
}
