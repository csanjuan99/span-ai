package main

import (
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	"span/internal/engine"
)

const metricsUsage = "usage: ./span metrics [--json] [--spans .span/runs]   (exit 0 ok, 2 usage)"

func metricsCmd(args []string) int { return metricsMain(args, os.Stdout, os.Stderr) }

// metricsMain is the metrics page from a terminal: the same report GET /metrics answers, read
// from the same state directory with the same code. A folder with no runs is an empty report.
func metricsMain(args []string, out, errw io.Writer) int {
	fs, v := verbSet("metrics", errw, false)
	pos, ok := parseVerb(fs, args)
	if !ok || len(pos) != 0 {
		fmt.Fprintln(errw, metricsUsage)
		return 2
	}
	report := engine.CollectMetrics(v.spans)
	if v.asJSON {
		return emit(out, errw, report)
	}
	fmt.Fprintf(out, "%d runs: done %d, stopped %d, held %d, running %d; %d finished on the first try\n",
		report.Runs.Total, report.Runs.Done, report.Runs.Stopped, report.Runs.Held, report.Runs.Running, report.Runs.FirstTry)
	table := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
	fmt.Fprintln(table, "STAGE\tRUNS\tADVANCE\tHOLD\tSTOP\tMEDIAN\tP90")
	for _, stage := range report.Stages {
		fmt.Fprintf(table, "%s\t%d\t%d\t%d\t%d\t%.0fs\t%.0fs\n", stage.Stage, stage.Runs, stage.Advance, stage.Hold, stage.Stop, stage.Median, stage.P90)
	}
	if err := table.Flush(); err != nil {
		fmt.Fprintln(errw, err)
		return 1
	}
	return 0
}
