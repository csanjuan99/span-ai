import {
  FileTextIcon,
  KeyboardIcon,
  ListIcon,
  MessageSquareIcon,
  PauseIcon,
  PlayIcon,
  SettingsIcon,
  SparklesIcon,
  SquareTerminalIcon,
  SunMoonIcon,
  Undo2Icon,
  XCircleIcon,
  type LucideIcon,
} from "lucide-react"

const icons: [RegExp, LucideIcon][] = [
  [/resume/, PlayIcon],
  [/note/, MessageSquareIcon],
  [/pause/, PauseIcon],
  [/rewind/, Undo2Icon],
  [/stop/, XCircleIcon],
  [/transcript/, SquareTerminalIcon],
  [/diff/, FileTextIcon],
  [/go-desk/, ListIcon],
  [/go-settings/, SettingsIcon],
  [/shortcuts/, KeyboardIcon],
  [/theme/, SunMoonIcon],
  [/shaping|metrics/, SparklesIcon],
]

/** The icon of a palette command, found from its id. Commands without a known id get a plain dot. */
export function commandIcon(id: string): LucideIcon | undefined {
  return icons.find(([pattern]) => pattern.test(id))?.[1]
}

/** Whether the command ends the run, so the palette draws it as destructive. */
export function isDanger(id: string): boolean {
  return /stop/.test(id)
}
