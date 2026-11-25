"use client"

import { SearchIcon } from "lucide-react"
import { Link, NavLink } from "react-router"

import { ThemeToggle } from "@/components/theme-toggle/ThemeToggle"
import { openPaletteEvent } from "@/lib/help"
import { cn } from "@/lib/utils"

const links = [
  { to: "/", label: "Desk", end: true },
  { to: "/settings", label: "Settings", end: false },
] as const

function linkClass({ isActive }: { isActive: boolean }) {
  return cn(
    "rounded-[10px] px-3 py-1 text-[13px] leading-5 text-muted-foreground hover:text-foreground",
    isActive && "bg-accent font-semibold text-primary hover:text-primary"
  )
}

/** The bar every page shares: where you are, the two places to go, search, and the theme. */
export function AppBar({ path }: { path: string }) {
  return (
    <header className="flex h-14 shrink-0 items-center gap-3 border-b border-border bg-card px-4 sm:gap-6 sm:px-10">
      <Link to="/" className="flex items-center gap-2.5 hover:no-underline">
        <span
          aria-hidden="true"
          className="grid size-6 place-items-center rounded-md bg-primary font-mono text-[13px] font-semibold text-primary-foreground"
        >
          S
        </span>
        <span className="text-sm font-semibold text-foreground">Span AI</span>
      </Link>
      <nav aria-label="Main" className="flex gap-1">
        {links.map((link) => (
          <NavLink
            key={link.to}
            to={link.to}
            end={link.end}
            className={linkClass}
          >
            {link.label}
          </NavLink>
        ))}
      </nav>
      <div className="ml-auto flex min-w-0 items-center gap-3">
        <button
          type="button"
          onClick={() => window.dispatchEvent(new Event(openPaletteEvent))}
          className="inline-flex h-8 items-center gap-2 rounded-[10px] border border-input bg-card px-2.5 text-[13px] text-muted-foreground hover:text-foreground"
        >
          <SearchIcon className="size-3.5" aria-hidden="true" />
          <span className="hidden sm:inline">Search or run a command</span>
          <span className="sr-only sm:hidden">Search</span>
          <kbd className="hidden h-[22px] min-w-[22px] items-center justify-center rounded-md border border-b-2 border-input bg-card px-1.5 font-mono text-xs text-foreground sm:inline-flex">
            ⌘K
          </kbd>
        </button>
        <span
          className="hidden h-3.5 w-px bg-border md:block"
          aria-hidden="true"
        />
        <span
          title={path}
          className="hidden max-w-64 min-w-0 truncate font-mono text-xs text-muted-foreground tabular-nums md:block"
        >
          {path}
        </span>
        <ThemeToggle />
      </div>
    </header>
  )
}
