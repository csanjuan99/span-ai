"use client"

import { useEffect, useState, type ReactNode } from "react"
import { Link } from "react-router"

import { AgentPicker } from "@/components/agent-picker/AgentPicker"
import { Shell } from "@/components/shell/Shell"
import { useOptionalTheme } from "@/components/theme-provider"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { configDefaults, getJSON, projectConfig } from "@/lib/api"
import { reasonText } from "@/lib/guards"
import type { ProjectConfig } from "@/lib/types"
import { cn } from "@/lib/utils"

const sections = [
  "Project",
  "Agents",
  "Checks",
  "Approvals",
  "Appearance",
] as const

const themes = [
  { value: "system", label: "System" },
  { value: "light", label: "Light" },
  { value: "dark", label: "Dark" },
] as const

function Section({
  name,
  hint,
  children,
}: {
  name: string
  hint: string
  children: ReactNode
}) {
  return (
    <section
      id={name.toLowerCase()}
      aria-label={name}
      className="flex scroll-mt-4 flex-col gap-3 rounded-xl border border-border bg-card p-5 shadow-sm"
    >
      <div className="flex flex-col gap-1">
        <h2 className="text-lg font-semibold">{name}</h2>
        <p className="text-sm text-muted-foreground">{hint}</p>
      </div>
      {children}
    </section>
  )
}

function Row({
  label,
  value,
  fallback,
}: {
  label: string
  value: string
  fallback: string
}) {
  return (
    <div className="grid gap-1 border-t border-border pt-3 first:border-t-0 first:pt-0 sm:grid-cols-[12rem_minmax(0,1fr)]">
      <div className="flex flex-col">
        <span className="text-sm font-medium">{label}</span>
        <span className="text-xs text-muted-foreground">
          Default: {fallback}
        </span>
      </div>
      <div className="flex min-w-0 flex-wrap items-center gap-2">
        <code className="min-w-0 font-mono text-sm break-words">
          {value === "" ? fallback : value}
        </code>
        {value !== "" ? (
          <span className="rounded-md bg-hold-soft px-1.5 py-0.5 text-xs text-hold">
            Changed
          </span>
        ) : null}
      </div>
    </div>
  )
}

function Commands({
  label,
  items,
  fallback,
}: {
  label: string
  items: string[]
  fallback: string
}) {
  return <Row label={label} value={items.join("  ·  ")} fallback={fallback} />
}

const modeLabel: Record<string, string> = {
  auto: "Runs on its own",
  hold: "Asks me",
  conditional: "Asks me when the story is deep",
}

/** Project settings: what span.yaml says, the agent that writes, and how the page looks. */
export function ProjectSettings() {
  const [folder, setFolder] = useState(".")
  const [config, setConfig] = useState<ProjectConfig | null>(null)
  const [note, setNote] = useState("")
  const theme = useOptionalTheme()

  useEffect(() => {
    let live = true
    const timer = window.setTimeout(() => {
      void getJSON(`/config?path=${encodeURIComponent(folder)}`)
        .then((body) => {
          if (live) {
            setConfig(projectConfig(body))
            setNote("")
          }
        })
        .catch((err: unknown) => {
          if (live) {
            setNote(reasonText(err, "span.yaml could not be read"))
          }
        })
    }, 250)
    return () => {
      live = false
      window.clearTimeout(timer)
    }
  }, [folder])

  return (
    <Shell path="~/settings" status="settings" lines={[]}>
      <header className="flex flex-col gap-2">
        <Link
          className="text-sm text-primary underline-offset-4 hover:underline"
          to="/"
        >
          $ desk
        </Link>
        <h1 className="text-xl font-semibold">Settings</h1>
        <p className="text-sm text-pretty text-muted-foreground">
          The agent and how the page looks are saved here. Checks and approvals
          are read from span.yaml: edit that file to change them.
        </p>
      </header>
      <div className="grid gap-6 lg:grid-cols-[10rem_minmax(0,1fr)]">
        <nav
          aria-label="Settings sections"
          className="flex gap-2 overflow-x-auto lg:sticky lg:top-4 lg:flex-col lg:self-start"
        >
          {sections.map((name) => (
            <a
              key={name}
              href={`#${name.toLowerCase()}`}
              className="rounded-lg px-3 py-1.5 text-sm text-muted-foreground hover:bg-muted hover:text-foreground"
            >
              {name}
            </a>
          ))}
        </nav>
        <div className="flex min-w-0 flex-col gap-6">
          {note !== "" ? (
            <p className="text-sm text-pretty text-destructive">{note}</p>
          ) : null}
          {config !== null && config.reason !== "" ? (
            <p className="text-sm text-pretty text-hold">{config.reason}</p>
          ) : null}
          <Section name="Project" hint="Where Span works.">
            <div className="flex flex-col gap-2">
              <Label htmlFor="settings-folder">Project folder</Label>
              <Input
                id="settings-folder"
                value={folder}
                onChange={(event) => setFolder(event.target.value)}
              />
              <span className="text-xs text-muted-foreground">
                Default: the folder Span was started in
              </span>
            </div>
            <Row
              label="Base branch"
              value={config?.base ?? ""}
              fallback="dev (master for a hotfix)"
            />
          </Section>
          <Section
            name="Agents"
            hint="Which CLI writes the code, and on which model. Saved to span.yaml."
          >
            <AgentPicker project={folder} />
          </Section>
          <Section
            name="Checks"
            hint="Commands that prove the work. Read from span.yaml."
          >
            <Commands
              label="Focused"
              items={config?.prove.focused ?? []}
              fallback="none"
            />
            <Commands
              label="Fast"
              items={config?.prove.fast ?? []}
              fallback="none"
            />
            <Commands
              label="Acceptance"
              items={config?.prove.acceptance ?? []}
              fallback="none"
            />
            <Row label="QA command" value={config?.qa ?? ""} fallback="none" />
            <Row
              label="Repair attempts"
              value={
                config !== null && config.repair !== configDefaults.repair
                  ? String(config.repair)
                  : ""
              }
              fallback={String(configDefaults.repair)}
            />
          </Section>
          <Section
            name="Approvals"
            hint="Which stages stop and wait for your yes. Read from span.yaml."
          >
            {(["open", "merge", "promote", "plan", "conflict"] as const).map(
              (stage) => (
                <Row
                  key={stage}
                  label={stage}
                  value={
                    config !== null &&
                    config.stages[stage] !== configDefaults.stages[stage]
                      ? (modeLabel[config.stages[stage]] ??
                        config.stages[stage])
                      : ""
                  }
                  fallback={
                    stage === "conflict" ? "Asks me" : "Runs on its own"
                  }
                />
              )
            )}
          </Section>
          <Section
            name="Appearance"
            hint="How Span looks. Saved in this browser."
          >
            {theme !== undefined ? (
              <div role="group" aria-label="Theme" className="flex gap-2">
                {themes.map((item) => (
                  <button
                    key={item.value}
                    type="button"
                    aria-pressed={theme.theme === item.value}
                    onClick={() => theme.setTheme(item.value)}
                    className={cn(
                      "h-9 rounded-lg border border-border px-3 text-sm text-muted-foreground",
                      "aria-pressed:border-primary aria-pressed:bg-accent aria-pressed:text-primary"
                    )}
                  >
                    {item.label}
                  </button>
                ))}
              </div>
            ) : null}
            <p className="text-xs text-muted-foreground">
              Motion follows your system. With reduced motion the start-up
              animation and skeleton shimmer are off.
            </p>
          </Section>
        </div>
      </div>
    </Shell>
  )
}
