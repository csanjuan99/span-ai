import { cn } from "cn"

import { tidy } from "@/lib/tidy"

/** The most a reason shows before it folds. A failing hook can print a whole review. */
const limit = 280

/** A line the engine wrote, cleaned of terminal colors, and folded when it runs long. */
export function ClampedText({
  text,
  className,
}: {
  text: string
  className?: string
}) {
  const clean = tidy(text)
  if (clean.length <= limit) {
    return <p className={className}>{clean}</p>
  }
  return (
    <details className={cn("group", className)}>
      <summary className="cursor-pointer list-none text-pretty break-words focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none">
        {clean.slice(0, limit).trimEnd()}…{" "}
        <span className="text-xs text-primary underline-offset-4 group-open:hidden">
          Show all
        </span>
      </summary>
      <p className="mt-2 break-words whitespace-pre-wrap">{clean}</p>
    </details>
  )
}
