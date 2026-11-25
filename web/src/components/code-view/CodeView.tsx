import { TerminalIcon } from "lucide-react"

import { DisclosureChevron } from "@/components/code-view/DisclosureChevron"

/** A closed entry that opens onto numbered lines on the code surface. */
export function CodeView({
  title,
  caption,
  lines,
  empty,
}: {
  title: string
  caption: string
  lines: string[]
  empty: string
}) {
  return (
    <details className="group border-t border-border">
      <summary className="flex min-h-11 cursor-pointer list-none items-center gap-2 py-2 text-[13px] hover:bg-muted/50 focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none">
        <DisclosureChevron />
        <span>{title}</span>
        <span className="text-xs text-muted-foreground tabular-nums">
          {caption}
        </span>
        <TerminalIcon
          aria-hidden="true"
          className="ml-auto size-4 text-muted-foreground"
        />
      </summary>
      {lines.length === 0 ? (
        <p className="border-t border-border py-3 text-sm text-muted-foreground">
          {empty}
        </p>
      ) : (
        <div className="max-h-96 overflow-auto border-t border-border bg-surface-code py-2 font-mono text-xs leading-5">
          {lines.map((line, index) => (
            <div key={index} className="flex min-w-max gap-3 px-3">
              <span className="w-8 shrink-0 text-right text-muted-foreground/70 tabular-nums select-none">
                {index + 1}
              </span>
              <span className="whitespace-pre">{line}</span>
            </div>
          ))}
        </div>
      )}
    </details>
  )
}
