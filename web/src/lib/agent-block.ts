import type { AgentBlock } from "@/lib/types"

/** What a person has to do about an agent that cannot work, in one line each. */
export interface AgentBlockLine {
  title: string
  detail: string
  hint: string
  /** What the hold leaves safe. */
  kept: string
  /** The one thing to do, in words. */
  next: string
  /** A login a person can copy into a terminal, when that CLI has one. */
  command?: string
}

/** Where the run stands, so the line can say what is kept and which stage waits. */
export interface BlockContext {
  stage: string
  sealed: string[]
}

const logins: Record<string, string> = {
  codex: "codex login",
  "cursor-agent": "cursor-agent login",
}

function keptOf(sealed: string[]): string {
  if (sealed.length === 0) {
    return "Nothing is sealed yet, and nothing is lost."
  }
  const names =
    sealed.length === 1
      ? sealed[0]
      : `${sealed.slice(0, -1).join(", ")} and ${sealed[sealed.length - 1]}`
  return `${names} ${sealed.length === 1 ? "is" : "are"} sealed. Nothing is lost.`
}

/**
 * A blocked agent is a hold, not a failure of the change. The answer differs by
 * kind: a spent quota waits or switches CLI, missing credentials need a login.
 */
export function agentBlockLine(
  block: AgentBlock | null | undefined,
  context: BlockContext = { stage: "", sealed: [] }
): AgentBlockLine | null {
  if (!block || block.agent === "") {
    return null
  }
  const detail = (block.detail ?? "").trim()
  const kept = keptOf(context.sealed)
  const stage = context.stage === "" ? "this stage" : context.stage
  if (block.kind === "credentials") {
    return {
      title: `${block.agent} has no usable credentials`,
      detail:
        detail === "" ? "The CLI did not accept the key it found." : detail,
      hint: "Log in to that CLI, then resume. Nothing in this project can fix it.",
      kept,
      next: `Log in in a terminal, then resume. Or pick another agent for ${stage}.`,
      command: logins[block.agent],
    }
  }
  if (block.kind === "quota") {
    return {
      title: `${block.agent} is out of quota`,
      detail: detail === "" ? "The account reached its limit." : detail,
      hint: "Resume after the limit resets, or point agent: at another CLI.",
      kept,
      next: `Choose another agent for ${stage} and resume, or wait for the reset.`,
    }
  }
  return {
    title: `${block.agent} cannot run`,
    detail,
    hint: "Resume when that CLI works again.",
    kept,
    next: `Resume ${stage} when that CLI works again.`,
  }
}
