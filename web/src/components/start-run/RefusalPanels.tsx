"use client"

import {
  ArrowRightIcon,
  BotIcon,
  FolderIcon,
  KeyRoundIcon,
  LinkIcon,
  LockIcon,
  SettingsIcon,
  ShieldAlertIcon,
  type LucideIcon,
} from "lucide-react"
import { Link } from "react-router"

import { CommandLine } from "@/components/problem/CommandLine"
import { KeptNext } from "@/components/problem/KeptNext"
import { Panel, type Tone } from "@/components/shaping/session-ui"
import { Button, buttonVariants } from "@/components/ui/button"
import type { StartRefusal } from "@/components/start-run/types"
import { wideButton } from "@/lib/button-class"
import { cn } from "@/lib/utils"

interface Refusal {
  band: string
  tone: Tone
  icon: LucideIcon
  title: string
  kept: string
  next: string
  /** Where the main button goes; none means it hands the form back. */
  to?: string
  action: string
  actionIcon: LucideIcon
  command?: string
}

const refusals: Record<number, Refusal> = {
  6: {
    band: "Run already exists",
    tone: "hold",
    icon: LinkIcon,
    title: "A run for this story already exists",
    kept: "That run, its branch and its proof are untouched. Nothing new was started.",
    next: "Open it and resolve what holds it. To start over, rewind it from its page.",
    action: "Open the run",
    actionIcon: ArrowRightIcon,
  },
  8: {
    band: "Jira signed you out",
    tone: "hold",
    icon: KeyRoundIcon,
    title: "Jira is not connected",
    kept: "The story key and every other choice on this screen.",
    next: "Connect Jira under What, then start again.",
    action: "Back to the form",
    actionIcon: ArrowRightIcon,
  },
  14: {
    band: "Not a git repository",
    tone: "fail",
    icon: FolderIcon,
    title: "This folder is not a git repository",
    kept: "Nothing was created. Your files are untouched.",
    next: "Pick another folder, or turn this one into a repository.",
    action: "Choose another folder",
    actionIcon: FolderIcon,
    command: "git init",
  },
  15: {
    band: "Base branch missing",
    tone: "fail",
    icon: ShieldAlertIcon,
    title: "That base branch does not exist",
    kept: "No branch and no run were created.",
    next: "Pick a branch that exists, or fix the base in span.yaml.",
    action: "Back to the form",
    actionIcon: ArrowRightIcon,
  },
  16: {
    band: "Checkout busy",
    tone: "hold",
    icon: LockIcon,
    title: "This checkout is busy",
    kept: "The run that holds it keeps running. Your new run was not created.",
    next: "Wait for it to finish, or open it to see what it is doing.",
    action: "See the run",
    actionIcon: ArrowRightIcon,
  },
  17: {
    band: "No agent available",
    tone: "fail",
    icon: BotIcon,
    title: "No agent can write this run",
    kept: "Nothing was created. Your choices are still on the screen.",
    next: "Log in to one agent, then start again. Or wait for the reset.",
    to: "/settings",
    action: "Open Settings · Agents",
    actionIcon: SettingsIcon,
  },
  18: {
    band: "Token needed",
    tone: "fail",
    icon: ShieldAlertIcon,
    title: "This Span server wants an access token",
    kept: "Everything you filled in. No run was created.",
    next: "Add the token in Settings · Connection, then start again.",
    to: "/settings",
    action: "Open Settings · Connection",
    actionIcon: SettingsIcon,
  },
}

const generic: Refusal = {
  band: "Not started",
  tone: "fail",
  icon: ShieldAlertIcon,
  title: "Span refused to start",
  kept: "Nothing was created. Your choices are still on the screen.",
  next: "Fix what the cause says, then start again.",
  action: "Back to the form",
  actionIcon: ArrowRightIcon,
}

const bands: Record<Tone, string> = {
  acc: "bg-accent text-primary",
  pass: "bg-pass-soft text-pass",
  hold: "bg-hold-soft text-hold",
  fail: "bg-fail-soft text-fail",
  mut: "bg-muted text-muted-foreground",
}

/** N4: a coded refusal before any run exists: the cause, what is kept, and the one way on. */
export function RefusalPanels({
  refusal,
  onDismiss,
}: {
  refusal: StartRefusal
  onDismiss: () => void
}) {
  const view = refusals[refusal.number] ?? generic
  const Band = view.icon
  const Action = view.actionIcon
  const target =
    view.to ?? (refusal.runId !== undefined && refusal.runId !== ""
      ? `/spans/${refusal.runId}`
      : undefined)
  return (
    <Panel
      aria-label="Refusal"
      className="flex flex-col overflow-hidden"
      role="alert"
    >
      <div
        className={cn(
          "flex items-center gap-2 px-4 py-2 text-xs font-semibold",
          bands[view.tone]
        )}
      >
        <Band aria-hidden="true" className="size-4" />
        {view.band}
      </div>
      <div className="flex flex-col gap-3 p-4 md:p-5">
        <h2 className="text-lg leading-[26px] font-semibold">{view.title}</h2>
        <p className="text-sm text-pretty">
          <b>Cause.</b> {refusal.message}
        </p>
        {view.command !== undefined ? (
          <CommandLine command={view.command} />
        ) : null}
        <KeptNext kept={view.kept} next={view.next} />
        {target !== undefined ? (
          <Link
            to={target}
            className={cn(buttonVariants(), wideButton, "gap-2")}
          >
            <Action aria-hidden="true" />
            {refusal.runId !== undefined && refusal.runId !== ""
              ? `${view.action}: ${refusal.runId}`
              : view.action}
          </Link>
        ) : (
          <Button type="button" className={wideButton} onClick={onDismiss}>
            <Action aria-hidden="true" />
            {view.action}
          </Button>
        )}
      </div>
    </Panel>
  )
}
