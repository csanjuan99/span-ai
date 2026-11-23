import { splitPath, type DiffFile } from "@/lib/diff"
import type { FeatureReport } from "@/lib/featuredoc"
import type { Cursor, Statement } from "@/lib/types"

/** How worried a person should be. */
export type Level = "high" | "medium" | "low"

/** Where a line of the report came from, so a reader knows how far to trust it. */
export type Origin = "detected" | "reported" | "verified"

/** One thing that could hurt, with the files it points at. */
export interface Risk {
  id: string
  level: Level
  origin: Origin
  title: string
  detail: string
  files: string[]
}

/** The family a changed file belongs to, for the "what changed" summary. */
export type Kind = "source" | "test" | "docs" | "config" | "deps" | "ci" | "generated" | "other"

/** How the change is bucketed by purpose. */
export interface Group {
  kind: Kind
  files: string[]
  added: number
  removed: number
}

/** The checks Span ran and what the inspector said. */
export interface Evidence {
  commands: { stage: string; command: string; exitCode: number; ok: boolean }[]
  inspect: { verdict: string; reason: string } | null
  sentBack: number
  repairs: number
  flaky: boolean
}

/** What the reader should do with this change, from the level. */
export interface Verdict {
  level: Level
  headline: string
  depth: string
}

/** The whole report. */
export interface Report {
  verdict: Verdict
  risks: Risk[]
  groups: Group[]
  totals: { files: number; added: number; removed: number }
  evidence: Evidence
  /** Things the report could not check, so silence is never read as safety. */
  gaps: string[]
}

const codeExt = /\.(go|ts|tsx|js|jsx|mjs|cjs|py|rb|rs|java|kt|swift|c|cc|cpp|h|hpp|cs|php|sh|sql)$/i
const testPath = /(^|\/)(__tests__|tests?|spec|e2e)\/|(_test|\.test|\.spec|_spec)\.[a-z]+$|(^|\/)test_[^/]+\.py$/i
const docsPath = /\.(md|mdx|rst|txt)$|(^|\/)docs?\//i
const depsPath =
  /(^|\/)(go\.(mod|sum)|package(-lock)?\.json|pnpm-lock\.yaml|yarn\.lock|requirements[^/]*\.txt|poetry\.lock|Pipfile(\.lock)?|Cargo\.(toml|lock)|Gemfile(\.lock)?|composer\.(json|lock))$/
const ciPath =
  /(^|\/)(\.github\/workflows\/|\.gitlab-ci\.yml|Jenkinsfile|Dockerfile[^/]*|docker-compose[^/]*\.ya?ml|\.circleci\/|deploy\/|k8s\/|helm\/|terraform\/)|\.tf$/i
const configPath = /(^|\/)(\.env[^/]*|[^/]*\.(ya?ml|toml|ini|conf|cfg)|config\/[^/]+|tsconfig[^/]*\.json)$/i
const generatedPath =
  /(^|\/)(__pycache__|node_modules|dist|build|\.next|coverage|vendor)\/|\.(pyc|pyo|class|o|so|dll|exe|min\.js|map)$|(^|\/)\.DS_Store$/i
const authPath = /(auth|login|logout|session|token|secret|credential|passw|permission|acl|rbac|crypto|oauth|jwt|sso)/i
const migrationPath = /(^|\/)(migrations?|db\/migrate|schema)\/|\.sql$/i

/** The family of one path. The first match wins, so a test in a docs folder is still a test. */
export function kindOf(path: string): Kind {
  if (generatedPath.test(path)) {
    return "generated"
  }
  if (testPath.test(path)) {
    return "test"
  }
  if (ciPath.test(path)) {
    return "ci"
  }
  if (depsPath.test(path)) {
    return "deps"
  }
  if (docsPath.test(path)) {
    return "docs"
  }
  if (codeExt.test(path)) {
    return "source"
  }
  return configPath.test(path) ? "config" : "other"
}

const secretShapes: { name: string; pattern: RegExp }[] = [
  { name: "an AWS access key id", pattern: /AKIA[0-9A-Z]{16}/ },
  { name: "a private key block", pattern: /-----BEGIN [A-Z ]*PRIVATE KEY-----/ },
  { name: "a GitHub token", pattern: /\bgh[pousr]_[A-Za-z0-9]{30,}/ },
  { name: "a hard-coded credential", pattern: /(api[_-]?key|secret|passw(or)?d|token)\s*[:=]\s*["'][^"'\s]{12,}["']/i },
]

function addedText(file: DiffFile): string[] {
  return file.hunks.flatMap((hunk) => hunk.lines.filter((line) => line.kind === "add").map((line) => line.text))
}

function risk(id: string, level: Level, title: string, detail: string, files: string[], origin: Origin = "detected"): Risk {
  return { id, level, origin, title, detail, files }
}

/** The three size bands: past 400 changed lines review stops finding defects (SmartBear/Cisco), and past about 100 Google calls a change big. */
export const sizeBands = { medium: 200, high: 400, manyFiles: 10 } as const

