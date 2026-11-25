"use client"

import type { FormEvent, RefObject } from "react"

import { AgentPicker } from "@/components/agent-picker/AgentPicker"
import { JiraLookup, type StoryHit } from "@/components/start-run/JiraLookup"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Textarea } from "@/components/ui/textarea"
import { basePlaceholder } from "@/lib/api"
import type { AgentChoice, BranchChoice } from "@/lib/types"
import type { JiraSession } from "@/lib/jira-session"

export interface RepoRow {
  name: string
  checked: boolean
}

/** Everything the start form shows and reports back. */
export interface StartFormProps {
  formRef: RefObject<HTMLFormElement | null>
  workspace: string
  onWorkspace: (value: string) => void
  repos: RepoRow[]
  onToggleRepo: (name: string, checked: boolean) => void
  folderNote: string
  onFindRepos: () => void
  pick: AgentChoice | null
  onPickLoaded: (value: AgentChoice) => void
  issue: string
  onIssue: (value: string) => void
  kind: string
  onKind: (value: string) => void
  base: string
  onBase: (value: string) => void
  branches: BranchChoice
  story: string
  onStory: (value: string) => void
  session: JiraSession
  jiraUrl: string
  onJiraUrl: (value: string) => void
  jiraEmail: string
  onJiraEmail: (value: string) => void
  jiraToken: string
  onJiraToken: (value: string) => void
  stories: StoryHit[]
  jiraNote: string
  storyNote: string
  onLoadStory: () => void
  onLoadMine: () => void
  onSaveSession: (event: FormEvent<HTMLFormElement>) => void
  onPickStory: (key: string) => void
  runNote: string
  onSubmit: (event: FormEvent<HTMLFormElement>) => void
}

/** N1: start a run form. */
export function StartForm({
  formRef,
  workspace,
  onWorkspace,
  repos,
  onToggleRepo,
  folderNote,
  onFindRepos,
  pick,
  onPickLoaded,
  issue,
  onIssue,
  kind,
  onKind,
  base,
  onBase,
  branches,
  story,
  onStory,
  session,
  jiraUrl,
  onJiraUrl,
  jiraEmail,
  onJiraEmail,
  jiraToken,
  onJiraToken,
  stories,
  jiraNote,
  storyNote,
  onLoadStory,
  onLoadMine,
  onSaveSession,
  onPickStory,
  runNote,
  onSubmit,
}: StartFormProps) {
  return (
    <form
      ref={formRef}
      className="flex flex-col gap-4"
      onSubmit={onSubmit}
      aria-label="Start a run"
    >
      <header className="flex flex-col gap-1">
        <h1 className="text-2xl font-semibold tracking-tight">Start a run</h1>
        <p className="text-[13px] text-muted-foreground">
          Choose the folder, the story and how to work. Nothing is created until
          you press Start.
        </p>
      </header>
      <div className="flex flex-col gap-2">
        <Label htmlFor="workspace">Folder</Label>
        <Input
          id="workspace"
          name="workspace"
          value={workspace}
          onChange={(event) => onWorkspace(event.target.value)}
        />
      </div>
      <div id="agent-field">
        <AgentPicker project={workspace} onLoaded={onPickLoaded} />
      </div>
      <div className="flex flex-col gap-2">
        <Button
          type="button"
          variant="secondary"
          className="min-h-11 self-start"
          onClick={onFindRepos}
        >
          Find repos
        </Button>
        {folderNote !== "" ? (
          <p className="text-sm text-pretty text-destructive">{folderNote}</p>
        ) : null}
        {repos.map((repo) => (
          <label
            key={repo.name}
            className="flex min-h-11 items-center gap-2 text-sm"
          >
            <Checkbox
              checked={repo.checked}
              onCheckedChange={(checked) =>
                onToggleRepo(repo.name, checked === true)
              }
            />
            {repo.name}
            {repo.checked ? (
              <input type="hidden" name="repo" value={repo.name} />
            ) : null}
          </label>
        ))}
      </div>
      <JiraLookup
        session={session}
        issue={issue}
        onIssue={onIssue}
        url={jiraUrl}
        onUrl={onJiraUrl}
        email={jiraEmail}
        onEmail={onJiraEmail}
        token={jiraToken}
        onToken={onJiraToken}
        stories={stories}
        note={jiraNote !== "" ? jiraNote : storyNote}
        onLoadOne={onLoadStory}
        onLoadMine={onLoadMine}
        onSaveSession={onSaveSession}
        onPick={(key) => {
          onIssue(key)
          onPickStory(key)
        }}
      />
      <div className="flex flex-col gap-2">
        <Label htmlFor="kind">Kind</Label>
        <Select value={kind} onValueChange={onKind}>
          <SelectTrigger id="kind" className="h-11 w-40">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="infer">infer</SelectItem>
            <SelectItem value="feat">feat</SelectItem>
            <SelectItem value="fix">fix</SelectItem>
            <SelectItem value="hotfix">hotfix</SelectItem>
          </SelectContent>
        </Select>
        {kind !== "infer" ? (
          <input type="hidden" name="kind" value={kind} />
        ) : null}
      </div>
      <div className="flex flex-col gap-2">
        <Label htmlFor="base">Base branch</Label>
        <Input
          id="base"
          name="base"
          list="base-branches"
          className="h-11 w-64"
          placeholder={basePlaceholder(kind, branches.defaultBase)}
          value={base}
          onChange={(event) => onBase(event.target.value)}
        />
        <datalist id="base-branches">
          {branches.branches.map((name) => (
            <option key={name} value={name} />
          ))}
        </datalist>
      </div>
      <div className="flex flex-col gap-2">
        <Label htmlFor="story">Story</Label>
        <Textarea
          id="story"
          name="story"
          value={story}
          onChange={(event) => onStory(event.target.value)}
          rows={8}
        />
      </div>
      <div className="flex flex-col gap-2">
        <Label htmlFor="file">Handoff files</Label>
        <Input
          id="file"
          name="file"
          type="file"
          multiple
          className="min-h-11 max-w-full"
        />
      </div>
      <div className="flex flex-wrap items-center gap-3">
        <Button type="submit" className="min-h-11">
          Start run
        </Button>
        {runNote !== "" ? (
          <p className="text-sm text-pretty text-destructive">{runNote}</p>
        ) : null}
      </div>
      {pick === null ? null : (
        <p className="sr-only" aria-live="polite">
          Agent loaded
        </p>
      )}
    </form>
  )
}
