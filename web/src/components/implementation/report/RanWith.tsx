import { ranWith } from "@/lib/ranwith"
import type { JournalEvent } from "@/hooks/use-events"

/**
 * The agent and model each stage really ran with. A stage with no chosen model says so,
 * so a default nobody picked is visible instead of silent.
 */
export function RanWithList({ events }: { events: JournalEvent[] }) {
  const rows = ranWith(events)
  if (rows.length === 0) {
    return null
  }
  return (
    <section aria-label="Ran with" className="flex flex-col gap-1 text-sm">
      <h2 className="text-balance">
        <span className="text-primary">$ </span>ran with
      </h2>
      <dl className="flex flex-col gap-1">
        {rows.map((row) => (
          <div key={row.stage} className="flex flex-wrap gap-x-3">
            <dt className="w-16 shrink-0 text-muted-foreground">{row.stage}</dt>
            <dd className="min-w-0 break-all">
              {row.agent}
              {row.model === "" ? (
                <span className="text-hold">
                  {" "}
                  · no model chosen, Gentle AI default
                </span>
              ) : (
                <span> · {row.model}</span>
              )}
            </dd>
          </div>
        ))}
      </dl>
    </section>
  )
}
