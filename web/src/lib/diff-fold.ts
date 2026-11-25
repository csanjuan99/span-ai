import type { DiffLine } from "@/lib/diff"

/** A stretch of a hunk: lines shown as they are, or a run of unchanged lines folded away. */
export type Segment =
  { kind: "lines"; lines: DiffLine[] } | { kind: "fold"; lines: DiffLine[] }

/** How many unchanged lines stay visible on each side of a change. */
export const keepAround = 3

/**
 * Folds the long stretches of unchanged lines of a hunk. A run of context longer than twice
 * the lines kept around a change collapses to a fold the reader can open; shorter runs stay.
 */
export function foldContext(lines: DiffLine[]): Segment[] {
  const segments: Segment[] = []
  let index = 0
  while (index < lines.length) {
    const start = index
    const context = lines[index]?.kind === "ctx"
    while (index < lines.length && (lines[index]?.kind === "ctx") === context) {
      index += 1
    }
    const run = lines.slice(start, index)
    if (!context || run.length <= keepAround * 2 + 1) {
      segments.push({ kind: "lines", lines: run })
      continue
    }
    const first = start === 0
    const last = index === lines.length
    const head = first ? 0 : keepAround
    const tail = last ? 0 : keepAround
    if (head > 0) {
      segments.push({ kind: "lines", lines: run.slice(0, head) })
    }
    segments.push({ kind: "fold", lines: run.slice(head, run.length - tail) })
    if (tail > 0) {
      segments.push({ kind: "lines", lines: run.slice(run.length - tail) })
    }
  }
  return segments
}

/** The file with only its first `max` diff lines, hunks past the cut dropped, for a file too large to show whole. */
export function limitLines<
  T extends { hunks: { header: string; section: string; lines: DiffLine[] }[] },
>(file: T, max: number): T {
  let left = max
  const hunks = []
  for (const hunk of file.hunks) {
    if (left <= 0) {
      break
    }
    hunks.push({ ...hunk, lines: hunk.lines.slice(0, left) })
    left -= hunk.lines.length
  }
  return { ...file, hunks }
}
