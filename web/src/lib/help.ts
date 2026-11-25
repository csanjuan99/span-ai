/** What a palette command does when it runs. */
export type CommandAction =
  | { type: "go"; path: string }
  | { type: "theme"; value: "system" | "light" | "dark" }
  | { type: "sheet" }
  | { type: "run"; run: () => void }

/** One line of the command palette. */
export interface PaletteCommand {
  id: string
  group: string
  title: string
  hint: string
  action: CommandAction
  /** The keys that do the same thing, one per key cap. */
  keys?: string[]
}

/** The commands the palette lists: only things the app really does. */
export function paletteCommands(): PaletteCommand[] {
  return [
    {
      id: "go-desk",
      group: "Go to",
      title: "Desk",
      hint: "All runs, grouped by what needs you.",
      action: { type: "go", path: "/" },
      keys: ["G", "D"],
    },
    {
      id: "go-settings",
      group: "Go to",
      title: "Settings",
      hint: "Project, agent, checks, approvals.",
      action: { type: "go", path: "/settings" },
      keys: ["G", "S"],
    },
    {
      id: "go-metrics",
      group: "Go to",
      title: "Metrics",
      hint: "How the runs on this server went.",
      action: { type: "go", path: "/metrics" },
    },
    {
      id: "go-shaping",
      group: "Go to",
      title: "Shape a story",
      hint: "Turn an idea into a sealed story before the run.",
      action: { type: "go", path: "/shaping" },
    },
    {
      id: "theme-system",
      group: "Appearance",
      title: "Theme: System",
      hint: "Follow the theme of your system.",
      action: { type: "theme", value: "system" },
    },
    {
      id: "theme-light",
      group: "Appearance",
      title: "Theme: Light",
      hint: "Always use the light theme.",
      action: { type: "theme", value: "light" },
    },
    {
      id: "theme-dark",
      group: "Appearance",
      title: "Theme: Dark",
      hint: "Always use the dark theme.",
      action: { type: "theme", value: "dark" },
    },
    {
      id: "shortcuts",
      group: "Help",
      title: "Keyboard shortcuts",
      hint: "See every key.",
      action: { type: "sheet" },
      keys: ["?"],
    },
  ]
}

/** The commands whose title or hint contain the query, ignoring case. An empty query keeps them all. */
export function filterCommands(
  commands: PaletteCommand[],
  query: string
): PaletteCommand[] {
  const needle = query.trim().toLowerCase()
  if (needle === "") {
    return commands
  }
  return commands.filter((command) =>
    `${command.title} ${command.hint}`.toLowerCase().includes(needle)
  )
}

/** One row of the shortcut sheet. */
export interface Shortcut {
  label: string
  keys: string[]
}

/** The shortcuts the page answers, in the groups the sheet shows. */
export const shortcuts: { group: string; items: Shortcut[] }[] = [
  {
    group: "Everywhere",
    items: [
      { label: "Command palette", keys: ["⌘", "K"] },
      { label: "This sheet", keys: ["?"] },
      { label: "Go to Desk", keys: ["G", "D"] },
      { label: "Go to Settings", keys: ["G", "S"] },
      { label: "Close a panel", keys: ["Esc"] },
    ],
  },
]

/** Whether a key press happens while the person types, when a shortcut must stay out of the way. */
export function isTyping(target: EventTarget | null): boolean {
  if (!(target instanceof HTMLElement)) {
    return false
  }
  return (
    target.isContentEditable ||
    target.getAttribute("contenteditable") === "true" ||
    ["INPUT", "TEXTAREA", "SELECT"].includes(target.tagName)
  )
}

const chords: Record<string, string> = { d: "/", s: "/settings" }

/** The state of a two-key chord after `key`: what is pending, and where to go when it completed. */
export function nextChord(
  pending: string,
  key: string
): { pending: string; go: string } {
  if (pending === "g" && chords[key] !== undefined) {
    return { pending: "", go: chords[key] }
  }
  return pending === "" && key === "g"
    ? { pending: "g", go: "" }
    : { pending: "", go: "" }
}

/** What the top bar sends to open the command palette from a click. */
export const openPaletteEvent = "span:open-palette"

/** One action a run page offers the palette: its words, its key, and what it does. */
export interface RunCommand {
  id: string
  title: string
  hint: string
  /** The one key that does it while no field has focus, in lower case: "r", "n", "p"... */
  key: string
  /** True when the key needs Shift: Shift+S stops the run, S alone does nothing. */
  shift?: boolean
  run: () => void
}

/** What the page in front tells the palette about itself: whose run it is and what can be done to it. */
export interface RunContext {
  /** The run's name, for the group title: "This run · SPAN-482". */
  label: string
  commands: RunCommand[]
}

let context: RunContext | null = null
const listeners = new Set<() => void>()

/** Makes `next` the page's run context. Pass null to clear it. Returns a function that clears it if still current. */
export function setRunContext(next: RunContext | null): () => void {
  context = next
  listeners.forEach((listener) => listener())
  return () => {
    if (context === next) {
      context = null
      listeners.forEach((listener) => listener())
    }
  }
}

/** The run context in front, or null when the page is not a run. */
export function getRunContext(): RunContext | null {
  return context
}

/** Calls `listener` when the run context changes. Returns the way to stop. */
export function subscribeRunContext(listener: () => void): () => void {
  listeners.add(listener)
  return () => {
    listeners.delete(listener)
  }
}

/** The keys that run a command, as key caps: ["⇧", "S"] for Shift+S. */
export function runKeys(command: RunCommand): string[] {
  return [...(command.shift === true ? ["⇧"] : []), command.key.toUpperCase()]
}

/** The run command a key press asks for, if the context has one for it. */
export function runCommandFor(
  ctx: RunContext | null,
  key: string,
  shift: boolean
): RunCommand | undefined {
  return ctx?.commands.find(
    (command) =>
      command.key === key.toLowerCase() && (command.shift === true) === shift
  )
}

/** The palette's own lines for a run context, first in the list. */
export function runPaletteCommands(ctx: RunContext | null): PaletteCommand[] {
  if (ctx === null) {
    return []
  }
  return ctx.commands.map((command) => ({
    id: `run-${command.id}`,
    group: `This run · ${ctx.label}`,
    title: command.title,
    hint: command.hint,
    action: { type: "run", run: command.run },
    keys: runKeys(command),
  }))
}
