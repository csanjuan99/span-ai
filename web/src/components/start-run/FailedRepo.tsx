"use client"

import { RefreshCwIcon } from "lucide-react"
import { useEffect, useState } from "react"

import { Pill } from "@/components/shaping/session-ui"
import { Button } from "@/components/ui/button"
import { readBranchChoice } from "@/lib/branches"
import { touchButton } from "@/lib/button-class"
import { reasonText } from "@/lib/guards"
import type { BranchChoice } from "@/lib/types"

/**
 * The base to retry with when the one that was wanted is not in the repository; empty when
 * it is there, so a failure for another reason is retried as it was asked.
 */
function missingBaseFix(
  { branches, defaultBase }: BranchChoice,
  tried: string
): string {
  if (branches.includes(tried === "" ? defaultBase : tried)) {
    return ""
  }
  return (
    [defaultBase, "main", "master"].find((name) => branches.includes(name)) ??
    ""
  )
}

/**
 * N3: a repository that did not start. It can be retried from the branch it does call its
 * default, or left for later.
 */
export function FailedRepo({
  repo,
  reason,
  workspace,
  tried,
  onRetry,
  onSkip,
}: {
  repo: string
  reason: string
  /** The folder that holds the repositories. */
  workspace: string
  /** The base that was asked for; empty when Span chose it. */
  tried: string
  onRetry: (repo: string, base: string) => Promise<void>
  onSkip: (repo: string) => void
}) {
  const [base, setBase] = useState("")
  const [busy, setBusy] = useState(false)
  const [problem, setProblem] = useState("")

  useEffect(() => {
    let live = true
    const path = `${workspace.replace(/\/+$/, "")}/${repo}`
    readBranchChoice(path)
      .then((choice) => {
        if (live) {
          setBase(missingBaseFix(choice, tried))
        }
      })
      .catch((err: unknown) => {
        if (live) {
          setProblem(reasonText(err, "its branches could not be read"))
        }
      })
    return () => {
      live = false
    }
  }, [workspace, repo, tried])

  async function retry() {
    setBusy(true)
    setProblem("")
    try {
      await onRetry(repo, base)
    } catch (err: unknown) {
      setProblem(reasonText(err, "the run did not start"))
    } finally {
      setBusy(false)
    }
  }

  return (
    <li className="flex flex-col gap-2 border-b border-border px-4 py-3 last:border-b-0">
      <span className="flex flex-wrap items-center gap-2 text-sm font-semibold">
        {repo}
        <Pill tone="fail" small>
          Not started
        </Pill>
      </span>
      <p className="text-sm text-pretty">{reason}</p>
      <p className="text-xs text-muted-foreground">
        Nothing was created in this repository.
      </p>
      {problem !== "" ? (
        <p role="alert" className="text-sm text-destructive">
          {problem}
        </p>
      ) : null}
      <div className="flex flex-wrap gap-2">
        <Button
          type="button"
          className={touchButton}
          disabled={busy}
          onClick={() => void retry()}
        >
          <RefreshCwIcon aria-hidden="true" />
          {base === "" ? "Retry" : `Retry with base ${base}`}
        </Button>
        <Button
          type="button"
          variant="ghost"
          className={touchButton}
          disabled={busy}
          onClick={() => onSkip(repo)}
        >
          Skip this one
        </Button>
      </div>
    </li>
  )
}
