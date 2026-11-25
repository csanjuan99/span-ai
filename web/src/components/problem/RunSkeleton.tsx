import { cn } from "@/lib/utils"

function Bar({ className }: { className: string }) {
  return (
    <span
      className={cn(
        "block rounded-md bg-muted motion-safe:animate-pulse",
        className
      )}
    />
  )
}

/** What the run page shows while the run is read: the same blocks, so nothing jumps when it arrives. */
export function RunSkeleton() {
  return (
    <div
      role="status"
      aria-busy="true"
      aria-label="Loading run"
      className="flex flex-col gap-6"
    >
      <div className="flex flex-col gap-3">
        <Bar className="h-4 w-20" />
        <Bar className="h-6 w-2/3 max-w-md" />
        <div className="flex flex-wrap gap-2">
          {Array.from({ length: 10 }, (_, index) => (
            <Bar key={index} className="h-7 w-20" />
          ))}
        </div>
      </div>
      <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_20rem] lg:items-start">
        <div className="flex min-w-0 flex-col gap-6">
          <div className="flex flex-col gap-3 rounded-xl border border-border bg-card p-4">
            <Bar className="h-5 w-40" />
            <Bar className="h-4 w-full" />
            <Bar className="h-4 w-4/5" />
          </div>
          <div className="flex flex-col gap-3 rounded-xl border border-border bg-card p-4">
            <Bar className="h-5 w-32" />
            <Bar className="h-4 w-full" />
            <Bar className="h-4 w-3/5" />
          </div>
          <div className="flex flex-col gap-3">
            <Bar className="h-10 w-full rounded-2xl" />
            {Array.from({ length: 4 }, (_, index) => (
              <Bar key={index} className="h-6 w-full" />
            ))}
          </div>
        </div>
        <div className="flex flex-col gap-3 rounded-xl border border-border bg-card p-4">
          <Bar className="h-5 w-28" />
          <Bar className="h-4 w-full" />
          <Bar className="h-4 w-2/3" />
        </div>
      </div>
    </div>
  )
}
