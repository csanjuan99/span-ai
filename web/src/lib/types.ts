/** What a check's own output said about its tests. Absent when it printed no count. */
export interface TestCounts {
  passed: number
  failed: number
  skipped: number
}

/** One sealed stage the page can read without the engine types. */
export interface StatementCommand {
  command: string
  exitCode: number
  error?: string
  band?: string
  /** Short tail of what the command printed. The failure is at the end. */
  output?: string
  /** Run-relative path to the full output. */
  logPath?: string
  fingerprint?: string
  tests?: TestCounts
}

/** The control record for one stage. */
export interface Statement {
  stage: string
  iteration: number
  verdict: string
  reason: string
  time?: string
  parentDigest?: string
  gitSHA?: string
  changeURL?: string
  commands: StatementCommand[]
  digest?: string
  fingerprint?: string
  /** light, standard, or deep. It decides how much machinery the run paid for. */
  weight?: string
  /** The same command passed and failed on this commit. */
  inconclusive?: boolean
  /** How many repairs this run may start. */
  cap?: number
}

/** One quoted source of a decision: a file and line, or the output of a check. */
export interface DecisionEvidence {
  ref: string
  excerpt: string
}

/** One answer to a decision, with what Span does when it is chosen. */
export interface DecisionOption {
  id: string
  label: string
  consequence: string
  reversible: boolean
}

/** A question the run cannot go on without. */
export interface DecisionRequest {
  id: string
  class: string
  stage: string
  title: string
  happened: string
  evidence: DecisionEvidence[]
  options: DecisionOption[]
  recommendation: string
}

/** Where the run is standing. */
export interface Cursor {
  id: string
  stage: string
  iteration: number
  hold: boolean
  stop: boolean
  done: boolean
  reason: string
  gitSHA: string
  changeURL: string
  branch: string
  base: string
  weight?: string
  workspace: string
  issue: string
  questions: string[]
  /** One row per failing check that was sent back to build. */
  repairs?: Repair[]
  /** The agent CLI that cannot work until a person acts. */
  agentBlock?: AgentBlock | null
  /** The merge conflict the merge stage is resolving. Absent when there is none. */
  conflict?: Conflict
  /** The decision the run waits on. While it is open, only an answer moves the run. */
  decision?: DecisionRequest
}

/**
 * Where a conflict resolution stands. Only `choosing` offers cards: before it the agent
 * is still reading both sides, after it the branch is being written.
 */
export const conflictStates = ["proposing", "choosing", "applying", "failed"] as const

/** One of {@link conflictStates}. */
export type ConflictState = (typeof conflictStates)[number]

/** The git shapes a conflict group can take. `other` is what an unknown kind degrades to. */
export const conflictKinds = [
  "content",
  "add-add",
  "modify-delete",
  "rename",
  "mode",
  "binary",
  "lockfile",
  "generated",
  "other",
] as const

/** One of {@link conflictKinds}. */
export type ConflictKind = (typeof conflictKinds)[number]

/** How much an option can break. An unknown risk degrades to `high`, the safe reading. */
export const conflictRisks = ["low", "medium", "high"] as const

/** One of {@link conflictRisks}. */
export type ConflictRisk = (typeof conflictRisks)[number]

/** What an option writes. An unknown action degrades to `keep`. */
export const conflictActions = ["ours", "theirs", "combine", "regenerate", "delete", "keep"] as const

/** One of {@link conflictActions}. */
export type ConflictAction = (typeof conflictActions)[number]

/** One way to resolve a group, with the exact diff it would write. */
export interface ConflictOption {
  id: string
  title: string
  summary: string
  risk: ConflictRisk
  action: ConflictAction
  /** Set only for `regenerate`: the command that runs after the person applies. */
  command: string
  /** Unified diff of what the option writes. */
  diff: string
}

/** Files that conflict for the same reason, and the options the agent proposed for them. */
export interface ConflictGroup {
  id: string
  kind: ConflictKind
  files: string[]
  summary: string
  options: ConflictOption[]
  /** The option id once the person picked one; empty before. */
  chosen: string
}

/** A merge of the base into the run's branch that git could not finish alone. */
export interface Conflict {
  base: string
  baseSHA: string
  state: ConflictState
  note: string
  groups: ConflictGroup[]
}

/** One agent CLI Span can drive, and whether it can run today. */
export interface AgentState {
  id: string
  label: string
  binary: string
  installed: boolean
  gentle: boolean
  ready: boolean
  reason?: string
}

/** One run as the desk lists it: no statements, just the label and where it stands. */
export interface SpanHead {
  id: string
  label: string
  issue: string
  repo: string
  stage: string
  verdict: string
}

/** The agent section of the desk: what is known, what was asked, what would run. */
export interface AgentChoice {
  agents: AgentState[]
  selected: string
  chosen: string
  exists: boolean
  reason: string
  /** The model chosen for each CLI id. A CLI without an entry keeps the one Gentle AI set. */
  models: Record<string, string>
}

