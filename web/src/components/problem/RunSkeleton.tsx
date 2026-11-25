import { cn } from "@/lib/utils"

/** One block of the loading page: a rounded bar the width and height of what will replace it. */
function Bar({ className }: { className: string }) {
  return (
    <span
      className={cn(
        "block rounded-md bg-shimmer bg-[length:200%_100%] motion-safe:animate-[shim_1.4s_ease-in-out_infinite]",
        className
      )}
    />
  )
}

const tabs = ["Ledger", "Implementation", "QA"]

/** What the run page shows while the run is read: the same blocks, so nothing jumps when it arrives. */
export function RunSkeleton() {
  return (
    <div
      role="status"
      aria-busy="true"
      aria-label="Loading run"
      className="flex flex-col gap-6"
    >
      <div className="flex flex-col gap-2">
        <Bar className="h-5 w-24" />
        <Bar className="h-8 w-[70%]" />
        <Bar className="h-5 w-[48%]" />
      </div>
      <section className="rounded-[14px] border border-border bg-card p-4 shadow-sm sm:px-5">
        <div className="grid grid-cols-5 gap-2 sm:grid-cols-10">
          {Array.from({ length: 10 }, (_, index) => (
            <Bar key={index} className="h-8 rounded-full" />
          ))}
        </div>
      </section>
      <section
        aria-hidden="true"
        className="overflow-hidden rounded-[14px] border border-border bg-card shadow-sm"
      >
        <div className="bg-muted px-4 py-2.5 sm:px-5">
          <Bar className="h-4 w-40" />
        </div>
        <div className="flex flex-col gap-4 p-4 lg:grid lg:grid-cols-[minmax(0,1fr)_340px] lg:gap-0 lg:p-0">
          <div className="flex min-w-0 flex-col gap-4 lg:p-6">
            <Bar className="h-[26px] w-3/5" />
            <Bar className="h-[22px] w-full" />
            <Bar className="h-[22px] w-[85%]" />
            <Bar className="mt-2 h-[140px] w-full" />
          </div>
          <div className="flex min-w-0 flex-col gap-4 lg:border-l lg:border-border lg:p-6">
            <Bar className="h-[190px] w-full" />
            <Bar className="h-10 w-full" />
            <Bar className="h-16 w-full" />
          </div>
        </div>
      </section>
      <section
        aria-hidden="true"
        className="grid rounded-[14px] border border-border bg-card shadow-sm sm:grid-cols-3"
      >
        {Array.from({ length: 3 }, (_, index) => (
          <div
            key={index}
            className="flex min-w-0 flex-col gap-1 border-t border-border p-4 first:border-t-0 sm:border-t-0 sm:border-l sm:px-5 sm:first:border-l-0"
          >
            <Bar className="h-3 w-[60px]" />
            <Bar className="h-[26px] w-[140px] max-w-full" />
            <Bar className="h-3.5 w-[180px] max-w-full" />
          </div>
        ))}
      </section>
      <section
        aria-hidden="true"
        className="rounded-[14px] border border-border bg-card shadow-sm"
      >
        <div className="px-4 pt-3">
          <div className="flex gap-1 rounded-[14px] bg-muted p-1">
            {tabs.map((name, index) => (
              <span
                key={name}
                className={cn(
                  "flex h-9 flex-1 items-center justify-center rounded-[10px] text-[13px] font-medium text-muted-foreground",
                  index === 0 && "bg-card font-semibold text-primary shadow-sm"
                )}
              >
                {name}
              </span>
            ))}
          </div>
        </div>
        <div className="px-4 pt-2 pb-1 sm:px-5">
          {Array.from({ length: 3 }, (_, index) => (
            <div
              key={index}
              className="grid grid-cols-[24px_minmax(0,1fr)] items-start gap-x-3 gap-y-1 border-t border-border py-3 first:border-t-0 sm:grid-cols-[24px_96px_minmax(0,1fr)_auto]"
            >
              <Bar className="size-[22px] rounded-full" />
              <Bar className="h-4 w-[70px]" />
              <Bar className="col-start-2 h-4 w-full sm:col-start-auto" />
              <Bar className="col-start-2 h-3 w-10 sm:col-start-auto" />
            </div>
          ))}
        </div>
      </section>
    </div>
  )
}
