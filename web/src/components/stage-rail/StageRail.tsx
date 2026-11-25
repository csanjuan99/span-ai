import { cn } from "cn"

import { stages } from "@/lib/types"

/**
 * The ten stages as pills. The mark stays (`+` sealed, `>` current, `·` waiting) so color
 * never carries the state alone, and a sealed stage shows how long it took when that is known.
 */
export function StageRail({
  stage,
  verdict,
  sealed = [],
  durations = {},
}: {
  stage: string
  verdict: string
  sealed?: string[]
  durations?: Record<string, string>
}) {
  const done = new Set(sealed)
  return (
    <ol className="flex flex-wrap gap-x-2 gap-y-2 text-xs" aria-label="Stages">
      {stages.map((name) => {
        // A finished run has no current stage: its last one is sealed like the rest.
        const current = name === stage && verdict !== "done"
        const tone = current
          ? verdict === "stop"
            ? "bg-fail-soft text-fail"
            : verdict === "hold"
              ? "bg-hold-soft text-hold"
              : "bg-accent-soft text-primary"
          : done.has(name)
            ? "bg-pass-soft text-pass"
            : "text-muted-foreground"
        const mark = current ? ">" : done.has(name) ? "+" : "·"
        const took = !current && done.has(name) ? durations[name] : undefined
        return (
          <li
            key={name}
            aria-current={current ? "step" : undefined}
            className={cn(
              "flex h-6 shrink-0 items-center gap-1.5 whitespace-nowrap rounded-full px-2.5 font-mono tabular-nums",
              current && "font-medium",
              tone,
            )}
          >
            <span aria-hidden="true">{mark}</span>
            {name}
            {took !== undefined ? <span className="text-[0.6875rem] opacity-80">{took}</span> : null}
          </li>
        )
      })}
    </ol>
  )
}
