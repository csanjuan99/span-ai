"use client"

import { CheckIcon } from "lucide-react"
import { useState } from "react"

import { CheckLine } from "@/components/onboarding/CheckLine"
import { FixBox } from "@/components/onboarding/FixBox"
import { FolderBrowser } from "@/components/onboarding/FolderBrowser"
import { Panel } from "@/components/onboarding/Frame"
import { Pill } from "@/components/onboarding/Pill"
import { usable, type Probe } from "@/components/onboarding/use-folder-probe"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"

/** The project folder, checked as it is typed: found, a git repository, and anything uncommitted. */
export function FolderPanel({
  folder,
  onFolder,
  probe,
}: {
  folder: string
  onFolder: (value: string) => void
  probe: Probe
}) {
  const [browsing, setBrowsing] = useState(false)
  const found = probe.state === "found" ? probe.folder : null

  return (
    <Panel
      title="Project folder"
      badge={
        usable(probe) ? (
          <Pill tone="pass">
            <CheckIcon aria-hidden="true" className="size-3" />
            Ready
          </Pill>
        ) : null
      }
      lead="The folder with your code. Span creates branches and runs your checks here."
    >
      <div className="flex gap-2">
        <Input
          aria-label="Project folder"
          className="h-auto flex-1 rounded-[10px] border-input bg-card px-3 py-2.5 font-mono text-sm shadow-sm dark:bg-card"
          placeholder="/path/to/your/project"
          value={folder}
          onChange={(event) => onFolder(event.target.value)}
        />
        <Button
          type="button"
          variant="outline"
          className="h-auto"
          aria-expanded={browsing}
          onClick={() => setBrowsing((open) => !open)}
        >
          Browse
        </Button>
      </div>
      {browsing ? (
        <FolderBrowser
          start={folder}
          onPick={(path) => {
            onFolder(path)
            setBrowsing(false)
          }}
        />
      ) : null}
      <ul className="flex flex-col gap-1">
        {probe.state === "failed" ? (
          <CheckLine tone="fail">{probe.reason}</CheckLine>
        ) : null}
        {found !== null && !found.exists ? (
          <CheckLine
            tone="fail"
            fix={
              <FixBox>
                Check the path, or press Browse and pick the folder.
              </FixBox>
            }
          >
            <b className="text-fail">Not found.</b> {found.reason}
          </CheckLine>
        ) : null}
        {found !== null && found.exists ? (
          <CheckLine tone="pass">
            <b className="text-pass">Found.</b> The folder exists and is
            readable.
          </CheckLine>
        ) : null}
        {found !== null && found.exists && !found.git ? (
          <CheckLine
            tone="fail"
            fix={
              <FixBox command="git init">
                Run it in the folder, or pick the folder that holds your code.
              </FixBox>
            }
          >
            <b className="text-fail">Not a git repository.</b> Span cuts a
            branch for every run, so it needs one.
          </CheckLine>
        ) : null}
        {found !== null && found.git ? (
          <CheckLine tone="pass">
            <b className="text-pass">Git repository.</b>{" "}
            {found.branch !== "" ? (
              <>
                On branch{" "}
                <span className="font-mono text-xs">{found.branch}</span>
              </>
            ) : (
              "No branch checked out"
            )}
            {found.remote !== "" ? (
              <>
                {" "}
                with remote{" "}
                <span className="font-mono text-xs">{found.remote}</span>
              </>
            ) : (
              " and no remote"
            )}
            .
          </CheckLine>
        ) : null}
        {found !== null && found.git && found.dirty > 0 ? (
          <CheckLine
            tone="hold"
            fix={
              <FixBox command="git stash --include-untracked">
                Commit or stash them if you want a clean start.
              </FixBox>
            }
          >
            <b className="text-hold">
              {found.dirty} uncommitted {found.dirty === 1 ? "file" : "files"}.
            </b>{" "}
            Runs use their own branch, so these stay where they are.
          </CheckLine>
        ) : null}
      </ul>
    </Panel>
  )
}
