"use client"

import { useState } from "react"

import { Input } from "@/components/ui/input"
import { searchModels } from "@/lib/model-search"
import type { ModelOption } from "@/lib/types"
import { cn } from "@/lib/utils"

/**
 * Every model a CLI listed, with a search. The native suggestion list of a text field only
 * shows what matches the text already typed, so a model that is chosen hides all the others;
 * this one always opens on the whole list.
 */
export function ModelBrowser({
  id,
  label,
  models,
  selected,
  onPick,
}: {
  id: string
  label: string
  models: ModelOption[]
  selected: string
  onPick: (model: string) => void
}) {
  const [query, setQuery] = useState("")
  const shown = searchModels(models, query)
  if (models.length === 0) {
    return null
  }
  return (
    <details className="group w-full max-w-xl">
      <summary className="cursor-pointer text-sm max-md:flex max-md:min-h-11 max-md:items-center text-primary underline-offset-4 hover:underline focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none">
        Browse {models.length} {models.length === 1 ? "model" : "models"}
      </summary>
      <div className="mt-2 flex flex-col gap-2 border border-border p-2">
        <Input
          aria-label={`Search ${label} models`}
          placeholder="Search by name"
          autoComplete="off"
          spellCheck={false}
          value={query}
          onChange={(event) => setQuery(event.target.value)}
        />
        <p className="text-xs text-muted-foreground tabular-nums" aria-live="polite">
          {shown.length === models.length ? `${models.length} models` : `${shown.length} of ${models.length} models`}
        </p>
        {shown.length === 0 ? (
          <p className="text-pretty text-sm text-muted-foreground">No model matches “{query}”.</p>
        ) : (
          <ul aria-label={`${label} models`} id={`browse-${id}`} className="flex max-h-64 flex-col overflow-y-auto">
            {shown.map((model) => (
              <li key={model.id}>
                <button
                  type="button"
                  aria-pressed={model.id === selected}
                  className={cn(
                    "flex w-full flex-wrap items-baseline gap-x-3 px-2 py-1 text-left max-md:min-h-11 max-md:items-center text-sm hover:bg-muted focus-visible:bg-muted focus-visible:outline-none",
                    model.id === selected ? "text-primary" : "",
                  )}
                  onClick={() => onPick(model.id)}
                >
                  <span className="break-all">{model.id}</span>
                  {model.label !== "" && model.label !== model.id ? <span className="text-muted-foreground">{model.label}</span> : null}
                </button>
              </li>
            ))}
          </ul>
        )}
      </div>
    </details>
  )
}
