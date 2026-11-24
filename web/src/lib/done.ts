import { changeLabel, shortHash } from "@/lib/ledger"
import { stages } from "@/lib/types"

/** What the page says once a run is over: what finished, where it went, how to open it. */
export interface Done {
  title: string
  landed: string
  branch: string
  issue: string
  change: { label: string; href: string } | null
}

/**
 * Reads the end of a run from the cursor and the ledger. A merge is only claimed when the
 * ledger sealed the merge stage, so a run that ended some other way is not shown as shipped.
 * Whether the story moved in Jira is not recorded, so it is not stated.
 */
export function doneOf(input: {
  sealed: number
  merged: boolean
  sha: string
  base: string
  branch: string
  issue: string
  changeURL: string
}): Done {
  const sha = input.sha === "" ? "" : shortHash(input.sha)
  const landed = input.merged
    ? `Merged into ${input.base}${sha === "" ? "" : ` as ${sha}`}`
    : sha === ""
      ? "Finished"
      : `Finished at ${sha}${input.branch === "" ? "" : ` on ${input.branch}`}`
  return {
    title:
      input.sealed === stages.length
        ? `all ${input.sealed} stages sealed`
        : `${input.sealed} ${input.sealed === 1 ? "stage" : "stages"} sealed`,
    landed,
    branch: input.branch,
    issue: input.issue,
    change: input.changeURL === "" ? null : { label: `Open ${changeLabel(input.changeURL)}`, href: input.changeURL },
  }
}
