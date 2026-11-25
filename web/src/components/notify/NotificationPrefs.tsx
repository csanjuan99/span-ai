"use client"

import { useState } from "react"

import { Checkbox } from "@/components/ui/checkbox"
import {
  kindCopy,
  loadPrefs,
  notifyKinds,
  savePrefs,
  type Channels,
  type NotifyKind,
  type NotifyPrefs,
} from "@/lib/notify"

type Channel = keyof Channels

const columns: { channel: Channel; label: string }[] = [
  { channel: "toast", label: "Toast" },
  { channel: "tab", label: "Tab and badge" },
  { channel: "system", label: "System" },
]

/** What the browser says about its own notifications: `none` when it has no such thing. */
function permissionNow(): NotificationPermission | "none" {
  return typeof Notification === "undefined" ? "none" : Notification.permission
}

const permissionCopy: Record<"none" | "denied" | "default" | "granted", { title: string; hint: string }> = {
  none: { title: "This browser has no system notifications", hint: "Toasts and the tab badge still work." },
  denied: {
    title: "System notifications are blocked",
    hint: "Allow them for this site in the browser's settings. Toasts and the tab badge work without it.",
  },
  default: { title: "System notifications are off", hint: "The browser asks once, when you turn one on." },
  granted: { title: "System notifications are allowed", hint: "Turn them on for the events you want." },
}

/** The notifications section of Settings: which events reach a person, and how. */
export function NotificationPrefs() {
  const [prefs, setPrefs] = useState(loadPrefs)
  const [permission, setPermission] = useState(permissionNow)

  function keep(next: NotifyPrefs) {
    setPrefs(next)
    savePrefs(next)
  }

  function toggle(kind: NotifyKind, channel: Channel, on: boolean) {
    keep({ ...prefs, events: { ...prefs.events, [kind]: { ...prefs.events[kind], [channel]: on } } })
    if (channel === "system" && on && permission === "default") {
      void Notification.requestPermission().then(setPermission)
    }
  }

  const copy = permissionCopy[permission]
  const systemOff = permission === "none" || permission === "denied"

  return (
    <section className="flex flex-col gap-4">
      <div>
        <h2 className="text-base font-semibold">Notifications</h2>
        <p className="text-sm text-muted-foreground">Choose what gets your attention and how.</p>
      </div>
      <div className="rounded-lg border border-border bg-card p-3">
        <p className="text-sm font-medium">{copy.title}</p>
        <p className="text-xs text-muted-foreground">{copy.hint}</p>
      </div>
      <table className="w-full text-sm">
        <thead>
          <tr className="text-left text-xs text-muted-foreground">
            <th className="py-2 font-medium">Event</th>
            {columns.map((column) => (
              <th key={column.channel} className="py-2 font-medium">
                {column.label}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {notifyKinds.map((kind) => (
            <tr key={kind} className="border-t border-border">
              <td className="py-2">
                <div className="font-medium">{kindCopy[kind].label}</div>
                <div className="text-xs text-muted-foreground">{kindCopy[kind].hint}</div>
              </td>
              {columns.map((column) => (
                <td key={column.channel} className="py-2">
                  <Checkbox
                    aria-label={`${kindCopy[kind].label}: ${column.label.toLowerCase()}`}
                    checked={prefs.events[kind][column.channel]}
                    disabled={column.channel === "system" && systemOff}
                    onCheckedChange={(on) => toggle(kind, column.channel, on === true)}
                  />
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
      <div className="flex items-center justify-between gap-4 border-t border-border pt-3">
        <div>
          <div className="text-sm font-medium">Only when Span is in the background</div>
          <div className="text-xs text-muted-foreground">Skip toasts while this tab is in front. Default: on.</div>
        </div>
        <Checkbox
          aria-label="Only when Span is in the background"
          checked={prefs.background}
          onCheckedChange={(on) => keep({ ...prefs, background: on === true })}
        />
      </div>
    </section>
  )
}
