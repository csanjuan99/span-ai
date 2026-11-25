import type { Conflict, ConflictAction, ConflictKind, ConflictRisk } from "@/lib/types"

/** The option id a person picked, by group id. A group without a key is not chosen yet. */
export type ConflictChoices = Readonly<Record<string, string>>

/** One `group=<id>&option=<id>` pair of the choose request. */
export interface ConflictPair {
  group: string
  option: string
}

/** How a conflict hold reads: the person must choose, or the merge was aborted. */
export type ConflictHoldKind = "choose" | "aborted"

/** Every hold the merge-conflict phase takes starts with this, so the page can route it. */
const holdPrefix = "conflict:"

/** The reason the server writes after `git merge --abort`. */
const abortedReason = "conflict: aborted"

function chosenOption(conflict: Conflict, choices: ConflictChoices, groupId: string): string {
  const group = conflict.groups.find((item) => item.id === groupId)
  const picked = choices[groupId] ?? ""
  if (group === undefined || picked === "") {
    return ""
  }
  return group.options.some((item) => item.id === picked) ? picked : ""
}

/**
 * How many groups still need a choice. A choice naming an option the group does not
 * offer (a re-proposal replaced it) does not count: the server would reject it.
 */
export function remaining(conflict: Conflict, choices: ConflictChoices): number {
  return conflict.groups.filter((group) => chosenOption(conflict, choices, group.id) === "").length
}

/** Apply is allowed only while choosing and with a valid choice for every group. */
export function canApply(conflict: Conflict, choices: ConflictChoices): boolean {
  if (conflict.state !== "choosing") {
    return false
  }
  return remaining(conflict, choices) === 0
}

/** The pairs to post, in group order, skipping groups without a valid choice. */
export function choicePairs(conflict: Conflict, choices: ConflictChoices): ConflictPair[] {
  return conflict.groups
    .map((group) => ({ group: group.id, option: chosenOption(conflict, choices, group.id) }))
    .filter((pair) => pair.option !== "")
}

const riskLabels: Record<ConflictRisk, string> = {
  low: "low risk",
  medium: "medium risk",
  high: "high risk",
}

/** The risk in words, so it never depends on color alone. */
export function riskLabel(risk: ConflictRisk): string {
  return riskLabels[risk]
}

const actionLabels: Record<ConflictAction, string> = {
  ours: "Keep this branch's version",
  theirs: "Take the base's version",
  combine: "Combine both",
  regenerate: "Regenerate",
  delete: "Delete",
  keep: "Keep",
}

/** What an option writes, in the words a person reads instead of git's ours/theirs. */
export function actionLabel(action: ConflictAction): string {
  return actionLabels[action]
}

const kindLabels: Record<ConflictKind, string> = {
  content: "both changed",
  "add-add": "both added",
  "modify-delete": "changed and deleted",
  rename: "renamed",
  mode: "file mode",
  binary: "binary",
  lockfile: "lockfile",
  generated: "generated",
  other: "other",
}

/** The git shape of a group, in words. */
export function kindLabel(kind: ConflictKind): string {
  return kindLabels[kind]
}

/** How one diff line is drawn. */
export type DiffLineKind = "add" | "del" | "context" | "header"

/** One classified line of a unified diff. */
export interface DiffLine {
  kind: DiffLineKind
  text: string
}

const headerStarts = [
  "diff ",
  "index ",
  "--- ",
  "+++ ",
  "@@",
  "new file",
  "deleted file",
  "old mode",
  "new mode",
  "similarity",
  "rename ",
  "Binary files",
]

function lineKind(line: string): DiffLineKind {
  if (headerStarts.some((start) => line.startsWith(start))) {
    return "header"
  }
  if (line.startsWith("+")) {
    return "add"
  }
  if (line.startsWith("-")) {
    return "del"
  }
  return "context"
}

/** Splits a unified diff into lines the card can style. The trailing newline adds no line. */
export function diffLines(diff: string): DiffLine[] {
  if (diff === "") {
    return []
  }
  const lines = diff.endsWith("\n") ? diff.slice(0, -1).split("\n") : diff.split("\n")
  return lines.map((text) => ({ kind: lineKind(text), text }))
}

/** Whether a hold belongs to the merge-conflict phase, not to a failed check. */
export function isConflictHold(reason: string): boolean {
  return reason.startsWith(holdPrefix)
}

/** Which conflict hold this is, or null for any other reason. */
export function conflictHold(reason: string): ConflictHoldKind | null {
  if (!isConflictHold(reason)) {
    return null
  }
  return reason === abortedReason ? "aborted" : "choose"
}

/**
 * A fingerprint of what the server last said about the conflict. A pending action waits
 * until the socket reports something different.
 */
export function conflictSignature(conflict: Conflict): string {
  const chosen = conflict.groups.map((group) => `${group.id}=${group.chosen}`)
  return [conflict.state, conflict.note, proposalKey(conflict), ...chosen].join("|")
}

/**
 * Which proposal the cards show: the groups and their option ids. A new proposal (after
 * a hint) gets new cards, so a choice made on the old ones does not carry over.
 */
export function proposalKey(conflict: Conflict): string {
  return conflict.groups.map((group) => `${group.id}:${group.options.map((item) => item.id).join(",")}`).join(";")
}

const hunkStart = /^@@ -(\d+)(?:,\d+)? \+(\d+)/

/**
 * The line number each diff line carries in the margin: the old side for a removed line, the
 * new side for the rest. Headers carry none, and neither does anything before the first hunk.
 */
export function lineNumbers(lines: readonly DiffLine[]): (number | null)[] {
  let before = 0
  let after = 0
  return lines.map((line) => {
    const hunk = hunkStart.exec(line.text)
    if (hunk !== null) {
      before = Number(hunk[1])
      after = Number(hunk[2])
      return null
    }
    if (line.kind === "header" || after === 0) {
      return null
    }
    if (line.kind === "del") {
      return before++
    }
    before += 1
    return after++
  })
}

/** How many files a conflict holds across its groups. */
export function conflictFileCount(conflict: Conflict): number {
  return conflict.groups.reduce((total, group) => total + group.files.length, 0)
}

/**
 * For each option's diff, the added and removed lines the other options do not all share.
 * Comparing options side by side highlights these, so the differences stand out from the
 * hunk header and context they have in common.
 */
export function differingLines(diffs: readonly string[]): ReadonlySet<string>[] {
  const changed = diffs.map((diff) => new Set(diffLines(diff).filter((line) => line.kind === "add" || line.kind === "del").map((line) => line.text)))
  return changed.map((mine, at) => new Set([...mine].filter((text) => changed.some((other, to) => to !== at && !other.has(text)))))
}
