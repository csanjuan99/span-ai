import { ChevronRightIcon } from "lucide-react"

/** A closed entry that opens onto numbered lines on the code surface. */
export function CodeView({ title, caption, lines, empty }: { title: string; caption: string; lines: string[]; empty: string }) {
  return (
    <details className="group overflow-hidden rounded-lg border border-border">
      <summary className="flex cursor-pointer list-none items-center gap-2 px-3 py-2 text-sm hover:bg-muted/50 focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none">
        <ChevronRightIcon aria-hidden="true" className="size-3.5 text-muted-foreground transition-transform group-open:rotate-90" />
        <span className="font-medium">{title}</span>
        <span className="text-xs text-muted-foreground tabular-nums">{caption}</span>
      </summary>
      {lines.length === 0 ? (
        <p className="border-t border-border p-3 text-sm text-muted-foreground">{empty}</p>
      ) : (
        <div className="max-h-96 overflow-auto border-t border-border bg-surface-code py-2 font-mono text-xs leading-5">
          {lines.map((line, index) => (
            <div key={index} className="flex min-w-max gap-3 px-3">
              <span className="w-8 shrink-0 text-right text-muted-foreground/70 tabular-nums select-none">{index + 1}</span>
              <span className="whitespace-pre">{line}</span>
            </div>
          ))}
        </div>
      )}
    </details>
  )
}
