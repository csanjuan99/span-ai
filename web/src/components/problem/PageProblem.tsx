"use client"

import { Link } from "react-router"

import { ProblemCard } from "@/components/problem/ProblemCard"
import { Button, buttonVariants } from "@/components/ui/button"
import type { Problem } from "@/lib/problem"

/** The page when the run cannot be read at all: the server is gone, or the run is not on this desk. */
export function PageProblem({ problem, id }: { problem: Problem; id: string }) {
  if (problem === "unreachable") {
    return (
      <ProblemCard
        label="Server unreachable"
        tone="fail"
        title={`Span can’t reach its server at ${window.location.host}`}
        cause="The Span server is not running, or it started on another port."
        command="span serve"
        kept="Your runs, branches and files are untouched."
        next="Start Span in a terminal, then retry. This page reconnects by itself."
      >
        <Button type="button" onClick={() => window.location.reload()}>
          Retry connection
        </Button>
      </ProblemCard>
    )
  }
  return (
    <ProblemCard
      label="Run not found"
      tone="info"
      title={`${id} is not on this desk`}
      cause="It may belong to another project folder, or it was removed."
      kept="Nothing was deleted by opening this link."
      next="Go back to the Desk, or start Span from the project folder that owns this run."
    >
      <Link to="/" className={buttonVariants()}>
        Back to Desk
      </Link>
    </ProblemCard>
  )
}
