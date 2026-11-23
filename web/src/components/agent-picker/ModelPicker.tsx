import { useEffect, useState } from "react"

import { getJSON, modelList, saveModel, agentChoice } from "@/lib/api"
import type { AgentChoice, ModelList } from "@/lib/types"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"

interface Props {
  project: string
  choice: AgentChoice
  /** The agent view the server answered after a save. */
  onSaved: (next: AgentChoice) => void
}

/**
 * One model per CLI. The input suggests what the CLI can list and accepts any valid name,
 * so a model the list does not show can still be typed. Empty keeps what Gentle AI set up.
 */
export function ModelPicker({ project, choice, onSaved }: Props) {
  const [lists, setLists] = useState<Record<string, ModelList>>({})
  const [drafts, setDrafts] = useState<Record<string, string>>({})
  const [notes, setNotes] = useState<Record<string, string>>({})
  const [busy, setBusy] = useState("")

  const readyIds = choice.agents
    .filter((row) => row.ready)
    .map((row) => row.id)
    .join(",")

  useEffect(() => {
    let live = true
    for (const id of readyIds.split(",").filter((item) => item !== "")) {
      void getJSON(`/agent/models?cli=${encodeURIComponent(id)}`)
        .then((body) => {
          if (live) {
            setLists((prev) => ({ ...prev, [id]: modelList(body) }))
          }
        })
        .catch((err: unknown) => {
          if (live) {
            setNotes((prev) => ({ ...prev, [id]: err instanceof Error ? err.message : "the models could not be read" }))
          }
        })
    }
    return () => {
      live = false
    }
  }, [readyIds])

  async function save(id: string, model: string) {
    setBusy(id)
    setNotes((prev) => ({ ...prev, [id]: "" }))
    try {
      const body = await saveModel(project, id, model.trim())
      onSaved(agentChoice(body))
      setDrafts((prev) => Object.fromEntries(Object.entries(prev).filter(([key]) => key !== id)))
      setNotes((prev) => ({ ...prev, [id]: model.trim() === "" ? "Back to the Gentle AI model." : "Saved. The next run uses this model." }))
    } catch (err: unknown) {
      setNotes((prev) => ({ ...prev, [id]: err instanceof Error ? err.message : "the model was not saved" }))
    } finally {
      setBusy("")
    }
  }

  return (
    <div className="flex flex-col gap-3 border-t border-border pt-3">
      <p className="text-pretty text-sm text-muted-foreground">
        <strong className="font-medium text-foreground">Gentle AI and its configuration are required.</strong> Each CLI keeps
        the models Gentle AI set up for it. Choose one here only to change the model that orchestrates the work; leave it
        empty to keep Gentle AI&apos;s choice.
      </p>
      {choice.agents.map((row) => {
        const saved = choice.models[row.id] ?? ""
        const value = drafts[row.id] ?? saved
        const list = lists[row.id]
        const listId = `models-${row.id}`
        return (
          <div key={row.id} className="flex flex-col gap-1">
            <Label htmlFor={`model-${row.id}`}>{row.label} model</Label>
            <div className="flex flex-wrap items-center gap-2">
              <Input
                id={`model-${row.id}`}
                className="w-72"
                list={listId}
                value={value}
                disabled={!row.ready}
                placeholder="Gentle AI default"
                autoComplete="off"
                spellCheck={false}
                onChange={(event) => setDrafts((prev) => ({ ...prev, [row.id]: event.target.value }))}
              />
              <datalist id={listId}>
                {(list?.models ?? []).map((option) => (
                  <option key={option.id} value={option.id}>
                    {option.label}
                  </option>
                ))}
              </datalist>
              <Button
                type="button"
                variant="secondary"
                disabled={!row.ready || busy !== "" || value === saved}
                onClick={() => void save(row.id, value)}
              >
                Save model
              </Button>
            </div>
            {!row.ready ? (
              <p className="text-pretty text-sm text-hold">{row.reason ?? "not ready"}. Install Gentle AI for this CLI first.</p>
            ) : null}
            {list !== undefined && list.error !== "" ? (
              <p className="text-pretty text-sm text-muted-foreground">
                The list is not available ({list.error}). You can still type a model name.
              </p>
            ) : null}
            {list !== undefined && list.source === "none" && list.error === "" ? (
              <p className="text-pretty text-sm text-muted-foreground">This CLI cannot list its models. Type the name.</p>
            ) : null}
            {(notes[row.id] ?? "") !== "" ? <p className="text-pretty text-sm">{notes[row.id]}</p> : null}
          </div>
        )
      })}
    </div>
  )
}
