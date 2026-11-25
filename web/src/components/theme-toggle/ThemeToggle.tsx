"use client"

import { Monitor, Moon, Sun } from "lucide-react"

import { useOptionalTheme } from "@/components/theme-provider"
import { Button } from "@/components/ui/button"

const order = ["system", "light", "dark"] as const
const icons = { system: Monitor, light: Sun, dark: Moon }

/** Cycles the theme: follow the system, then light, then dark. The `d` key still flips it. */
export function ThemeToggle() {
  const state = useOptionalTheme()
  if (state === undefined) {
    return null
  }
  const { theme, setTheme } = state
  const next = order[(order.indexOf(theme) + 1) % order.length] ?? "system"
  const Icon = icons[theme]
  return (
    <Button
      type="button"
      variant="ghost"
      size="icon-sm"
      aria-label={`Theme: ${theme}. Switch to ${next}`}
      title={`Theme: ${theme}`}
      onClick={() => setTheme(next)}
    >
      <Icon />
    </Button>
  )
}