/** One model a CLI accepts. */
export interface ModelOption {
  id: string
  label: string
}

/** What the desk can offer for one CLI. `free` means any valid model name may be typed. */
export interface ModelList {
  models: ModelOption[]
  free: boolean
  source: "cli" | "fixed" | "none"
  error: string
}

/** An agent CLI that ran out of quota or credentials. */
export interface AgentBlock {
  kind: string
  agent: string
  detail?: string
  time?: string
}

/** One failing check sent back to build. */
export interface Repair {
  iteration: number
  stage: string
  fingerprint?: string
  gitSHA?: string
  reason?: string
  time?: string
}

/** Acceptance line stored on the run before the gate names a file. */
export interface AcceptanceLine {
  id: string
  text: string
}

/** Gate stored beside the run. */
export interface Gate {
  verdict: string
  command: string
  exitCode: number
  /** junit, vitest-json, prove, or exit-only. An exit code alone is not evidence. */
  evidence?: string
  report?: string
  reason?: string
  checks?: { id: string; result: string }[]
}

/** Durable reading of one run. */
export interface Snapshot {
  cursor: Cursor
  statements: Statement[]
  acceptance: AcceptanceLine[]
  gate: Gate | null
  /** The run's per-stage CLI choice, when the page set one. */
  stageAgents?: Record<string, string>
  /** {JIRA_KEY}-{SessionId}-{REPO_NAME}. What the page shows instead of the bare id. */
  label?: string
}

/** One acceptance case with the declaration that names it. */
export interface QACase {
  id: string
  text: string
  result: string
  file: string
  line: number
}

/** Acceptance reading for the QA tab. */
export interface AcceptanceView {
  skipped: boolean
  verdict: string
  command: string
  exitCode: number
  lines: QACase[]
}

/** Files changed from base to head. */
export interface ImplementationView {
  stat: string
  patch: string
  truncated: boolean
  /** Commits the branch added to its base; 0 when the server did not count. */
  commits: number
}

/** One socket frame. The span field replaces the page on reconnect. */
export interface LiveFrame {
  type: string
  cursor?: Cursor
  statement?: Statement
  gate?: Gate
  stage?: string
  lines?: string[]
  transcript?: string[]
  span?: Snapshot
  /** The `conflict` frame: the resolution changed state. Absent means it was cleared. */
  conflict?: Conflict
  /** Orders the frames of one run; a replay names the seq the stream resumes after. */
  seq?: number
  /** transcript.log size after this frame's lines; on a replay, where its tail ends. */
  offset?: number
}

/** What the latest agent review found for one run. */
export interface AgentStatus {
  id: string
  running: boolean
  relaunch: boolean
  name: string
  pid: number
  stage: string
  elapsed: string
  summary: string
}

/** The ten stages, in the order the engine walks them. */
export const stages = [
  "intake",
  "frame",
  "build",
  "prove",
  "qa",
  "inspect",
  "open",
  "merge",
  "promote",
  "note",
] as const

/** The branches a run may start from, and the base an empty field means (span.yaml's base, else dev). */
export interface BranchChoice {
  branches: string[]
  defaultBase: string
}

/** Where a fact in the example map came from. */
export const shapeSourceKinds = ["prompt", "answer", "code", "context"] as const

/** The states of one question. */
export const shapeQuestionStates = ["open", "answered", "assumed"] as const

/** The states of one product-context proposal. */
export const shapeProposalStates = ["proposed", "accepted"] as const

/** The states of a shaping session. */
export const shapeStates = ["interviewing", "ready", "split", "sealed"] as const

/** Where a fact in the example map came from. */
export interface ShapeSource {
  kind: (typeof shapeSourceKinds)[number]
  ref: string
}

export interface ShapeRule {
  id: string
  text: string
  source: ShapeSource
}

export interface ShapeExample {
  id: string
  /** The id of the rule the example illustrates. */
  rule: string
  text: string
  source: ShapeSource
}

export interface ShapeQuestion {
  id: string
  text: string
  blocks: string
  assume: string
  status: (typeof shapeQuestionStates)[number]
  answer: string
}

export interface ShapeSplit {
  story: string
  leftOut: string
}

export interface ShapeProposal {
  id: string
  section: string
  text: string
  status: (typeof shapeProposalStates)[number]
}

export interface ShapeJira {
  key: string
  url: string
  status: string
}

/** One shaping session, as the server keeps it. */
export interface ShapeSession {
  id: string
  workspace: string
  repo: string
  idea: string
  status: (typeof shapeStates)[number]
  rules: ShapeRule[]
  examples: ShapeExample[]
  questions: ShapeQuestion[]
  split: ShapeSplit[]
  proposals: ShapeProposal[]
  story: string
  digest: string
  jira: ShapeJira | null
}

/** What creating the story in Jira would do, shown before anything is created. */
export interface ShapePreview {
  project: string
  board: { id: number; name: string }
  backlogStatuses: string[]
  summary: string
  description: string
}
