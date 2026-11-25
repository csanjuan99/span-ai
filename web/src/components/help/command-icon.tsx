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
import type { ReactElement } from "react"

type Draw = (className: string) => ReactElement

/** Binds an icon at module load, so no component is ever picked while a page renders. */
function draw(Icon: LucideIcon): Draw {
  return (className) => <Icon aria-hidden="true" className={className} />
}

const icons: [RegExp, Draw][] = [
  [/resume/, draw(PlayIcon)],
  [/note/, draw(MessageSquareIcon)],
  [/pause/, draw(PauseIcon)],
  [/rewind/, draw(Undo2Icon)],
  [/stop/, draw(XCircleIcon)],
  [/transcript/, draw(SquareTerminalIcon)],
  [/diff/, draw(FileTextIcon)],
  [/go-desk/, draw(ListIcon)],
  [/go-settings/, draw(SettingsIcon)],
  [/shortcuts/, draw(KeyboardIcon)],
  [/theme/, draw(SunMoonIcon)],
  [/shaping|metrics/, draw(SparklesIcon)],
]

/** The icon of a palette command, found from its id, or null for a command without one. */
export function commandIcon(
  id: string,
  className: string
): ReactElement | null {
  return icons.find(([pattern]) => pattern.test(id))?.[1](className) ?? null
}

/** Whether the command ends the run, so the palette draws it as destructive. */
export function isDanger(id: string): boolean {
  return /stop/.test(id)
}