function sizeRisks(files: DiffFile[], changed: number): Risk[] {
  const out: Risk[] = []
  if (changed > sizeBands.high) {
    out.push(
      risk("size", "high", `${changed} changed lines`, "Review stops finding defects past about 400 lines at a time. Read it in parts.", []),
    )
  } else if (changed > sizeBands.medium) {
    out.push(risk("size", "medium", `${changed} changed lines`, "Large for one sitting. Google calls about 100 lines reasonable.", []))
  }
  if (files.length > sizeBands.manyFiles) {
    out.push(risk("spread", "medium", `${files.length} files touched`, "Changes that spread over many files hide interactions.", []))
  }
  return out
}

function pathRisks(files: DiffFile[]): Risk[] {
  const named = (match: (file: DiffFile) => boolean) => files.filter(match).map((file) => file.path)
  const rules: { id: string; level: Level; title: string; detail: string; files: string[] }[] = [
    {
      id: "auth",
      level: "high",
      title: "Touches authentication or secrets",
      detail: "A mistake here exposes data or access. Read these lines, not the summary.",
      files: named((file) => kindOf(file.path) === "source" && authPath.test(splitPath(file.path).name)),
    },
    {
      id: "migration",
      level: "high",
      title: "Changes the database schema or data",
      detail: "Hard to undo once it runs against real data. Check the way back.",
      files: named((file) => migrationPath.test(file.path)),
    },
    {
      id: "ci",
      level: "high",
      title: "Changes build, deploy or infrastructure",
      detail: "Affects every later run, not only this feature.",
      files: named((file) => kindOf(file.path) === "ci"),
    },
    {
      id: "env",
      level: "high",
      title: "Touches an environment file",
      detail: "Environment files often hold secrets and should not be committed.",
      files: named((file) => /(^|\/)\.env[^/]*$/.test(file.path)),
    },
    {
      id: "deps",
      level: "medium",
      title: "Changes dependencies",
      detail: "New code you did not review enters the build.",
      files: named((file) => kindOf(file.path) === "deps"),
    },
    {
      id: "config",
      level: "medium",
      title: "Changes configuration",
      detail: "Behavior changes without a code change.",
      files: named((file) => kindOf(file.path) === "config"),
    },
    {
      id: "generated",
      level: "medium",
      title: "Commits generated or binary files",
      detail: "Build output and caches do not belong in history.",
      files: named((file) => kindOf(file.path) === "generated" || file.binary),
    },
    {
      id: "tests-deleted",
      level: "high",
      title: "Deletes a test file",
      detail: "A removed test can hide the regression it used to catch.",
      files: named((file) => kindOf(file.path) === "test" && file.status === "deleted"),
    },
  ]
  return rules.filter((rule) => rule.files.length > 0).map((rule) => risk(rule.id, rule.level, rule.title, rule.detail, rule.files))
}

function contentRisks(files: DiffFile[]): Risk[] {
  const out: Risk[] = []
  const secrets = new Map<string, Set<string>>()
  const todo: string[] = []
  for (const file of files) {
    for (const text of addedText(file)) {
      for (const shape of secretShapes) {
        if (shape.pattern.test(text)) {
          secrets.set(shape.name, (secrets.get(shape.name) ?? new Set()).add(file.path))
        }
      }
      if (/\b(TODO|FIXME|HACK|XXX)\b/.test(text) && !todo.includes(file.path)) {
        todo.push(file.path)
      }
    }
  }
  for (const [name, paths] of secrets) {
    out.push(risk("secret", "high", `Looks like ${name}`, "A match on added lines. Confirm it is not real and rotate it if it is.", [...paths]))
  }
  if (todo.length > 0) {
    out.push(risk("todo", "low", "Adds TODO or FIXME markers", "Unfinished work is being merged.", todo))
  }
  return out
}

function testRisks(groups: Group[]): Risk[] {
  const source = groups.find((group) => group.kind === "source")
  const tests = groups.find((group) => group.kind === "test")
  if (source === undefined || tests !== undefined) {
    return []
  }
  return [risk("untested", "medium", "Source changed and no test did", "Nothing in this change would catch a regression in the code it edits.", source.files)]
}

function runRisks(statements: Statement[], cursor: Cursor): Risk[] {
  const out: Risk[] = []
  const flaky = statements.filter((item) => item.inconclusive === true)
  if (flaky.length > 0) {
    out.push(
      risk("flaky", "medium", "A check passed and failed on the same commit", "The result cannot be attributed to the change.", [], "verified"),
    )
  }
  const sentBack = statements.filter((item) => item.stage === "inspect" && item.verdict !== "advance").length
  if (sentBack >= 2) {
    out.push(
      risk("sent-back", "medium", `Inspect sent this back ${sentBack} times`, "Each return was a finding the earlier attempt missed. Check the last fix did not trade one problem for another.", [], "verified"),
    )
  }
  const stopped = [...statements].reverse().find((item) => item.stage === "inspect" && item.verdict !== "advance")
  if (stopped !== undefined && cursor.reason.startsWith("inspect cap:")) {
    out.push(risk("inspect-cap", "high", "Inspect still blocks this change", stopped.reason, [], "verified"))
  }
  return out
}

