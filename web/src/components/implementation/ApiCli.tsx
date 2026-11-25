"use client"

import { DisclosureChevron } from "@/components/code-view/DisclosureChevron"
import { CommandLine } from "@/components/problem/CommandLine"

/** One action of a board with the route that serves it and the command that does the same from a shell. */
export interface ApiAction {
  name: string
  method: "GET" | "POST"
  route: string
  command: string
}

/**
 * The route and the command behind each action of a board, closed by default. The page only
 * calls the API, so everything it shows can be read or done from a terminal.
 */
export function ApiCli({
  actions,
  exits = "0 ok · 1 failed · 2 usage · 3 assumed decisions unchecked · 4 not found · 12 audit chain broken",
}: {
  actions: ApiAction[]
  /** The exit codes of the commands, as the footer says them. */
  exits?: string
}) {
  return (
    <details className="group rounded-[14px] border border-border bg-card shadow-sm">
      <summary className="flex cursor-pointer list-none items-center gap-2 px-4 py-3 text-[13px] font-medium focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none">
        <DisclosureChevron />
        API / CLI
        <span className="font-normal text-muted-foreground">
          Everything here also works from a terminal
        </span>
      </summary>
      <div className="flex flex-col gap-4 border-t border-border p-4">
        {actions.map((action) => (
          <section
            key={action.name}
            aria-label={action.name}
            className="flex flex-col gap-1.5"
          >
            <h3 className="text-[13px] font-semibold">{action.name}</h3>
            <p className="font-mono text-xs break-all">
              <span className="text-muted-foreground">HTTP </span>
              <span className="font-semibold">{action.method}</span>{" "}
              {action.route}
            </p>
            <CommandLine command={action.command} />
          </section>
        ))}
        <p className="text-xs text-muted-foreground">
          Exit codes: {exits}. Add --json for the route's body.
        </p>
      </div>
    </details>
  )
}
