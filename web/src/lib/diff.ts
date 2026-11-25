/** What a diff line is: an addition, a removal, unchanged context, or a note such as "No newline". */
export type LineKind = "add" | "del" | "ctx" | "note"

/** One line of a hunk with its line numbers on each side. A side that does not apply is null. */
export interface DiffLine {
  kind: LineKind
  oldNo: number | null
  newNo: number | null
  text: string
}

/** A run of changed lines, introduced by an `@@ -a,b +c,d @@` header. */
export interface Hunk {
  header: string
  /** The function or section git printed after the second `@@`. */
  section: string
  lines: DiffLine[]
}

/** How a file changed between base and head. */
export type FileStatus = "modified" | "added" | "deleted" | "renamed"

/** One file of a patch. */
export interface DiffFile {
  path: string
  oldPath: string
  status: FileStatus
  binary: boolean
  added: number
  removed: number
  hunks: Hunk[]
}

/** One row of `git diff --stat`. */
export interface StatRow {
  path: string
  changes: number
  plus: number
  minus: number
}

const hunkHead = /^@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@ ?(.*)$/

function unquote(path: string): string {
  return path.replace(/^"|"$/g, "")
}

/** Parses a unified patch. A patch cut mid-file keeps what was read so far. */
export function parseDiff(patch: string): DiffFile[] {
  const files: DiffFile[] = []
  let file: DiffFile | undefined
  let hunk: Hunk | undefined
  let oldNo = 0
  let newNo = 0
  for (const raw of patch.split("\n")) {
    if (raw.startsWith("diff --git ")) {
      const match = /^diff --git a\/(.+) b\/(.+)$/.exec(raw)
      const oldPath = unquote(match?.[1] ?? "")
      const path = unquote(match?.[2] ?? oldPath)
      file = {
        path,
        oldPath,
        status: "modified",
        binary: false,
        added: 0,
        removed: 0,
        hunks: [],
      }
      files.push(file)
      hunk = undefined
      continue
    }
    if (file === undefined) {
      continue
    }
    const head = hunkHead.exec(raw)
    if (head !== null) {
      oldNo = Number(head[1])
      newNo = Number(head[2])
      hunk = { header: raw, section: head[3] ?? "", lines: [] }
      file.hunks.push(hunk)
      continue
    }
    if (hunk === undefined) {
      if (raw.startsWith("new file mode")) {
        file.status = "added"
      } else if (raw.startsWith("deleted file mode")) {
        file.status = "deleted"
      } else if (raw.startsWith("rename from ")) {
        file.status = "renamed"
        file.oldPath = raw.slice("rename from ".length)
      } else if (
        raw.startsWith("Binary files ") ||
        raw.startsWith("GIT binary patch")
      ) {
        file.binary = true
      }
      continue
    }
    if (raw.startsWith("+")) {
      hunk.lines.push({ kind: "add", oldNo: null, newNo, text: raw.slice(1) })
      newNo += 1
      file.added += 1
    } else if (raw.startsWith("-")) {
      hunk.lines.push({ kind: "del", oldNo, newNo: null, text: raw.slice(1) })
      oldNo += 1
      file.removed += 1
    } else if (raw.startsWith("\\")) {
      hunk.lines.push({
        kind: "note",
        oldNo: null,
        newNo: null,
        text: raw.slice(2),
      })
    } else if (raw !== "" || hunk.lines.length > 0) {
      hunk.lines.push({ kind: "ctx", oldNo, newNo, text: raw.slice(1) })
      oldNo += 1
      newNo += 1
    }
  }
  return files
}

/** The file rows of `git diff --stat`. The summary line is dropped. */
export function parseStat(stat: string): StatRow[] {
  const rows: StatRow[] = []
  for (const line of stat.split("\n")) {
    const match = /^\s*(.+?)\s+\|\s+(\d+)\s*([+-]*)\s*$/.exec(line)
    if (match === null) {
      continue
    }
    const marks = match[3] ?? ""
    rows.push({
      path: match[1] ?? "",
      changes: Number(match[2]),
      plus: [...marks].filter((mark) => mark === "+").length,
      minus: [...marks].filter((mark) => mark === "-").length,
    })
  }
  return rows
}

/** Totals across files, for the one-line summary above the list. */
export function totals(files: { added: number; removed: number }[]): {
  files: number
  added: number
  removed: number
} {
  return {
    files: files.length,
    added: files.reduce((sum, file) => sum + file.added, 0),
    removed: files.reduce((sum, file) => sum + file.removed, 0),
  }
}

/** Splits a path into its folder (with trailing slash) and file name. */
export function splitPath(path: string): { dir: string; name: string } {
  const at = path.lastIndexOf("/")
  return at < 0
    ? { dir: "", name: path }
    : { dir: path.slice(0, at + 1), name: path.slice(at + 1) }
}

/** Files the stat lists that the patch did not reach, because the patch was cut. */
function missingFromPatch(stat: string, shown: DiffFile[]): DiffFile[] {
  const known = new Set(shown.map((file) => file.path))
  return parseStat(stat)
    .filter((row) => !known.has(row.path))
    .map((row) => ({
      path: row.path,
      oldPath: row.path,
      status: "modified" as const,
      binary: false,
      added: row.plus,
      removed: row.minus,
      hunks: [],
    }))
}

/** Every file of the change: the ones the patch holds, then the ones only the stat names. */
export function filesOf(stat: string, patch: string): DiffFile[] {
  const parsed = parseDiff(patch)
  return [...parsed, ...missingFromPatch(stat, parsed)]
}

const statusOfLetter: Record<string, FileStatus> = {
  A: "added",
  M: "modified",
  D: "deleted",
  R: "renamed",
}

/** The status a git letter names; anything unknown reads as modified. */
export function statusOf(letter: string): FileStatus {
  return statusOfLetter[letter] ?? "modified"
}
