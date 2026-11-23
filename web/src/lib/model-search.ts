import type { ModelOption } from "@/lib/types"

/**
 * The models whose id or label contain every word of query, in the order the CLI listed them.
 * An empty query keeps all of them: what is already chosen never narrows the list.
 */
export function searchModels(models: ModelOption[], query: string): ModelOption[] {
  const words = query
    .toLowerCase()
    .split(/\s+/)
    .filter((word) => word !== "")
  if (words.length === 0) {
    return models
  }
  return models.filter((model) => {
    const haystack = `${model.id} ${model.label}`.toLowerCase()
    return words.every((word) => haystack.includes(word))
  })
}
