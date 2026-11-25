"use client"

import { useEffect, useState } from "react"
import { Link } from "react-router"

import {
  Cell,
  DataTable,
  nonZero,
  RowHead,
} from "@/components/data-table/DataTable"
import { Shell } from "@/components/shell/Shell"
import { getJSON } from "@/lib/api"
import {
  barWidth,
  duration,
  loadFailure,
  metricsOf,
  percent,
  type Metrics as MetricsData,
} from "@/lib/metrics"

const tones: Record<string, string> = {
  done: "text-pass",
  stop: "text-destructive",
  hold: "text-hold",
  running: "text-primary",
}

function Heading({ children }: { children: string }) {
  return (
    <h2 className="text-sm text-balance">
      <span className="text-primary">$ </span>
      {children}
    </h2>
  )
}

function Note({ children }: { children: string }) {
  return <p className="text-xs text-pretty text-muted-foreground">{children}</p>
}

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

  const slowest =
    data === null ? 0 : Math.max(0, ...data.stages.map((stage) => stage.p90))

  return (
    <Shell path="~/span/metrics" status="metrics">
      <header className="flex flex-wrap items-baseline justify-between gap-2">
        <div>
          <h1 className="text-xl text-balance">
            <span className="text-primary">$ </span>span metrics
          </h1>
          <p className="text-pretty text-muted-foreground">
            How the runs on this server went.
          </p>
        </div>
        <Link
          className="text-sm text-primary underline-offset-4 hover:underline max-sm:inline-flex max-sm:min-h-11 max-sm:items-center"
          to="/"
        >
          Back to the desk
        </Link>
      </header>
      {note !== "" ? (
        <p className="text-sm text-pretty text-destructive">{note}</p>
      ) : null}
      {data === null ? (
        note === "" ? (
          <p className="text-muted-foreground">Reading runs…</p>
        ) : null
      ) : data.runs.total === 0 ? (
        <p className="text-pretty text-muted-foreground">
          No runs yet. Metrics appear after the first run on this server.
        </p>
      ) : (
        <>
          <section aria-label="Runs" className="flex flex-col gap-2">
            <Heading>runs</Heading>
            <p className="text-pretty">
              {data.runs.total} {data.runs.total === 1 ? "run" : "runs"}:{" "}
              <span className="text-pass">{data.runs.done} done</span> (
              {percent(data.runs.done, data.runs.total)}),{" "}
              <span className="text-destructive">
                {data.runs.stopped} stopped
              </span>
              , <span className="text-hold">{data.runs.held} held</span>,{" "}
              <span className="text-primary">{data.runs.running} running</span>.{" "}
              {data.runs.done > 0 ? (
                <>
                  {data.runs.firstTry} of {data.runs.done} finished on the first
                  try ({percent(data.runs.firstTry, data.runs.done)}).
                </>
              ) : null}
            </p>
            <Note>
              Cost and tokens are not recorded by runs yet, so they are not
              shown.
            </Note>
          </section>
          <section aria-label="Stages" className="flex flex-col gap-2">
            <Heading>stages</Heading>
            <DataTable
              minWidth="min-w-[40rem]"
              columns={[
                { label: "stage" },
                { label: "runs", right: true },
                { label: "advance", right: true },
                { label: "hold", right: true },
                { label: "stop", right: true },
                { label: "median", right: true },
                { label: "p90" },
              ]}
            >
              {data.stages.map((stage) => (
                <tr key={stage.stage}>
                  <RowHead>{stage.stage}</RowHead>
                  <Cell right>{stage.runs}</Cell>
                  <Cell right className="text-pass">
                    {stage.advance}
                  </Cell>
                  <Cell right className={nonZero(stage.hold, "text-hold")}>
                    {stage.hold}
                  </Cell>
                  <Cell
                    right
                    className={nonZero(stage.stop, "text-destructive")}
                  >
                    {stage.stop}
                  </Cell>
                  <Cell right>{duration(stage.median)}</Cell>
                  <Cell>
                    <span className="flex items-center gap-2">
                      <span className="w-16 text-right">
                        {duration(stage.p90)}
                      </span>
                      <span aria-hidden className="h-1.5 w-24 bg-border">
                        <span
                          className="block h-full bg-primary"
                          style={{ width: `${barWidth(stage.p90, slowest)}%` }}
                        />
                      </span>
                    </span>
                  </Cell>
                </tr>
              ))}
            </DataTable>
            <Note>
              Time runs from the previous seal to this one, so it includes
              waiting on a person. p90 is the time 9 in 10 stages beat.
            </Note>
          </section>
          <section
            aria-label="Agents and models"
            className="flex flex-col gap-2"
          >
            <Heading>agents and models</Heading>
            {data.agents.length === 0 ? (
              <p className="text-pretty text-muted-foreground">
                No run has recorded its agent yet. Runs started on this version
                do.
              </p>
            ) : (
              <DataTable
                minWidth="min-w-[34rem]"
                columns={[
                  { label: "agent · model" },
                  { label: "stages", right: true },
                  { label: "advance", right: true },
                  { label: "hold", right: true },
                  { label: "stop", right: true },
                  { label: "advance rate", right: true },
                ]}
              >
                {data.agents.map((row) => (
                  <tr key={`${row.agent}|${row.model}`}>
                    <RowHead>
                      {row.agent}
                      {row.model === "" ? (
                        <span className="text-hold"> · no model chosen</span>
                      ) : (
                        <span> · {row.model}</span>
                      )}
                    </RowHead>
                    <Cell right>{row.stages}</Cell>
                    <Cell right className="text-pass">
                      {row.advance}
                    </Cell>
                    <Cell right className={nonZero(row.hold, "text-hold")}>
                      {row.hold}
                    </Cell>
                    <Cell
                      right
                      className={nonZero(row.stop, "text-destructive")}
                    >
                      {row.stop}
                    </Cell>
                    <Cell right>
                      {percent(row.advance, row.advance + row.hold + row.stop)}
                    </Cell>
                  </tr>
                ))}
              </DataTable>
            )}
          </section>
          <section aria-label="Recent runs" className="flex flex-col gap-2">
            <Heading>recent runs</Heading>
            <DataTable
              minWidth="min-w-[40rem]"
              columns={[
                { label: "run" },
                { label: "stage" },
                { label: "state" },
                { label: "iterations", right: true },
                { label: "elapsed", right: true },
                { label: "ran with" },
              ]}
            >
              {data.recent.map((run) => (
                <tr key={run.id}>
                  <Cell>
                    <Link
                      className="text-primary underline-offset-4 hover:underline max-sm:inline-flex max-sm:min-h-11 max-sm:items-center"
                      to={`/spans/${run.id}`}
                      title={run.id}
                    >
                      {run.label === "" ? run.id : run.label}
                    </Link>
                  </Cell>
                  <Cell>{run.stage}</Cell>
                  <Cell className={tones[run.verdict] ?? ""}>
                    {run.verdict}
                  </Cell>
                  <Cell right>{run.iterations}</Cell>
                  <Cell right>{duration(run.seconds)}</Cell>
                  <Cell className="text-muted-foreground">
                    {run.agents.length === 0 ? "—" : run.agents.join(", ")}
                  </Cell>
                </tr>
              ))}
            </DataTable>
          </section>
        </>
      )}
    </Shell>
  )
}
