import { CodeView } from "@/components/code-view/CodeView"
import { commandLog } from "@/lib/command-log"
import type { Statement } from "@/lib/types"

/** The raw record behind the report: what the agent wrote and what the run executed. Closed until asked for. */
export function RawOutput({
  transcript,
  statements,
}: {
  transcript: string[]
  statements: Statement[]
}) {
  const commands = commandLog(statements)
  return (
    <section aria-label="Raw output" className="flex flex-col">
      <h2 className="pt-2 text-xs font-medium tracking-[0.02em] text-muted-foreground">
        Raw output · collapsed
      </h2>
      <CodeView
        title="Agent transcript"
        caption={`${transcript.length} ${transcript.length === 1 ? "line" : "lines"} · opens in code view`}
        lines={transcript}
        empty="The agent has not written yet."
      />
      <CodeView
        title="Command log"
        caption={`${commands.length} ${commands.length === 1 ? "command" : "commands"} · opens in code view`}
        lines={commands}
        empty="No command has run yet."
      />
    </section>
  )
}
