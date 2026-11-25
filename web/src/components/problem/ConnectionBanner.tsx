"use client"

import { CheckIcon, WifiOffIcon } from "lucide-react"
import type { ReactNode } from "react"

import type { ConnectionNotice } from "@/hooks/use-connection-notice"
import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"

const tones = {
  hold: "bg-hold-soft text-hold",
  pass: "bg-pass-soft text-pass",
  fail: "bg-fail-soft text-fail",
}

function Banner({
  tone,
  icon,
  children,
  retry,
}: {
  tone: keyof typeof tones
  icon: ReactNode
  children: ReactNode
  retry?: string
}) {
  return (
    <p
      role="status"
      className={cn(
        "flex flex-wrap items-center gap-3 rounded-[14px] px-4 py-2.5 text-[13px] leading-5 md:px-5",
        tones[tone]
      )}
    >
      {icon}
      <span>{children}</span>
      {retry !== undefined ? (
        <Button
          type="button"
          variant="outline"
          size="sm"
          className="ml-auto h-8 rounded-[10px] bg-card px-3 text-[13px] text-foreground"
          onClick={() => window.location.reload()}
        >
          {retry}
        </Button>
      ) : null}
    </p>
  )
}

/**
 * The line under the header while the live connection is down: calm for the first minute, a
 * short one when it returns, and a plain statement when nothing answers after a minute.
 */
export function ConnectionBanner({
  notice,
  lastSeen,
}: {
  notice: ConnectionNotice
  lastSeen: string
}) {
  if (notice === "idle") {
    return null
  }
  if (notice === "back") {
    return (
      <Banner
        tone="pass"
        icon={<CheckIcon aria-hidden="true" className="size-4" />}
      >
        <b>Back online.</b> Updated just now. This message closes by itself.
      </Banner>
    )
  }
  if (notice === "lost") {
    return (
      <Banner
        tone="fail"
        icon={<WifiOffIcon aria-hidden="true" className="size-4" />}
        retry="Retry"
      >
        <b>Span is not answering.</b>
        {lastSeen !== "" ? ` Last seen ${lastSeen}.` : ""} Nothing was lost.
      </Banner>
    )
  }
  return (
    <Banner
      tone="hold"
      icon={<WifiOffIcon aria-hidden="true" className="size-4" />}
      retry="Retry now"
    >
      <b>Reconnecting to Span…</b> Your run keeps going on your machine.
      {lastSeen !== "" ? ` Last updated ${lastSeen}.` : ""}
    </Banner>
  )
}
