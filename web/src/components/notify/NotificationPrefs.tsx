"use client"

import { useState } from "react"

import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import {
  kindCopy,
  loadPrefs,
  notifyKinds,
  playTone,
  savePrefs,
  type Channels,
  type NotifyKind,
  type NotifyPrefs,
} from "@/lib/notify"
import { cn } from "@/lib/utils"

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

const permissionCopy: Record<
  "none" | "denied" | "default" | "granted",
  { title: string; hint: string }
> = {
  none: {
    title: "This browser has no system notifications",
    hint: "Toasts and the tab badge still work.",
  },
  denied: {
    title: "System notifications are blocked",
    hint: "Allow them for this site in the browser's settings. Toasts and the tab badge work without it.",
  },
  default: {
    title: "System notifications are off",
    hint: "The browser asks once, when you turn one on.",
  },
  granted: {
    title: "System notifications are allowed",
    hint: "Turn them on for the events you want.",
  },
}

/** One yes-or-no setting: what it does, and a switch that says On or Off in words too. */
function Setting({
  name,
  hint,
  label,
  on,
  onChange,
}: {
  name: string
  hint: string
  label: string
  on: boolean
  onChange: (on: boolean) => void
}) {
  return (
    <div className="flex items-center justify-between gap-4 border-t border-border pt-3">
      <div>
        <div className="text-[13px] font-medium">{name}</div>
        <div className="text-xs text-muted-foreground">{hint}</div>
      </div>
      <span className="flex items-center gap-2">
        <span className="text-xs font-semibold">{on ? "On" : "Off"}</span>
        <button
          type="button"
          role="switch"
          aria-checked={on}
          aria-label={label}
          onClick={() => onChange(!on)}
          className={cn(
            "relative h-6 w-10 shrink-0 rounded-full border border-input bg-muted transition-colors",
            "after:absolute after:top-0.5 after:left-0.5 after:size-4.5 after:rounded-full after:bg-muted-foreground after:transition-transform",
            on &&
              "border-primary bg-accent after:translate-x-4 after:bg-primary"
          )}
        />
      </span>
    </div>
  )
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
    keep({
      ...prefs,
      events: {
        ...prefs.events,
        [kind]: { ...prefs.events[kind], [channel]: on },
      },
    })
    if (channel === "system" && on && permission === "default") {
      void Notification.requestPermission().then(setPermission)
    }
  }

  const copy = permissionCopy[permission]
  const systemOff = permission === "none" || permission === "denied"

  return (
    <section className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center gap-3 rounded-[14px] border border-border bg-muted p-3">
        <div className="min-w-0 flex-1">
          <p className="text-[13px] font-medium">{copy.title}</p>
          <p className="text-xs text-muted-foreground">{copy.hint}</p>
        </div>
        {permission === "default" ? (
          <Button
            type="button"
            size="sm"
            variant="outline"
            onClick={() =>
              void Notification.requestPermission().then(setPermission)
            }
          >
            Allow notifications
          </Button>
        ) : null}
      </div>
      <table className="w-full text-sm">
        <thead>
          <tr className="text-left text-xs font-medium tracking-[0.02em] text-muted-foreground">
            <th className="px-2 py-3 font-medium">Event</th>
            {columns.map((column) => (
              <th key={column.channel} className="px-2 py-3 font-medium">
                {column.label}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {notifyKinds.map((kind) => (
            <tr key={kind} className="border-t border-border">
              <td className="px-2 py-3 align-top">
                <div className="text-[13px] leading-5 font-medium">
                  {kindCopy[kind].label}
                </div>
                <div className="text-xs text-muted-foreground">
                  {kindCopy[kind].hint}
                </div>
              </td>
              {columns.map((column) => (
                <td key={column.channel} className="px-2 py-3 align-top">
                  <Checkbox
                    aria-label={`${kindCopy[kind].label}: ${column.label.toLowerCase()}`}
                    checked={prefs.events[kind][column.channel]}
                    disabled={column.channel === "system" && systemOff}
                    onCheckedChange={(on) =>
                      toggle(kind, column.channel, on === true)
                    }
                  />
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
      <Setting
        name="Only when Span is in the background"
        hint="Skip toasts while this tab is in front. Default: on."
        label="Only when Span is in the background"
        on={prefs.background}
        onChange={(on) => keep({ ...prefs, background: on })}
      />
      <Setting
        name="Sound"
        hint="A soft tone with each toast. Default: off."
        label="Sound"
        on={prefs.sound}
        onChange={(on) => {
          keep({ ...prefs, sound: on })
          if (on) {
            playTone()
          }
        }}
      />
    </section>
  )
}
