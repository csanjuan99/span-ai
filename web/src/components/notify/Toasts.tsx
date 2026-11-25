"use client"

import {
  BellIcon,
  CheckIcon,
  XCircleIcon,
  XIcon,
  type LucideIcon,
} from "lucide-react"

import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"
import type { Toast } from "@/hooks/use-notifications"

const looks: Record<Toast["kind"], { icon: LucideIcon; tone: string }> = {
  finished: { icon: CheckIcon, tone: "bg-pass-soft text-pass" },
  failed: { icon: XCircleIcon, tone: "bg-fail-soft text-destructive" },
  needs: { icon: BellIcon, tone: "bg-hold-soft text-hold" },
  blocked: { icon: BellIcon, tone: "bg-hold-soft text-hold" },
  stopped: { icon: XCircleIcon, tone: "bg-fail-soft text-destructive" },
}

interface ToastsProps {
  toasts: Toast[]
  /** Opens the run a toast is about. */
  onOpen: (runId: string) => void
  onDismiss: (id: number) => void
  /** Called with true when the pointer enters a toast and false when it leaves. */
  onHold: (id: number, held: boolean) => void
}

/** The toasts, bottom right. Mount it once, where the page can show them over everything. */
export function Toasts({ toasts, onOpen, onDismiss, onHold }: ToastsProps) {
  if (toasts.length === 0) {
    return null
  }
  return (
    <ul
      aria-live="polite"
      className="fixed right-4 bottom-4 z-50 flex w-[min(380px,calc(100vw-2rem))] flex-col gap-3 motion-safe:[&>li]:animate-in motion-safe:[&>li]:fade-in-0 motion-safe:[&>li]:slide-in-from-bottom-2"
    >
      {toasts.map((toast) => {
        const { icon: Icon, tone } = looks[toast.kind]
        return (
          <li
            key={toast.id}
            onMouseEnter={() => onHold(toast.id, true)}
            onMouseLeave={() => onHold(toast.id, false)}
            className="grid grid-cols-[32px_minmax(0,1fr)_auto] items-center gap-3 rounded-[14px] border border-border bg-card py-3 pr-3 pl-4 shadow-md sm:grid-cols-[32px_minmax(0,1fr)_auto_auto]"
          >
            <span
              aria-hidden
              className={cn(
                "grid size-8 place-items-center rounded-full",
                tone
              )}
            >
              <Icon className="size-4" />
            </span>
            <div className="min-w-0">
              <p className="truncate text-sm font-semibold">{toast.title}</p>
              <p className="truncate text-xs text-muted-foreground">
                {toast.detail}
              </p>
            </div>
            <Button
              variant="outline"
              className="order-last col-span-2 col-start-2 h-11 sm:order-none sm:col-span-1 sm:col-start-auto sm:h-8"
              onClick={() => {
                onOpen(toast.runId)
                onDismiss(toast.id)
              }}
            >
              {toast.action}
            </Button>
            <Button
              variant="ghost"
              aria-label="Close notification"
              className="size-11 sm:size-8"
              onClick={() => onDismiss(toast.id)}
            >
              <XIcon />
            </Button>
          </li>
        )
      })}
    </ul>
  )
}
