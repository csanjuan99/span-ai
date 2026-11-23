import { useEffect, useState } from "react"
import { Link } from "react-router"

import { Shell } from "@/components/shell/Shell"
import { getJSON } from "@/lib/api"
import { barWidth, duration, loadFailure, metricsOf, percent, type Metrics as MetricsData } from "@/lib/metrics"
import { cn } from "@/lib/utils"

const tones: Record<string, string> = {
  done: "text-pass",
  stop: "text-destructive",
  hold: "text-hold",
  running: "text-primary",
}

function Heading({ children }: { children: string }) {
  return (
    <h2 className="text-balance text-sm">
      <span className="text-primary">$ </span>
      {children}
    </h2>
  )
}

function Note({ children }: { children: string }) {
  return <p className="text-pretty text-xs text-muted-foreground">{children}</p>
}

const head = "px-3 py-1.5 text-left text-xs font-normal text-muted-foreground"
const cell = "px-3 py-1.5 tabular-nums"

/** How the runs of this server went: time, outcome, and which agent and model did the work. */
export function Metrics() {
  const [data, setData] = useState<MetricsData | null>(null)
  const [note, setNote] = useState("")

  useEffect(() => {
    let closed = false
    void getJSON("/metrics")
      .then((body) => {
        if (!closed) {
          setData(metricsOf(body))
        }
      })
      .catch((err: unknown) => {
        if (closed) {
          return
        }
        setNote(loadFailure(err))
      })
    return () => {
      closed = true
    }
  }, [])

  const slowest = data === null ? 0 : Math.max(0, ...data.stages.map((stage) => stage.p90))

  return (
    <Shell path="~/span/metrics" status="metrics">
      <header className="flex flex-wrap items-baseline justify-between gap-2">
        <div>
          <h1 className="text-balance text-xl">
            <span className="text-primary">$ </span>span metrics
          </h1>
          <p className="text-pretty text-muted-foreground">How the runs on this server went.</p>
        </div>
        <Link className="text-sm text-primary underline-offset-4 hover:underline" to="/">
          Back to the desk
        </Link>
      </header>
      {note !== "" ? <p className="text-pretty text-sm text-destructive">{note}</p> : null}
      {data === null ? (
        note === "" ? <p className="text-muted-foreground">Reading runs…</p> : null
      ) : data.runs.total === 0 ? (
        <p className="text-pretty text-muted-foreground">No runs yet. Metrics appear after the first run on this server.</p>
      ) : (
        <>
          <section aria-label="Runs" className="flex flex-col gap-2">
            <Heading>runs</Heading>
            <p className="text-pretty">
              {data.runs.total} {data.runs.total === 1 ? "run" : "runs"}:{" "}
              <span className="text-pass">{data.runs.done} done</span> ({percent(data.runs.done, data.runs.total)}),{" "}
              <span className="text-destructive">{data.runs.stopped} stopped</span>,{" "}
              <span className="text-hold">{data.runs.held} held</span>, <span className="text-primary">{data.runs.running} running</span>.{" "}
              {data.runs.done > 0 ? (
                <>
                  {data.runs.firstTry} of {data.runs.done} finished on the first try ({percent(data.runs.firstTry, data.runs.done)}).
                </>
              ) : null}
            </p>
            <Note>Cost and tokens are not recorded by runs yet, so they are not shown.</Note>
          </section>
          <section aria-label="Stages" className="flex flex-col gap-2">
            <Heading>stages</Heading>
            <div className="overflow-x-auto border border-border">
              <table className="w-full min-w-[40rem] text-sm">
                <thead className="border-b border-border">
                  <tr>
                    <th className={head}>stage</th>
                    <th className={cn(head, "text-right")}>runs</th>
                    <th className={cn(head, "text-right")}>advance</th>
                    <th className={cn(head, "text-right")}>hold</th>
                    <th className={cn(head, "text-right")}>stop</th>
                    <th className={cn(head, "text-right")}>median</th>
                    <th className={head}>p90</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-border">
                  {data.stages.map((stage) => (
                    <tr key={stage.stage}>
                      <th scope="row" className={cn(cell, "text-left font-normal")}>
                        {stage.stage}
                      </th>
                      <td className={cn(cell, "text-right")}>{stage.runs}</td>
                      <td className={cn(cell, "text-right text-pass")}>{stage.advance}</td>
                      <td className={cn(cell, "text-right", stage.hold > 0 ? "text-hold" : "text-muted-foreground")}>{stage.hold}</td>
                      <td className={cn(cell, "text-right", stage.stop > 0 ? "text-destructive" : "text-muted-foreground")}>{stage.stop}</td>
                      <td className={cn(cell, "text-right")}>{duration(stage.median)}</td>
                      <td className={cell}>
                        <span className="flex items-center gap-2">
                          <span className="w-16 text-right">{duration(stage.p90)}</span>
                          <span aria-hidden className="h-1.5 w-24 bg-border">
                            <span className="block h-full bg-primary" style={{ width: `${barWidth(stage.p90, slowest)}%` }} />
                          </span>
                        </span>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            <Note>Time runs from the previous seal to this one, so it includes waiting on a person. p90 is the time 9 in 10 stages beat.</Note>
          </section>
          <section aria-label="Agents and models" className="flex flex-col gap-2">
            <Heading>agents and models</Heading>
            {data.agents.length === 0 ? (
              <p className="text-pretty text-muted-foreground">
                No run has recorded its agent yet. Runs started on this version do.
              </p>
            ) : (
              <div className="overflow-x-auto border border-border">
                <table className="w-full min-w-[34rem] text-sm">
                  <thead className="border-b border-border">
                    <tr>
                      <th className={head}>agent · model</th>
                      <th className={cn(head, "text-right")}>stages</th>
                      <th className={cn(head, "text-right")}>advance</th>
                      <th className={cn(head, "text-right")}>hold</th>
                      <th className={cn(head, "text-right")}>stop</th>
                      <th className={cn(head, "text-right")}>advance rate</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-border">
                    {data.agents.map((row) => (
                      <tr key={`${row.agent}|${row.model}`}>
                        <th scope="row" className={cn(cell, "text-left font-normal")}>
                          {row.agent}
                          {row.model === "" ? <span className="text-hold"> · no model chosen</span> : <span> · {row.model}</span>}
                        </th>
                        <td className={cn(cell, "text-right")}>{row.stages}</td>
                        <td className={cn(cell, "text-right text-pass")}>{row.advance}</td>
                        <td className={cn(cell, "text-right", row.hold > 0 ? "text-hold" : "text-muted-foreground")}>{row.hold}</td>
                        <td className={cn(cell, "text-right", row.stop > 0 ? "text-destructive" : "text-muted-foreground")}>{row.stop}</td>
                        <td className={cn(cell, "text-right")}>{percent(row.advance, row.advance + row.hold + row.stop)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </section>
          <section aria-label="Recent runs" className="flex flex-col gap-2">
            <Heading>recent runs</Heading>
            <div className="overflow-x-auto border border-border">
              <table className="w-full min-w-[40rem] text-sm">
                <thead className="border-b border-border">
                  <tr>
                    <th className={head}>run</th>
                    <th className={head}>stage</th>
                    <th className={head}>state</th>
                    <th className={cn(head, "text-right")}>iterations</th>
                    <th className={cn(head, "text-right")}>elapsed</th>
                    <th className={head}>ran with</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-border">
                  {data.recent.map((run) => (
                    <tr key={run.id}>
                      <td className={cell}>
                        <Link className="text-primary underline-offset-4 hover:underline" to={`/spans/${run.id}`} title={run.id}>
                          {run.label === "" ? run.id : run.label}
                        </Link>
                      </td>
                      <td className={cell}>{run.stage}</td>
                      <td className={cn(cell, tones[run.verdict] ?? "")}>{run.verdict}</td>
                      <td className={cn(cell, "text-right")}>{run.iterations}</td>
                      <td className={cn(cell, "text-right")}>{duration(run.seconds)}</td>
                      <td className="px-3 py-1.5 text-muted-foreground">{run.agents.length === 0 ? "—" : run.agents.join(", ")}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </section>
        </>
      )}
    </Shell>
  )
}
