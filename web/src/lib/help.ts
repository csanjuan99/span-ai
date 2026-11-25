/** What a palette command does when it runs. */
export type CommandAction =
  | { type: "go"; path: string }
  | { type: "theme"; value: "system" | "light" | "dark" }
  | { type: "sheet" }

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
