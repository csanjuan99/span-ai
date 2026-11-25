import { parseStat } from "@/lib/diff"
import type { Statement } from "@/lib/types"

/** What a person keeps if they resume, leave a note or wait: nothing here is undone by those. */
export interface Kept {
  sealed: number
  branch: string
  head: string
  /** Commits the branch added to its base, as the server counted them; 0 when it did not. */
  commits: number
  files: { files: number; plus: number; minus: number } | null
}

/** The totals git prints under `--stat`. Its bars are scaled down on a large diff; this line is not. */
const summary = /(\d+) insertions?\(\+\)|(\d+) deletions?\(-\)/g

function changed(stat: string): Kept["files"] {
  const rows = parseStat(stat)
  if (rows.length === 0) {
    return null
  }
  let plus = 0
  let minus = 0
  let read = false
  for (const match of stat.matchAll(summary)) {
    read = true
    plus += Number(match[1] ?? 0)
    minus += Number(match[2] ?? 0)
  }
  if (!read) {
    plus = rows.reduce((sum, row) => sum + row.plus, 0)
    minus = rows.reduce((sum, row) => sum + row.minus, 0)
  }
  return { files: rows.length, plus, minus }
}

/**
 * The work that stays whatever the person does next, from data the run already holds. It
 * counts stages, and commits only when the server counted them (`git rev-list base..head`).
 * Null before the first stage is sealed, when nothing is kept yet.
 */
export function keptWork(input: { statements: Statement[]; branch: string; head: string; stat: string; commits?: number }): Kept | null {
  const sealed = new Set(input.statements.filter((item) => item.verdict === "advance").map((item) => item.stage))
  if (sealed.size === 0) {
    return null
  }
  return { sealed: sealed.size, branch: input.branch, head: input.head.slice(0, 7), commits: input.commits ?? 0, files: changed(input.stat) }
}
