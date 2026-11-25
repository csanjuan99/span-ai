"use client"

import { useEffect, useState, type ReactNode } from "react"

import { NotificationPrefs } from "@/components/notify/NotificationPrefs"
import { AgentPicker } from "@/components/agent-picker/AgentPicker"
import { ApprovalsFields } from "@/components/settings-edit/ApprovalsFields"
import { ChecksFields } from "@/components/settings-edit/ChecksFields"
import { JiraSection } from "@/components/settings-edit/JiraSection"
import { SettingsNav } from "@/components/settings-edit/SettingsNav"
import {
  changedEdits,
  editCount,
  type Draft,
  type DraftKey,
} from "@/components/settings-edit/draft"
import { FieldRow } from "@/components/settings-edit/FieldRow"
import { TextField, type EditProps } from "@/components/settings-edit/fields"
import { useConfigPreview } from "@/components/settings-edit/useConfigPreview"
import { WhatChanges } from "@/components/settings-edit/WhatChanges"
import { Shell } from "@/components/shell/Shell"
import { useOptionalTheme } from "@/components/theme-provider"
import { Input } from "@/components/ui/input"
import { getJSON, projectConfig, saveConfig } from "@/lib/api"
import { reasonText } from "@/lib/guards"
import type { ProjectConfig } from "@/lib/types"
import { cn } from "@/lib/utils"

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
      className="flex scroll-mt-4 flex-col gap-1 rounded-[14px] border border-border bg-card p-4 shadow-sm sm:p-5"
    >
      <h2 className="text-lg font-semibold">{name}</h2>
      <p className="mb-1 text-[13px] text-muted-foreground">{hint}</p>
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
  const [current, setCurrent] = useState("Project")

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
      <header className="flex flex-col gap-1">
        <h1 className="text-2xl font-semibold tracking-tight">Settings</h1>
        <p className="text-[13px] text-muted-foreground">
          Defaults are shown beside each value. Changed values are marked.
        </p>
      </header>
      <div className="grid gap-6 lg:grid-cols-[200px_minmax(0,1fr)_340px]">
        <SettingsNav current={current} onPick={setCurrent} />
        <div className="flex min-w-0 flex-col gap-6">
          {note !== "" ? (
            <p className="text-sm text-pretty text-destructive">{note}</p>
          ) : null}
          {config !== null && config.reason !== "" ? (
            <p className="text-sm text-pretty text-hold">{config.reason}</p>
          ) : null}
          <Section name="Project" hint="Where Span works.">
            <FieldRow
              label="Project folder"
              fallback="the folder Span was started in"
              changed={false}
              unsaved={false}
            >
              <Input
                aria-label="Project folder"
                className="h-auto rounded-[10px] border-input bg-card px-3 py-2.5 text-sm shadow-sm dark:bg-card"
                value={folder}
                onChange={(event) => changeFolder(event.target.value)}
              />
            </FieldRow>
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
            name="Jira"
            hint="Optional. Reads stories and moves them when a run finishes."
          >
            <JiraSection folder={folder} />
          </Section>
          <Section
            name="Appearance"
            hint="How Span looks. Saved in this browser."
          >
            {theme !== undefined ? (
              <FieldRow
                label="Theme"
                fallback="System"
                changed={theme.theme !== "system"}
                unsaved={false}
              >
                <div
                  role="group"
                  aria-label="Theme"
                  className="flex w-fit gap-1 rounded-[14px] bg-muted p-1"
                >
                  {themes.map((item) => (
                    <button
                      key={item.value}
                      type="button"
                      aria-pressed={theme.theme === item.value}
                      onClick={() => theme.setTheme(item.value)}
                      className={cn(
                        "h-9 rounded-[10px] px-4 text-[13px] font-medium text-muted-foreground",
                        "aria-pressed:bg-card aria-pressed:font-semibold aria-pressed:text-primary aria-pressed:shadow-sm"
                      )}
                    >
                      {item.label}
                    </button>
                  ))}
                </div>
                <p className="text-xs text-muted-foreground">
                  Motion follows your system. Reduced motion turns off the
                  start-up animation and skeleton shimmer.
                </p>
              </FieldRow>
            ) : null}
          </Section>
          <Section
            name="Notifications"
            hint="When Span tells you a run needs you, failed or finished. Saved in this browser."
          >
            <NotificationPrefs />
          </Section>
        </div>
        <div className="flex min-w-0 flex-col gap-4 lg:sticky lg:top-4 lg:self-start">
          {done !== "" ? (
            <p role="status" className="text-sm text-pass">
              {done}
            </p>
          ) : null}
          <WhatChanges
            folder={folder}
            count={count}
            lines={preview.lines}
            error={preview.error}
            saving={saving}
            onSave={save}
            onDiscard={() => setDraft({})}
          />
        </div>
      </div>
    </Shell>
  )
}
