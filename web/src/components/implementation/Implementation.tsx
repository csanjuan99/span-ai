/** Files changed from base to HEAD, the patch, and the agent transcript while build is open. */
export function Implementation({
  stat,
  patch,
  truncated,
  transcript,
}: {
  stat: string
  patch: string
  truncated: boolean
  transcript: string[]
}) {
  const files = stat
    .split("\n")
    .map((line) => line.trim())
    .filter((line) => line.includes("|"))
  return (
    <div className="flex flex-col gap-6">
      <section>
        <h2 className="text-balance text-sm">
          <span className="text-primary">$ </span>git diff --stat
        </h2>
        {files.length === 0 ? (
          <p className="mt-2 text-pretty text-muted-foreground">No commit on this run yet.</p>
        ) : (
          <ul className="mt-2 flex flex-col gap-1 text-sm tabular-nums">
            {files.map((file) => (
              <li key={file}>{file}</li>
            ))}
          </ul>
        )}
        {truncated ? <p className="mt-2 text-sm text-muted-foreground">The patch was cut at 256 KiB.</p> : null}
        {patch !== "" ? (
          <pre className="mt-3 overflow-auto border border-border bg-card p-3 text-xs">{patch}</pre>
        ) : null}
      </section>
      <section>
        <h2 className="text-balance text-sm">
          <span className="text-primary">$ </span>transcript
        </h2>
        {transcript.length === 0 ? (
          <p className="mt-2 text-pretty text-muted-foreground">The agent has not written yet.</p>
        ) : (
          <pre className="mt-2 overflow-auto border border-border bg-card p-3 text-xs">{transcript.join("\n")}</pre>
        )}
      </section>
    </div>
  )
}
