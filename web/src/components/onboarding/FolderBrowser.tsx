"use client"

import { useEffect, useState } from "react"

import { Button } from "@/components/ui/button"
import { folderListingOf, getJSON } from "@/lib/api"
import { reasonText } from "@/lib/guards"
import type { FolderListing } from "@/lib/types"

/** A plain folder picker: the directories under a path, a way up, and a way to choose one. */
export function FolderBrowser({
  start,
  onPick,
}: {
  start: string
  onPick: (path: string) => void
}) {
  const [path, setPath] = useState(start.trim())
  const [listing, setListing] = useState<FolderListing | null>(null)
  const [note, setNote] = useState("")

  useEffect(() => {
    let live = true
    getJSON(`/onboarding/folders?path=${encodeURIComponent(path)}`)
      .then((body) => {
        if (live) {
          setListing(folderListingOf(body))
          setNote("")
        }
      })
      .catch((reason: unknown) => {
        if (live) {
          setNote(reasonText(reason, "that folder could not be listed"))
        }
      })
    return () => {
      live = false
    }
  }, [path])

  return (
    <div
      role="group"
      aria-label="Choose a folder"
      className="flex flex-col gap-2 rounded-[10px] border border-border bg-muted p-3"
    >
      <div className="flex flex-wrap items-center gap-2">
        <Button
          type="button"
          size="sm"
          variant="outline"
          disabled={listing === null}
          onClick={() => listing !== null && setPath(listing.parent)}
        >
          Up
        </Button>
        <span className="min-w-0 flex-1 truncate font-mono text-xs">
          {listing?.path ?? path}
        </span>
        <Button
          type="button"
          size="sm"
          disabled={listing === null}
          onClick={() => listing !== null && onPick(listing.path)}
        >
          Use this folder
        </Button>
      </div>
      {note !== "" ? (
        <p role="alert" className="text-sm text-destructive">
          {note}
        </p>
      ) : null}
      <ul className="flex max-h-56 flex-col overflow-y-auto">
        {(listing?.dirs ?? []).map((dir) => (
          <li key={dir.path}>
            <button
              type="button"
              onClick={() => setPath(dir.path)}
              className="flex w-full items-center justify-between gap-2 rounded-md px-2 py-1.5 text-left text-[13px] hover:bg-accent"
            >
              <span className="truncate">{dir.name}</span>
              {dir.git ? (
                <span className="font-mono text-xs text-pass">git</span>
              ) : null}
            </button>
          </li>
        ))}
        {listing !== null && listing.dirs.length === 0 ? (
          <li className="px-2 py-1.5 text-[13px] text-muted-foreground">
            No folders here.
          </li>
        ) : null}
      </ul>
    </div>
  )
}
