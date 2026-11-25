"use client"

import {
  BellIcon,
  BotIcon,
  FolderIcon,
  LinkIcon,
  PauseIcon,
  ShieldCheckIcon,
  SparklesIcon,
  UploadIcon,
  type LucideIcon,
} from "lucide-react"

import { cn } from "@/lib/utils"

/** The sections of the settings page, in the order the design lists them. */
export const settingsSections: { name: string; icon: LucideIcon }[] = [
  { name: "Connection", icon: LinkIcon },
  { name: "Project", icon: FolderIcon },
  { name: "Agents", icon: BotIcon },
  { name: "Checks", icon: ShieldCheckIcon },
  { name: "Approvals", icon: PauseIcon },
  { name: "Jira", icon: UploadIcon },
  { name: "Appearance", icon: SparklesIcon },
  { name: "Notifications", icon: BellIcon },
]

/** The list of sections beside the page; on a narrow screen it wraps into touch-sized chips. */
export function SettingsNav({
  current,
  onPick,
}: {
  current: string
  onPick: (name: string) => void
}) {
  return (
    <nav
      aria-label="Settings sections"
      className="flex flex-wrap gap-2 lg:flex-col lg:gap-0.5"
    >
      {settingsSections.map(({ name, icon: Icon }) => (
        <a
          key={name}
          href={`#${name.toLowerCase()}`}
          aria-current={current === name ? "page" : undefined}
          onClick={() => onPick(name)}
          className={cn(
            "flex min-h-11 items-center gap-2 rounded-[10px] border border-input px-3.5 py-2 text-[13px] leading-5 text-muted-foreground hover:text-foreground lg:min-h-0 lg:border-transparent lg:px-3",
            current === name &&
              "border-transparent bg-accent font-semibold text-primary hover:text-primary"
          )}
        >
          <Icon className="size-4" aria-hidden="true" />
          {name}
        </a>
      ))}
    </nav>
  )
}
