"use client"

import { MenuIcon, SearchIcon, XIcon } from "lucide-react"
import { useState } from "react"
import { Link, NavLink } from "react-router"

import { Brand } from "@/components/shell/Brand"
import { ThemeToggle } from "@/components/theme-toggle/ThemeToggle"
import { openPaletteEvent } from "@/lib/help"
import { cn } from "@/lib/utils"

const links = [
  { to: "/", label: "Desk", end: true },
  { to: "/shaping", label: "Shape a story", end: false },
  { to: "/settings", label: "Settings", end: false },
] as const

function linkClass({ isActive }: { isActive: boolean }) {
  return cn(
    "rounded-[10px] px-3 py-1 text-[13px] leading-5 text-muted-foreground hover:text-foreground",
    isActive && "bg-accent font-semibold text-primary hover:text-primary"
  )
}

const iconButton =
  "grid size-11 place-items-center rounded-[10px] border border-input bg-card text-foreground sm:hidden"

/** The bar every page shares: where you are, the two places to go, search, and the theme. */
export function AppBar({ path }: { path: string }) {
  const [menu, setMenu] = useState(false)
  const openPalette = () => window.dispatchEvent(new Event(openPaletteEvent))
  return (
    <header className="relative flex h-14 shrink-0 items-center gap-3 border-b border-border bg-card px-4 sm:gap-6 sm:px-10">
      <Link to="/" className="flex items-center gap-2.5 hover:no-underline">
        <Brand />
      </Link>
      <nav aria-label="Main" className="hidden gap-1 sm:flex">
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
      <button
        type="button"
        aria-label="Search or run a command"
        onClick={openPalette}
        className={cn(iconButton, "ml-auto")}
      >
        <SearchIcon className="size-4" aria-hidden="true" />
      </button>
      <button
        type="button"
        aria-label="Menu"
        aria-expanded={menu}
        onClick={() => setMenu((open) => !open)}
        className={iconButton}
      >
        {menu ? (
          <XIcon className="size-4" aria-hidden="true" />
        ) : (
          <MenuIcon className="size-4" aria-hidden="true" />
        )}
      </button>
      <div className="ml-auto hidden min-w-0 items-center gap-3 sm:flex">
        <button
          type="button"
          onClick={openPalette}
          className="inline-flex h-8 items-center gap-2 rounded-[10px] border border-input bg-card px-2.5 text-[13px] text-muted-foreground hover:text-foreground"
        >
          <SearchIcon className="size-3.5" aria-hidden="true" />
          Search or run a command
          <kbd className="inline-flex h-[22px] min-w-[22px] items-center justify-center rounded-md border border-b-2 border-input bg-card px-1.5 font-mono text-xs text-foreground">
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
      {menu ? (
        <div className="absolute inset-x-0 top-14 z-40 flex flex-col gap-1 border-b border-border bg-card p-3 shadow-md sm:hidden">
          {links.map((link) => (
            <NavLink
              key={link.to}
              to={link.to}
              end={link.end}
              onClick={() => setMenu(false)}
              className={(state) =>
                cn(linkClass(state), "flex min-h-11 items-center px-3")
              }
            >
              {link.label}
            </NavLink>
          ))}
          <div className="flex items-center justify-between gap-3 px-3 pt-2">
            <span
              title={path}
              className="min-w-0 truncate font-mono text-xs text-muted-foreground"
            >
              {path}
            </span>
            <ThemeToggle />
          </div>
        </div>
      ) : null}
    </header>
  )
}
