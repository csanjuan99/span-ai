import { getJSON, postForm } from "@/lib/api"
import { isRecord, rows, text } from "@/lib/guards"

/** A changed file as the server reads it: status, counts, kind flags and the viewed mark. */
export interface FileEntry {
  path: string
  from: string
  status: "A" | "M" | "D" | "R"
  added: number
  removed: number
  binary: boolean
  generated: boolean
  large: boolean
  viewed: boolean
}

/** One commit of the run's branch with the size of what it changed. */
export interface CommitStat {
  sha: string
  subject: string
  /** Commit time, RFC 3339. */
  time: string
  parent: string
  files: number
  added: number
  removed: number
  renameOnly: boolean
}

/** The diff of one file, whole branch or one commit. An empty patch with a flag means it was held back. */
export interface FileDiff extends FileEntry {
  commit: string
  patch: string
  truncated: boolean
  hunks: number
}

/** The diff of one commit with the files it touched. */
export interface CommitDiff {
  parent: string
  patch: string
  truncated: boolean
  files: FileEntry[]
}

/** A choice the agent recorded, with where it came from and whether a person checked it. */
export interface DecisionEntry {
  id: string
  what: string
  why: string
  source: string
  files: string[]
  assumed: boolean
  checked: boolean
}

/** What the run's feature document says about why the change is as it is. */
export interface Reasoning {
  title: string
  specs: { id: string; text: string }[]
  tasks: { text: string; done: boolean; commit: string }[]
  decisions: DecisionEntry[]
  unchecked: number
  recorded: boolean
}

/** One line of the run's journal with its hashes. */
export interface AuditEntry {
  seq: number
  time: string
  actor: string
  what: string
  hash: string
  prev: string
  want: string
}

/** The journal and whether its hash chain holds. */
export interface Audit {
  entries: AuditEntry[]
  intact: boolean
  brokenAt: number
  head: string
}

function num(value: unknown): number {
  return typeof value === "number" && Number.isFinite(value) ? value : 0
}

function statusOf(value: unknown): FileEntry["status"] {
  return value === "A" || value === "D" || value === "R" ? value : "M"
}

/** Reads one file entry of a `files` reply. */
export function fileEntry(value: unknown): FileEntry {
  const row = isRecord(value) ? value : {}
  return {
    path: text(row.path),
    from: text(row.from),
    status: statusOf(row.status),
    added: num(row.added),
    removed: num(row.removed),
    binary: row.binary === true,
    generated: row.generated === true,
    large: row.large === true,
    viewed: row.viewed === true,
  }
}

/** Reads the `files` reply. Anything that is not the expected shape reads as no files. */
export function fileEntries(value: unknown): FileEntry[] {
  return rows(isRecord(value) ? value.files : null).map(fileEntry)
}

/** Reads the `commits` reply. */
export function commitStats(value: unknown): CommitStat[] {
  return rows(isRecord(value) ? value.commits : null).map((row) => ({
    sha: text(row.sha),
    subject: text(row.subject),
    time: text(row.time),
    parent: text(row.parent),
    files: num(row.files),
    added: num(row.added),
    removed: num(row.removed),
    renameOnly: row.renameOnly === true,
  }))
}

/** Reads the diff of one file. */
export function fileDiff(value: unknown): FileDiff {
  const row = isRecord(value) ? value : {}
  return {
    ...fileEntry(value),
    commit: text(row.commit),
    patch: text(row.patch),
    truncated: row.truncated === true,
    hunks: num(row.hunks),
  }
}

/** Reads the diff of one commit. */
export function commitDiff(value: unknown): CommitDiff {
  const row = isRecord(value) ? value : {}
  return {
    parent: text(row.parent),
    patch: text(row.patch),
    truncated: row.truncated === true,
    files: fileEntries(row),
  }
}

/** Reads the reasoning reply. */
export function reasoning(value: unknown): Reasoning {
  const row = isRecord(value) ? value : {}
  return {
    title: text(row.title),
    specs: rows(row.specs).map((spec) => ({
      id: text(spec.id),
      text: text(spec.text),
    })),
    tasks: rows(row.tasks).map((task) => ({
      text: text(task.text),
      done: task.done === true,
      commit: text(task.commit),
    })),
    decisions: rows(row.decisions).map((item) => ({
      id: text(item.id),
      what: text(item.what),
      why: text(item.why),
      source: text(item.source),
      files: Array.isArray(item.files)
        ? item.files.filter((file): file is string => typeof file === "string")
        : [],
      assumed: item.assumed === true,
      checked: item.checked === true,
    })),
    unchecked: num(row.unchecked),
    recorded: row.recorded === true,
  }
}

/** Reads the audit reply. */
export function audit(value: unknown): Audit {
  const row = isRecord(value) ? value : {}
  return {
    entries: rows(row.entries).map((entry) => ({
      seq: num(entry.seq),
      time: text(entry.time),
      actor: text(entry.actor),
      what: text(entry.what),
      hash: text(entry.hash),
      prev: text(entry.prev),
      want: text(entry.want),
    })),
    intact: row.intact !== false,
    brokenAt: num(row.brokenAt),
    head: text(row.head),
  }
}

const base = (id: string) => `/spans/${encodeURIComponent(id)}`

/** Encodes each segment of a repository path, keeping the slashes the route reads as a path. */
function pathOf(file: string): string {
  return file.split("/").map(encodeURIComponent).join("/")
}

/** The changed files with their flags and viewed marks. */
export async function readFiles(id: string): Promise<FileEntry[]> {
  return fileEntries(await getJSON(`${base(id)}/files`))
}

/** The commits of the branch, newest first. */
export async function readCommits(id: string): Promise<CommitStat[]> {
  return commitStats(await getJSON(`${base(id)}/commits`))
}

/** The diff of one commit. */
export async function readCommitDiff(
  id: string,
  sha: string
): Promise<CommitDiff> {
  return commitDiff(
    await getJSON(`${base(id)}/commits/${encodeURIComponent(sha)}/diff`)
  )
}

/** The diff of one file; force asks for a generated, binary or very large file too. */
export async function readFileDiff(
  id: string,
  file: string,
  options: { commit?: string; force?: boolean } = {}
): Promise<FileDiff> {
  const query = new URLSearchParams()
  if (options.commit !== undefined && options.commit !== "") {
    query.set("commit", options.commit)
  }
  if (options.force === true) {
    query.set("force", "true")
  }
  const suffix = query.size > 0 ? `?${query.toString()}` : ""
  return fileDiff(
    await getJSON(`${base(id)}/files/${pathOf(file)}/diff${suffix}`)
  )
}

/** Marks a file viewed, or takes the mark off. */
export async function markViewed(
  id: string,
  file: string,
  viewed: boolean
): Promise<void> {
  await postForm(
    `${base(id)}/files/${pathOf(file)}/viewed${viewed ? "" : "?viewed=false"}`,
    new URLSearchParams()
  )
}

/** What the run's feature document says about the choices the agent made. */
export async function readReasoning(id: string): Promise<Reasoning> {
  return reasoning(await getJSON(`${base(id)}/reasoning`))
}

/** Marks assumed decisions as checked and answers the reasoning as it is now. */
export async function checkDecisions(
  id: string,
  ids: string[]
): Promise<Reasoning> {
  return reasoning(
    await postForm(
      `${base(id)}/decisions/check`,
      new URLSearchParams({ ids: ids.join(",") })
    )
  )
}

/** The run's journal and whether its chain holds. */
export async function readAudit(id: string): Promise<Audit> {
  return audit(await getJSON(`${base(id)}/audit`))
}
