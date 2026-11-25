import type { DiffFile, FileStatus } from "@/lib/diff"
import { kindOf, type Kind } from "@/lib/report"

/** How many files changed in each way. */
export interface StatusCounts {
  added: number
  modified: number
  renamed: number
  deleted: number
}

/** Counts the files by what happened to them. */
export function statusCounts(files: { status: FileStatus }[]): StatusCounts {
  const counts: StatusCounts = { added: 0, modified: 0, renamed: 0, deleted: 0 }
  for (const file of files) {
    counts[file.status] += 1
  }
  return counts
}

const testLine =
  /^\s*(?:func\s+Test\w*\(|(?:it|test|describe)\s*\(|(?:async\s+)?def\s+test_\w*\(|#\[test\]|@Test\b)/

/** The tests the change adds: lines that open a test in the files it wrote for tests, and which files hold them. */
export function testsAdded(files: DiffFile[]): {
  count: number
  files: string[]
} {
  let count = 0
  const holders: string[] = []
  for (const file of files) {
    if (kindOf(file.path) !== "test") {
      continue
    }
    const here = file.hunks.reduce(
      (sum, hunk) =>
        sum +
        hunk.lines.filter(
          (line) => line.kind === "add" && testLine.test(line.text)
        ).length,
      0
    )
    if (here > 0) {
      count += here
      holders.push(file.path)
    }
  }
  return { count, files: holders }
}

const order: Kind[] = [
  "source",
  "test",
  "docs",
  "config",
  "deps",
  "ci",
  "generated",
]

/** The labels the overview uses for each family, in the order the design lists them. */
export const purposeLabels: Record<Kind, string> = {
  source: "Source",
  test: "Tests",
  docs: "Docs",
  config: "Config",
  deps: "Dependencies",
  ci: "Build and deploy",
  generated: "Generated",
  other: "Other",
}

/** One bar of "Changes by purpose": the share of changed lines a family holds. */
export interface PurposeShare {
  kind: Kind
  label: string
  files: number
  added: number
  removed: number
  percent: number
}

/** Every family in the design's order, with its share of the changed lines; "other" shows only when something is in it. */
export function purposeShares(
  groups: { kind: Kind; files: string[]; added: number; removed: number }[]
): PurposeShare[] {
  const total = groups.reduce(
    (sum, group) => sum + group.added + group.removed,
    0
  )
  const kinds = groups.some((group) => group.kind === "other")
    ? [...order, "other" as const]
    : order
  return kinds.map((kind) => {
    const group = groups.find((item) => item.kind === kind)
    const lines = group === undefined ? 0 : group.added + group.removed
    return {
      kind,
      label: purposeLabels[kind],
      files: group?.files.length ?? 0,
      added: group?.added ?? 0,
      removed: group?.removed ?? 0,
      percent: total === 0 ? 0 : Math.round((lines / total) * 100),
    }
  })
}