/** Buckets the files by purpose, biggest first. */
export function groupFiles(files: DiffFile[]): Group[] {
  const by = new Map<Kind, Group>()
  for (const file of files) {
    const kind = kindOf(file.path)
    const group = by.get(kind) ?? { kind, files: [], added: 0, removed: 0 }
    group.files.push(file.path)
    group.added += file.added
    group.removed += file.removed
    by.set(kind, group)
  }
  return [...by.values()].sort((a, b) => b.added + b.removed - (a.added + a.removed))
}

/** The last statement of a stage: earlier rounds were about earlier commits. */
function lastOf(statements: Statement[], stage: string): Statement | undefined {
  return [...statements].reverse().find((item) => item.stage === stage)
}

function evidenceOf(statements: Statement[]): Evidence {
  const commands = [lastOf(statements, "prove"), lastOf(statements, "qa")]
    .filter((item): item is Statement => item !== undefined)
    .flatMap((item) =>
      item.commands.map((command) => ({
        stage: item.stage,
        command: command.command,
        exitCode: command.exitCode,
        ok: command.exitCode === 0 && (command.error ?? "") === "",
      })),
    )
  const inspects = statements.filter((item) => item.stage === "inspect")
  const last = inspects[inspects.length - 1]
  return {
    commands,
    inspect: last === undefined ? null : { verdict: last.verdict, reason: last.reason },
    sentBack: inspects.filter((item) => item.verdict !== "advance").length,
    repairs: statements.filter((item) => item.stage === "prove" && item.verdict === "iterate").length,
    flaky: statements.some((item) => item.inconclusive === true),
  }
}

function gapsOf(
  files: DiffFile[],
  truncated: boolean,
  evidence: Evidence,
  feature: FeatureReport | null,
  groups: Group[],
): string[] {
  const gaps: string[] = []
  if (files.length === 0) {
    gaps.push("No commit yet, so there is nothing to assess.")
  }
  if (truncated) {
    gaps.push("The patch was cut, so content checks (secrets, TODO) only covered the part that was read.")
  }
  if (evidence.commands.length === 0 && files.length > 0) {
    gaps.push("No prove or QA command has run on this change.")
  }
  if (evidence.inspect === null && files.length > 0) {
    gaps.push("Inspect has not read this change yet.")
  }
  if (feature === null || !feature.hasLog) {
    gaps.push("The agent left no feature document log, so there is no stated reasoning to read.")
  } else if (feature.decisions.length === 0 && feature.risks.length === 0 && groups.length > 0) {
    gaps.push("The agent's log has no Decision or Risk lines.")
  }
  return gaps
}

const rank: Record<Level, number> = { high: 0, medium: 1, low: 2 }

function verdictOf(risks: Risk[], changed: number): Verdict {
  const level: Level = risks.some((item) => item.level === "high") ? "high" : risks.some((item) => item.level === "medium") ? "medium" : "low"
  const count = (wanted: Level) => risks.filter((item) => item.level === wanted).length
  if (level === "high") {
    return { level, headline: `${count("high")} high-risk ${count("high") === 1 ? "signal" : "signals"}`, depth: "Read every flagged file line by line before approving." }
  }
  if (level === "medium") {
    return { level, headline: `${count("medium")} to check`, depth: "Read the flagged files. The rest can be skimmed." }
  }
  return {
    level,
    headline: changed === 0 ? "Nothing to assess yet" : "No risk signals",
    depth: changed === 0 ? "Wait for the first commit." : "A quick look is enough. Absence of signals is not proof of safety.",
  }
}

/** Builds the report from what Span already has. Nothing here calls a model. */
export function buildReport(input: {
  files: DiffFile[]
  truncated: boolean
  statements: Statement[]
  cursor: Cursor
  feature: FeatureReport | null
}): Report {
  const { files, truncated, statements, cursor, feature } = input
  const added = files.reduce((sum, file) => sum + file.added, 0)
  const removed = files.reduce((sum, file) => sum + file.removed, 0)
  const groups = groupFiles(files)
  const evidence = evidenceOf(statements)
  const risks = [
    ...sizeRisks(files, added + removed),
    ...pathRisks(files),
    ...contentRisks(files),
    ...testRisks(groups),
    ...runRisks(statements, cursor),
  ].sort((a, b) => rank[a.level] - rank[b.level])
  return {
    verdict: verdictOf(risks, added + removed),
    risks,
    groups,
    totals: { files: files.length, added, removed },
    evidence,
    gaps: gapsOf(files, truncated, evidence, feature, groups),
  }
}
