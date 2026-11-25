import { getJSON } from "@/lib/api"
import { isRecord, rows, strings, text, whole } from "@/lib/guards"

/** What the desk says will happen when a run starts, from the saved settings of the folder. */
export interface StartSummary {
  base: string
  chosenAgent: string
  agentReason: string
  /** Approval of each stage that can wait for a person: auto or hold. */
  stages: Record<string, string>
  checks: string[]
  repair: number
}

/** The summary before the server has answered. */
export const noSummary: StartSummary = {
  base: "",
  chosenAgent: "",
  agentReason: "",
  stages: {},
  checks: [],
  repair: 0,
}

/** A sealed Shape session a run can start from. */
export interface SealedStory {
  id: string
  title: string
}

/** Reads the `/start/what-will-happen` reply; anything unreadable reads as empty. */
export function summaryOf(value: unknown): StartSummary {
  if (!isRecord(value)) {
    return noSummary
  }
  const stages: Record<string, string> = {}
  if (isRecord(value.stages)) {
    for (const [name, approval] of Object.entries(value.stages)) {
      stages[name] = text(approval)
    }
  }
  return {
    base: text(value.base),
    chosenAgent: text(value.chosenAgent),
    agentReason: text(value.agentReason),
    stages,
    checks: strings(value.checks),
    repair: whole(value.repair),
  }
}

/** Reads the `/start/sealed` reply. */
export function sealedOf(value: unknown): SealedStory[] {
  if (!isRecord(value)) {
    return []
  }
  return rows(value.stories).map((row) => ({
    id: text(row.id),
    title: text(row.title),
  }))
}

/** The effective settings a run in this folder would use. */
export async function readSummary(
  folder: string,
  kind: string,
  base: string
): Promise<StartSummary> {
  const query = new URLSearchParams({ path: folder, kind, base })
  return summaryOf(await getJSON(`/start/what-will-happen?${query.toString()}`))
}

/** The sealed stories of this folder. */
export async function readSealed(folder: string): Promise<SealedStory[]> {
  return sealedOf(
    await getJSON(`/start/sealed?path=${encodeURIComponent(folder)}`)
  )
}

/** How many words a story has. */
export function wordCount(story: string): number {
  const words = story.trim()
  return words === "" ? 0 : words.split(/\s+/).length
}

/** A story this short leaves the agent guessing, so the desk offers to shape it first. */
export const shortStoryWords = 30

/** A sealed story handed to the desk by Shape a story, so the run starts from it. */
export interface ShapeArrival {
  id: string
  title: string
  workspace: string
  story: string
  issue: string
  repo: string
}

/** Reads the router state of a visit that came from a sealed session; anything else is null. */
export function shapeArrival(state: unknown): ShapeArrival | null {
  if (!isRecord(state) || !isRecord(state.shape)) {
    return null
  }
  const shape = state.shape
  if (text(shape.id) === "" || text(shape.story) === "") {
    return null
  }
  return {
    id: text(shape.id),
    title: text(shape.title),
    workspace: text(shape.workspace) || ".",
    story: text(shape.story),
    issue: text(shape.issue),
    repo: text(shape.repo),
  }
}
