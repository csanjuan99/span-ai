import { KeptNext } from "@/components/problem/KeptNext"
import { CommandLine } from "@/components/problem/CommandLine"
import type { AgentBlockLine } from "@/lib/agent-block"

/** An agent that cannot work is a hold, not a failure of the change: what happened, what is kept, what to do. */
export function AgentBlocked({ line }: { line: AgentBlockLine }) {
  return (
    <section className="flex flex-col gap-3" aria-label="Agent blocked">
      <h2 className="text-base font-medium">{line.title}</h2>
      <p className="text-sm text-pretty">{line.detail}</p>
      <p className="text-sm text-pretty text-muted-foreground">{line.hint}</p>
      {line.command !== undefined ? (
        <CommandLine command={line.command} />
      ) : null}
      <KeptNext kept={line.kept} next={line.next} />
    </section>
  )
}
