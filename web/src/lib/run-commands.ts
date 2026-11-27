import type { RunCommand, RunContext } from "@/lib/help"

/** The actions a run page can take now; an action it cannot take is left out. */
export interface RunActions {
  resume?: () => void
  note?: () => void
  pause?: () => void
  stop?: () => void
  rewind?: () => void
  transcript?: () => void
  diff?: () => void
  copySummary?: () => void
  ledger?: () => void
  implementation?: () => void
  qa?: () => void
}

const table: { name: keyof RunActions; command: Omit<RunCommand, "run"> }[] = [
  { name: "resume", command: { id: "resume", title: "Resume the run", hint: "Continue from where it stopped", key: "r" } },
  { name: "note", command: { id: "note", title: "Leave a note", hint: "A stage reads it when it starts", key: "n" } },
  { name: "pause", command: { id: "pause", title: "Pause", hint: "Stops the agent; a command stage finishes first", key: "p" } },
  { name: "stop", command: { id: "stop", title: "Stop the run", hint: "Sealed stages and the branch stay", key: "s", shift: true } },
  { name: "rewind", command: { id: "rewind", title: "Restart from a stage…", hint: "The attempt so far stays on disk", key: "w" } },
  { name: "transcript", command: { id: "transcript", title: "Open the agent transcript", hint: "Raw output of the run", key: "t" } },
  { name: "diff", command: { id: "diff", title: "Open the diff", hint: "Implementation tab", key: "d" } },
  { name: "copySummary", command: { id: "copy-summary", title: "Copy the summary", hint: "Markdown, to paste where you share it", key: "c" } },
  { name: "ledger", command: { id: "tab-ledger", title: "Show the ledger", hint: "Stage by stage, what happened", key: "1" } },
  { name: "implementation", command: { id: "tab-implementation", title: "Show the implementation", hint: "Diff, commits, reasoning and audit", key: "2" } },
  { name: "qa", command: { id: "tab-qa", title: "Show QA", hint: "The cases and how they ran", key: "3" } },
]

/** The palette context of a run: its name and one command per action it can take, or null when it has none. */
export function runContextOf(label: string, actions: RunActions): RunContext | null {
  const commands = table.flatMap(({ name, command }) => {
    const run = actions[name]
    return run === undefined ? [] : [{ ...command, run }]
  })
  return commands.length === 0 ? null : { label, commands }
}
