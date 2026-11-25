"use client"

import { useEffect, useState } from "react"

import { agentChoice, getJSON, modelList } from "@/lib/api"
import type { AgentChoice, ModelOption } from "@/lib/types"

const none: AgentChoice = {
  agents: [],
  selected: "any",
  chosen: "",
  exists: false,
  reason: "",
  models: {},
}

/** What the interview agent picker needs: the CLIs, the project default and the models. */
export interface InterviewAgents {
  choice: AgentChoice
  /** The CLI that runs when the session names none. Empty when none is ready. */
  projectAgent: string
  /** The model the project sets for that CLI. Empty when it keeps the CLI's own. */
  projectModel: string
  /** The models the CLI `agent` can list; empty when it lists none. */
  models: ModelOption[]
}

/**
 * Reads the agent registry of a folder, and the models of `agent` ("" is the project
 * default). A folder or CLI that cannot be read leaves the lists empty: the picker then
 * only offers Auto, which the server resolves the same way.
 */
export function useInterviewAgents(
  workspace: string,
  agent: string
): InterviewAgents {
  const [choice, setChoice] = useState<AgentChoice>(none)
  const [lists, setLists] = useState<Record<string, ModelOption[]>>({})

  useEffect(() => {
    let live = true
    void getJSON(`/agent?path=${encodeURIComponent(workspace)}`)
      .then((body) => {
        if (live) {
          setChoice(agentChoice(body))
        }
      })
      .catch(() => {
        if (live) {
          setChoice(none)
        }
      })
    return () => {
      live = false
    }
  }, [workspace])

  const cli = agent === "" ? choice.chosen : agent
  useEffect(() => {
    if (cli === "") {
      return
    }
    let live = true
    void getJSON(`/agent/models?cli=${encodeURIComponent(cli)}`)
      .then((body) => {
        if (live) {
          setLists((prev) => ({ ...prev, [cli]: modelList(body).models }))
        }
      })
      .catch(() => {
        if (live) {
          setLists((prev) => ({ ...prev, [cli]: [] }))
        }
      })
    return () => {
      live = false
    }
  }, [cli])

  return {
    choice,
    projectAgent: choice.chosen,
    projectModel: choice.models[choice.chosen] ?? "",
    models: cli === "" ? [] : (lists[cli] ?? []),
  }
}
