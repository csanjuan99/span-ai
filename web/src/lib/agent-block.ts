import type { AgentBlock } from "@/lib/types"

/** What a person has to do about an agent that cannot work, in one line each. */
export interface AgentBlockLine {
  title: string
  detail: string
  hint: string
}

/**
 * A blocked agent is a hold, not a failure of the change. The answer differs by
 * kind: a spent quota waits or switches CLI, missing credentials need a login.
 */
export function agentBlockLine(block: AgentBlock | null | undefined): AgentBlockLine | null {
  if (!block || block.agent === "") {
    return null
  }
  const detail = (block.detail ?? "").trim()
  if (block.kind === "credentials") {
    return {
      title: `${block.agent} has no usable credentials`,
      detail: detail === "" ? "The CLI did not accept the key it found." : detail,
      hint: "Log in to that CLI, then resume. Nothing in this project can fix it.",
    }
  }
  if (block.kind === "quota") {
    return {
      title: `${block.agent} is out of quota`,
      detail: detail === "" ? "The account reached its limit." : detail,
      hint: "Resume after the limit resets, or point agent: at another CLI.",
    }
  }
  return { title: `${block.agent} cannot run`, detail, hint: "Resume when that CLI works again." }
}
