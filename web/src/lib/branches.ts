import { branchChoice, getJSON } from "@/lib/api"
import type { BranchChoice } from "@/lib/types"

/** The branches of the repository in a folder, and the base Span would pick there. */
export async function readBranchChoice(path: string): Promise<BranchChoice> {
  return branchChoice(
    await getJSON(`/branches?path=${encodeURIComponent(path)}`)
  )
}
