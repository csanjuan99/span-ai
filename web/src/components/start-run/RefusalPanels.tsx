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

import { ProblemCard } from "@/components/problem/ProblemCard"
import { Button, buttonVariants } from "@/components/ui/button"
import type { StartRefusal } from "@/components/start-run/types"
import { cn } from "@/lib/utils"

interface Refusal {
  band: string
  tone: "hold" | "fail"
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
    tone: "hold",
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
    tone: "hold",
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

/** N4: a coded refusal before any run exists: the cause, what is kept, and the one way on. */
export function RefusalPanels({
  refusal,
  onDismiss,
}: {
  refusal: StartRefusal
  onDismiss: () => void
}) {
  const view = refusals[refusal.number] ?? generic
  const Action = view.actionIcon
  const title =
    refusal.number === 15 && refusal.base !== undefined && refusal.base !== ""
      ? `Branch ${refusal.base} does not exist`
      : view.title
  const runId = refusal.runId ?? ""
  const target = view.to ?? (runId !== "" ? `/spans/${runId}` : undefined)
  return (
    <div role="alert">
      <ProblemCard
        label={view.band}
        tone={view.tone}
        icon={view.icon}
        title={title}
        cause={refusal.message}
        command={view.command}
        kept={view.kept}
        next={view.next}
      >
        {target !== undefined ? (
          <Link to={target} className={cn(buttonVariants(), "gap-2")}>
            <Action aria-hidden="true" />
            {runId !== "" ? `${view.action}: ${runId}` : view.action}
          </Link>
        ) : (
          <Button type="button" onClick={onDismiss}>
            <Action aria-hidden="true" />
            {view.action}
          </Button>
        )}
      </ProblemCard>
    </div>
  )
}
