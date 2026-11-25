import { purposeShares } from "@/lib/overview"
import type { Group } from "@/lib/report"
import { LineCounts } from "@/components/files-changed/LineCounts"

/** Where the changed lines went, as a bar per family: source, tests, docs, config, dependencies, build and deploy, generated. */
export function PurposeBars({ groups }: { groups: Group[] }) {
  const shares = purposeShares(groups)
  const lines = groups.reduce(
    (sum, group) => sum + group.added + group.removed,
    0
  )
  return (
    <section
      aria-label="Changes by purpose"
      className="rounded-[14px] border border-border bg-card px-4 py-4 shadow-sm sm:px-5"
    >
      <div className="mb-2 flex flex-wrap items-center gap-2">
        <h2 className="text-lg leading-[26px] font-semibold">
          Changes by purpose
        </h2>
        <span className="text-xs text-muted-foreground">
          Share of the {lines} changed lines
        </span>
      </div>
      {shares.map((share) => (
        <div
          key={share.kind}
          className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-x-3 gap-y-1.5 border-t border-border py-2 first:border-t-0 sm:grid-cols-[150px_70px_minmax(0,1fr)_120px]"
        >
          <span className="text-[13px] leading-5 font-medium">
            {share.label}
          </span>
          <span className="order-2 text-xs text-muted-foreground max-sm:hidden sm:order-none">
            {share.files} {share.files === 1 ? "file" : "files"}
          </span>
          <div
            role="img"
            aria-label={`${share.label}: ${share.percent}% of changed lines`}
            className="order-3 col-span-2 h-2.5 overflow-hidden rounded-full border border-border bg-muted sm:order-none sm:col-span-1"
          >
            <i
              className="block h-full rounded-full bg-primary"
              style={{ width: `${share.percent}%` }}
            />
          </div>
          {share.added + share.removed === 0 ? (
            <span className="text-right text-xs text-muted-foreground">
              none
            </span>
          ) : (
            <span className="text-right text-[13px] tabular-nums">
              <LineCounts added={share.added} removed={share.removed} />
            </span>
          )}
        </div>
      ))}
    </section>
  )
}
