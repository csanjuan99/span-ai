"use client"

import { useEffect, useState, type ReactNode } from "react"
import { Link } from "react-router"

import { NotificationPrefs } from "@/components/notify/NotificationPrefs"
import { AgentPicker } from "@/components/agent-picker/AgentPicker"
import { ApprovalsFields } from "@/components/settings-edit/ApprovalsFields"
import { ChecksFields } from "@/components/settings-edit/ChecksFields"
import {
  changedEdits,
  editCount,
  type Draft,
  type DraftKey,
} from "@/components/settings-edit/draft"
import { TextField, type EditProps } from "@/components/settings-edit/fields"
import { useConfigPreview } from "@/components/settings-edit/useConfigPreview"
import { WhatChanges } from "@/components/settings-edit/WhatChanges"
import { Shell } from "@/components/shell/Shell"
import { useOptionalTheme } from "@/components/theme-provider"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { getJSON, projectConfig, saveConfig } from "@/lib/api"
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

/** Project settings: what span.yaml says, the agent that writes, and how the page looks. */
export function ProjectSettings() {
  const [folder, setFolder] = useState(".")
  const [config, setConfig] = useState<ProjectConfig | null>(null)
  const [note, setNote] = useState("")
  const [draft, setDraft] = useState<Draft>({})
  const [saving, setSaving] = useState(false)
  const [done, setDone] = useState("")
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

  const view = config ?? projectConfig({})
  const edits = changedEdits(view, draft)
  const count = editCount(edits)
  const preview = useConfigPreview(folder, edits, count > 0)

  function edit(key: DraftKey, value: string) {
    setDraft((now) => ({ ...now, [key]: value }))
    setDone("")
  }

  function changeFolder(value: string) {
    setFolder(value)
    setDraft({})
    setDone("")
  }

  function save() {
    setSaving(true)
    setNote("")
    void saveConfig(folder, edits)
      .then((saved) => {
        setConfig(saved.config)
        setDraft({})
        setDone(
          `Saved ${saved.lines.length} ${saved.lines.length === 1 ? "change" : "changes"} to span.yaml.`
        )
      })
      .catch((err: unknown) => {
        setNote(reasonText(err, "span.yaml could not be saved"))
      })
      .finally(() => setSaving(false))
  }

  const props: EditProps = {
    config: view,
    draft,
    disabled: config === null || saving,
    onEdit: edit,
  }

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
          The agent, base branch, checks and approvals are saved to span.yaml,
          and how the page looks is saved in this browser. Changes show up below
          before anything is written.
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
                onChange={(event) => changeFolder(event.target.value)}
              />
              <span className="text-xs text-muted-foreground">
                Default: the folder Span was started in
              </span>
            </div>
            <TextField
              props={props}
              name="base"
              label="Base branch"
              fallback="dev (master for a hotfix)"
              placeholder="dev"
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
            hint="Commands that prove the work. Saved to span.yaml."
          >
            <ChecksFields props={props} />
          </Section>
          <Section
            name="Approvals"
            hint="Which stages stop and wait for your yes. Saved to span.yaml."
          >
            <ApprovalsFields props={props} />
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
          <Section
            name="Notifications"
            hint="When Span tells you a run needs you, failed or finished. Saved in this browser."
          >
            <NotificationPrefs />
          </Section>
          {done !== "" ? (
            <p role="status" className="text-sm text-pass">
              {done}
            </p>
          ) : null}
          {count > 0 ? (
            <WhatChanges
              folder={folder}
              count={count}
              lines={preview.lines}
              error={preview.error}
              saving={saving}
              onSave={save}
              onDiscard={() => setDraft({})}
            />
          ) : null}
        </div>
      </div>
    </Shell>
  )
}
