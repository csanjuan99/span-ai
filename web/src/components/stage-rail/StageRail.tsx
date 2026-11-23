import { cn } from "cn"

import { stages } from "@/lib/types"

/** The ten stages on one line. The current stage carries the prompt. */
export function StageRail({
  stage,
  verdict,
  sealed = [],
}: {
  stage: string
  verdict: string
  sealed?: string[]
}) {
  const done = new Set(sealed)
  return (
    <ol className="flex flex-wrap gap-x-3 gap-y-1 text-sm" aria-label="Stages">
      {stages.map((name) => {
        const current = name === stage
        const tone = current
          ? verdict === "stop"
            ? "text-destructive"
            : verdict === "hold"
              ? "text-hold"
              : "text-primary"
          : done.has(name)
            ? "text-pass"
            : "text-muted-foreground"
        const mark = current ? ">" : done.has(name) ? "+" : "·"
        return (
          <li key={name} aria-current={current ? "step" : undefined} className={cn("shrink-0 whitespace-nowrap tabular-nums", tone)}>
            <span aria-hidden="true">{mark} </span>
            {name}
          </li>
        )
      })}
    </ol>
  )
}
