"use client"

import { XIcon } from "lucide-react"

import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"
import type { Toast } from "@/hooks/use-notifications"

const dots: Record<Toast["kind"], string> = {
  finished: "bg-pass",
  failed: "bg-destructive",
  needs: "bg-hold",
  blocked: "bg-hold",
  stopped: "bg-destructive",
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
      className="fixed right-4 bottom-4 z-50 flex w-80 flex-col gap-2"
    >
      {toasts.map((toast) => (
        <li
          key={toast.id}
          onMouseEnter={() => onHold(toast.id, true)}
          onMouseLeave={() => onHold(toast.id, false)}
          className="flex items-start gap-3 rounded-lg border border-border bg-card p-3 shadow-lg"
        >
          <span
            aria-hidden
            className={cn(
              "mt-1.5 size-2 shrink-0 rounded-full",
              dots[toast.kind]
            )}
          />
          <div className="min-w-0 flex-1">
            <p className="truncate text-sm font-medium">{toast.title}</p>
            <p className="truncate text-xs text-muted-foreground">
              {toast.detail}
            </p>
          </div>
          <Button
            size="xs"
            variant="outline"
            onClick={() => {
              onOpen(toast.runId)
              onDismiss(toast.id)
            }}
          >
            {toast.action}
          </Button>
          <Button
            size="icon-xs"
            variant="ghost"
            aria-label="Close notification"
            onClick={() => onDismiss(toast.id)}
          >
            <XIcon />
          </Button>
        </li>
      ))}
    </ul>
  )
}
