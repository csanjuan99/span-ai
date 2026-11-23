import { ChevronRightIcon } from "lucide-react"

/** The agent's raw output, closed by default. It is the source the report summarizes, kept for when it is not enough. */
export function Transcript({ lines }: { lines: string[] }) {
  return (
    <details className="group rounded-md border border-border">
      <summary className="flex cursor-pointer list-none items-center gap-2 px-3 py-2 text-sm hover:bg-muted/50 focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none">
        <ChevronRightIcon aria-hidden="true" className="size-3.5 text-muted-foreground transition-transform group-open:rotate-90" />
        <span>
          <span className="text-primary">$ </span>transcript
        </span>
        <span className="text-xs text-muted-foreground tabular-nums">{lines.length} lines</span>
      </summary>
      {lines.length === 0 ? (
        <p className="border-t border-border p-3 text-sm text-muted-foreground">The agent has not written yet.</p>
      ) : (
        <div className="max-h-96 overflow-auto border-t border-border py-2 font-mono text-xs leading-5">
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
