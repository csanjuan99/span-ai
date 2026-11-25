import { configDefaults } from "@/lib/api"
import type { ConfigEdits, ProjectConfig } from "@/lib/types"

/** The settings whose value is one line of text. */
export const textKeys = [
  "base",
  "open",
  "merge",
  "promote",
  "plan",
  "conflict",
  "qa",
] as const

/** The settings whose value is a list of checks, typed one per line. */
export const listKeys = ["focused", "fast", "acceptance"] as const

/** Every setting the page can change. */
export type DraftKey =
  (typeof textKeys)[number] | (typeof listKeys)[number] | "repair"

/** What a person has typed and not saved. Every value is text, as it sits in its field. */
export type Draft = Partial<Record<DraftKey, string>>

/** The saved value of every setting, as text. */
export function savedDraft(config: ProjectConfig): Record<DraftKey, string> {
  return {
    base: config.base,
    open: config.stages.open,
    merge: config.stages.merge,
    promote: config.stages.promote,
    plan: config.stages.plan,
    conflict: config.stages.conflict,
    qa: config.qa,
    focused: config.prove.focused.join("\n"),
    fast: config.prove.fast.join("\n"),
    acceptance: config.prove.acceptance.join("\n"),
    repair: String(config.repair),
  }
}

/** What span.yaml means for each setting when its key is absent. */
export const defaultDraft: Record<DraftKey, string> = {
  base: "",
  ...configDefaults.stages,
  qa: "",
  focused: "",
  fast: "",
  acceptance: "",
  repair: String(configDefaults.repair),
}

/** The value a field shows: what was typed, else what is saved. */
export function valueOf(
  config: ProjectConfig,
  draft: Draft,
  key: DraftKey
): string {
  return draft[key] ?? savedDraft(config)[key]
}

function checks(raw: string): string[] {
  return raw
    .split("\n")
    .map((line) => line.trim())
    .filter((line) => line !== "")
}

/**
 * The edits a save would send: only the settings whose typed value differs from the saved
 * one. Typing a value back to what is saved is not a change.
 */
export function changedEdits(config: ProjectConfig, draft: Draft): ConfigEdits {
  const saved = savedDraft(config)
  const edits: ConfigEdits = {}
  for (const key of textKeys) {
    const raw = draft[key]
    if (raw !== undefined && raw.trim() !== saved[key]) {
      edits[key] = raw.trim()
    }
  }
  for (const key of listKeys) {
    const raw = draft[key]
    if (raw !== undefined && checks(raw).join("\n") !== saved[key]) {
      edits[key] = checks(raw)
    }
  }
  if (draft.repair !== undefined && draft.repair.trim() !== saved.repair) {
    edits.repair = Number(draft.repair)
  }
  return edits
}

/** How many settings a save would change. */
export function editCount(edits: ConfigEdits): number {
  return Object.keys(edits).length
}
