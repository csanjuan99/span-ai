package engine

import (
	"math"
	"os"
	"path/filepath"
	"sort"
	"time"

	"span/internal/journal"
	"span/internal/statement"
)

// recentRuns is how many runs the page lists, newest first.
const recentRuns = 20

// MetricRuns counts runs by how they stand.
type MetricRuns struct {
	Total   int `json:"total"`
	Done    int `json:"done"`
	Stopped int `json:"stopped"`
	Held    int `json:"held"`
	Running int `json:"running"`
	// FirstTry is the finished runs that needed no repair and no second iteration.
	FirstTry int `json:"firstTry"`
}

// StageStat is what the sealed records say about one stage across runs. Seconds run from the
// previous seal to this one, so they include time spent waiting on a person.
type StageStat struct {
	Stage   string  `json:"stage"`
	Runs    int     `json:"runs"`
	Advance int     `json:"advance"`
	Hold    int     `json:"hold"`
	Stop    int     `json:"stop"`
	Median  float64 `json:"medianSeconds"`
	P90     float64 `json:"p90Seconds"`
}

// AgentStat is one CLI and model and how the stages it ran ended. A run from before the
// journal recorded agents has no row here.
type AgentStat struct {
	Agent   string `json:"agent"`
	Model   string `json:"model"`
	Stages  int    `json:"stages"`
	Advance int    `json:"advance"`
	Hold    int    `json:"hold"`
	Stop    int    `json:"stop"`
}

// RunRow is one run in the list.
type RunRow struct {
	ID         string   `json:"id"`
	Label      string   `json:"label"`
	Stage      string   `json:"stage"`
	Verdict    string   `json:"verdict"`
	Iterations int      `json:"iterations"`
	Seconds    float64  `json:"seconds"`
	Agents     []string `json:"agents"`
}

// MetricsReport is everything the metrics page shows. Cost and tokens are not here: no run
// records them, and a figure nobody recorded would be made up.
type MetricsReport struct {
	Runs   MetricRuns  `json:"runs"`
	Stages []StageStat `json:"stages"`
	Agents []AgentStat `json:"agents"`
	Recent []RunRow    `json:"recent"`
}

// CollectMetrics reads every run under root. A folder that is not a readable run is skipped,
// as the list of runs does, so one damaged run never hides the others.
func CollectMetrics(root string) MetricsReport {
	report := MetricsReport{Stages: []StageStat{}, Agents: []AgentStat{}, Recent: []RunRow{}}
	entries, err := os.ReadDir(root)
	if err != nil {
		return report
	}
	stages := map[string]*StageStat{}
	seconds := map[string][]float64{}
	var stageOrder []string
	agents := map[string]*AgentStat{}
	var rows []RunRow
	for _, entry := range entries {
		if !entry.IsDir() || !validID(entry.Name()) {
			continue
		}
		snap, err := Load(root, entry.Name())
		if err != nil {
			continue
		}
		cur := snap.Cursor
		report.Runs.Total++
		verdict := verdictOf(cur)
		switch verdict {
		case "done":
			report.Runs.Done++
			if cur.Iteration <= 1 && len(cur.Repairs) == 0 && cur.InspectBlocks == 0 {
				report.Runs.FirstTry++
			}
		case "stop":
			report.Runs.Stopped++
		case "hold":
			report.Runs.Held++
		default:
			report.Runs.Running++
		}

		times := make([]time.Time, 0, len(snap.Statements))
		for _, item := range snap.Statements {
			times = append(times, item.Time)
		}
		took := stageSeconds(cur.StartedAt, times)
		for index, item := range snap.Statements {
			stat, ok := stages[item.Stage]
			if !ok {
				stat = &StageStat{Stage: item.Stage}
				stages[item.Stage] = stat
				stageOrder = append(stageOrder, item.Stage)
			}
			stat.Runs++
			switch item.Verdict {
			case statement.Advance:
				stat.Advance++
			case statement.Hold:
				stat.Hold++
			case statement.Stop:
				stat.Stop++
			}
			seconds[item.Stage] = append(seconds[item.Stage], took[index])
		}

		ran := tally(entryDir(root, entry.Name()), agents)
		total := 0.0
		for _, value := range took {
			total += value
		}
		rows = append(rows, RunRow{
			ID: cur.ID, Label: snap.Label, Stage: cur.Stage, Verdict: verdict,
			Iterations: cur.Iteration, Seconds: total, Agents: ran,
		})
	}
	for _, name := range stageOrder {
		stat := stages[name]
		stat.Median = percentile(seconds[name], 50)
		stat.P90 = percentile(seconds[name], 90)
		report.Stages = append(report.Stages, *stat)
	}
	for _, stat := range agents {
		report.Agents = append(report.Agents, *stat)
	}
	sort.Slice(report.Agents, func(i, j int) bool {
		a, b := report.Agents[i], report.Agents[j]
		if a.Stages != b.Stages {
			return a.Stages > b.Stages
		}
		return a.Agent+"/"+a.Model < b.Agent+"/"+b.Model
	})
	sort.Slice(rows, func(i, j int) bool { return rows[i].ID > rows[j].ID })
	if len(rows) > recentRuns {
		rows = rows[:recentRuns]
	}
	if rows != nil {
		report.Recent = rows
	}
	return report
}

func entryDir(root, id string) string { return filepath.Join(root, id) }

// tally reads one run's journal: each agent.run is matched with the next stage line of the
// same stage, which says how that stage ended. It adds to totals and returns the run's own
// "stage agent/model" list, each stage once.
func tally(dir string, totals map[string]*AgentStat) []string {
	events, err := journal.Read(dir)
	if err != nil {
		return []string{}
	}
	listed := []string{}
	seen := map[string]bool{}
	for index, event := range events {
		if event.Type != "agent.run" {
			continue
		}
		key := event.Detail["agent"] + "\x00" + event.Detail["model"]
		stat, ok := totals[key]
		if !ok {
			stat = &AgentStat{Agent: event.Detail["agent"], Model: event.Detail["model"]}
			totals[key] = stat
		}
		stat.Stages++
		for _, later := range events[index+1:] {
			if later.Type == "stage" && later.Detail["stage"] == event.Detail["stage"] {
				switch later.Detail["verdict"] {
				case statement.Advance:
					stat.Advance++
				case statement.Hold:
					stat.Hold++
				case statement.Stop:
					stat.Stop++
				}
				break
			}
		}
		label := event.Detail["stage"] + " " + event.Detail["agent"]
		if event.Detail["model"] != "" {
			label += "/" + event.Detail["model"]
		}
		if !seen[label] {
			seen[label] = true
			listed = append(listed, label)
		}
	}
	return listed
}

// stageSeconds is how long each seal took after the one before it, the first after start. A
// seal stamped before its predecessor counts zero rather than a negative time.
func stageSeconds(start time.Time, sealed []time.Time) []float64 {
	out := make([]float64, len(sealed))
	previous := start
	for index, at := range sealed {
		if !at.IsZero() && !previous.IsZero() && at.After(previous) {
			out[index] = at.Sub(previous).Seconds()
		}
		if !at.IsZero() {
			previous = at
		}
	}
	return out
}

// percentile is the nearest-rank percentile of values, 0 for none.
func percentile(values []float64, p float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	rank := int(math.Ceil(p / 100 * float64(len(sorted))))
	if rank < 1 {
		rank = 1
	}
	return sorted[rank-1]
}
