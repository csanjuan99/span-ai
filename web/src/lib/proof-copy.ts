import type { Proof } from "@/lib/proof"

/** How many stages a run has; the sealed count reads against it. */
const stageCount = 10

/** Where the run is, as the proof panel words it. */
export interface RunPosition {
  kind: "running" | "held" | "paused" | "blocked" | "busy" | "approval" | "done" | "unsealed"
  stage: string
  sealed: number
}

/** A proof with nothing measured yet, for a run whose diff is not known. */
export function emptyProof(): Proof {
  return {
    verdict: { level: "low", headline: "" },
    diff: { files: 0, added: 0, removed: 0, kinds: "" },
    checks: { passed: 0, total: 0, failing: "", inspect: "not yet", tests: null },
  }
}

const capital = (word: string) => word.charAt(0).toUpperCase() + word.slice(1)

/** The verdict cell: where the run stands, with a caption that only states what the ledger shows. */
export function verdictCopy(run: RunPosition, proof: Proof): { title: string; caption: string } {
  const { checks } = proof
  switch (run.kind) {
    case "running":
      return { title: `${capital(run.stage)} running`, caption: `sealed stages: ${run.sealed} of ${stageCount}` }
    case "paused":
      return { title: `${capital(run.stage)} paused`, caption: `${run.stage} is not sealed yet` }
    case "blocked":
      return {
        title: `Held at ${run.stage}`,
        caption: checks.total === 0 ? "Agent could not run · no check ran" : "Agent could not run",
      }
    case "busy":
      return { title: `Held at ${run.stage}`, caption: "Waiting for the checkout · no check ran" }
    case "approval":
      return {
        title: `Ready to ${run.stage}`,
        caption: [
          `${checks.passed} of ${checks.total} checks green`,
          ...(checks.inspect === "passed" ? ["inspect found nothing blocking"] : []),
        ].join(" · "),
      }
    case "unsealed":
      return { title: "Not sealed", caption: "no proof results to show" }
    case "done":
      return { title: "Shipped", caption: `${run.sealed} of ${stageCount} stages sealed` }
    case "held": {
      const red = checks.total - checks.passed
      return {
        title: `Held at ${run.stage}`,
        caption:
          checks.total > 0 && red > 0
            ? `${red} of ${checks.total} checks not green`
            : checks.total > 0
              ? `everything before ${run.stage} passed`
              : `${run.sealed} of ${stageCount} stages sealed`,
      }
    }
  }
}

/** The diff cell's label and caption when the diff is empty or still being written. */
export function diffCopy(run: RunPosition, proof: Proof): { label: string; caption: string } {
  if (proof.diff.files > 0) {
    return run.kind === "running"
      ? { label: "Diff so far", caption: `${proof.diff.kinds} · live` }
      : { label: "Diff", caption: proof.diff.kinds }
  }
  const caption =
    run.stage === "intake" || run.stage === "frame"
      ? `${run.stage} does not touch the repo`
      : run.kind === "busy"
        ? `${run.stage} has not started`
        : run.kind === "running"
          ? `${run.stage} is writing`
          : `${run.stage} has not written anything`
  return { label: "Diff", caption }
}

const beforeProve = ["intake", "frame", "build"]

/**
 * The tests cell before any check ran, or null once the checks have something to say. It names
 * when the tests will run so "Not run" is never a bare absence.
 */
export function testsCopy(run: RunPosition, proof: Proof): { value: string; caption: string } | null {
  if (run.kind === "unsealed") {
    return { value: "Not run", caption: "build never sealed" }
  }
  if (proof.checks.total > 0 || proof.checks.tests !== null || !beforeProve.includes(run.stage)) {
    return null
  }
  const waiting = run.kind === "running" || run.kind === "paused"
  return {
    value: waiting ? "Not run yet" : "Not run",
    caption: run.stage === "build" && run.kind === "running" ? "prove runs right after build" : `prove comes after ${run.stage}`,
  }
}
