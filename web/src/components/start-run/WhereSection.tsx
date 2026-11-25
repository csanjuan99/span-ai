"use client"

import { FolderIcon, SearchIcon } from "lucide-react"
import { useState } from "react"

import { StartCard } from "@/components/start-run/StartCard"
import { Pill } from "@/components/shaping/session-ui"
import { noteText } from "@/components/shaping/styles"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { cn } from "@/lib/utils"

/** A repository found in the folder, and whether a run will start in it. */
export interface RepoRow {
  name: string
  checked: boolean
}

/** What the Where step shows and reports back. */
export interface WhereProps {
  workspace: string
  onWorkspace: (value: string) => void
  repos: RepoRow[]
  onToggleRepo: (name: string, checked: boolean) => void
  folderNote: string
  onFindRepos: () => void
}

/** N1 step 1: the folder the code lives in and the repositories a run starts in. */
export function WhereSection({
  workspace,
  onWorkspace,
  repos,
  onToggleRepo,
  folderNote,
  onFindRepos,
}: WhereProps) {
  const [editing, setEditing] = useState(false)
  const chosen = repos.filter((repo) => repo.checked).length
  return (
    <StartCard step={1} title="Where" hint="The folder your code lives in.">
      {editing || repos.length > 1 ? (
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="workspace">Project folder</Label>
          <div className="flex flex-wrap gap-2">
            <Input
              id="workspace"
              name="workspace"
              className="h-11 min-w-0 flex-[1_1_240px] md:h-10"
              value={workspace}
              onChange={(event) => onWorkspace(event.target.value)}
            />
            <Button
              type="button"
              variant="outline"
              className="h-11 md:h-10"
              onClick={onFindRepos}
            >
              <SearchIcon aria-hidden="true" />
              Find repos
            </Button>
          </div>
        </div>
      ) : (
        <div className="flex flex-wrap items-center gap-2">
          <FolderIcon
            aria-hidden="true"
            className="size-4 text-muted-foreground"
          />
          <span className="font-mono text-[13px]">{workspace}</span>
          {workspace === "." ? <Pill tone="acc">Span started here</Pill> : null}
          <input type="hidden" name="workspace" value={workspace} />
          <Button
            type="button"
            variant="outline"
            className="h-11 md:h-10"
            onClick={() => setEditing(true)}
          >
            Change folder
          </Button>
        </div>
      )}
      {folderNote !== "" ? (
        <p className="text-sm text-pretty text-destructive">{folderNote}</p>
      ) : null}
      {repos.length > 0 ? (
        <ul className="flex flex-col gap-2">
          {repos.map((repo) => (
            <li key={repo.name}>
              <label
                className={cn(
                  "flex min-h-14 items-center gap-3 rounded-[10px] border border-input bg-card px-3 text-sm font-medium",
                  repo.checked && "border-primary bg-accent"
                )}
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
            </li>
          ))}
        </ul>
      ) : null}
      {repos.length > 1 ? (
        <p className={noteText}>
          {chosen} of {repos.length} selected. Each run uses its own branch, so
          uncommitted files stay where they are. Span starts one run per
          selected repository.
        </p>
      ) : null}
    </StartCard>
  )
}
