"use client"

import type { ShapeSplit } from "@/lib/types"
import { PromptHeading } from "@/components/shell/PromptHeading"

/** The story gives more than one outcome. There is nothing to seal until it is narrowed. */
export function Split({ split }: { split: ShapeSplit[] }) {
  return (
    <section className="flex flex-col gap-2">
      <PromptHeading>split</PromptHeading>
      <p className="text-pretty text-hold">
        This idea holds more than one outcome. Narrow the story to one of these,
        then start a new session with it.
      </p>
      <ul className="flex flex-col gap-2">
        {split.map((item) => (
          <li key={item.story} className="border border-border p-3 text-sm">
            <p>{item.story}</p>
            <p className="text-muted-foreground">{`Left out: ${item.leftOut}`}</p>
          </li>
        ))}
      </ul>
    </section>
  )
}
