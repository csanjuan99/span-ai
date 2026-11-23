package engine

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMetricsOfAnEmptyRootAreEmptyNotAnError(t *testing.T) {
	report := CollectMetrics(t.TempDir())
	if report.Runs.Total != 0 || len(report.Stages) != 0 || len(report.Agents) != 0 || len(report.Recent) != 0 {
		t.Fatalf("%+v", report)
	}
	if report.Stages == nil || report.Agents == nil || report.Recent == nil {
		t.Fatal("empty lists must be empty arrays for the page, not null")
	}
}

func TestMetricsCountEachRunByHowItEnded(t *testing.T) {
	opt, agents, _, _, _, _ := baseOpt(t)
	opt.Agents = modelAgents{agents}
	if _, err := Start(context.Background(), opt); err != nil {
		t.Fatal(err)
	}
	second, _, _, _, _, _ := baseOpt(t)
	second.Root = opt.Root
	second.Agents = modelAgents{&fakeAgents{frameErr: os.ErrInvalid}}
	if _, err := Start(context.Background(), second); err != nil {
		t.Fatal(err)
	}
	report := CollectMetrics(opt.Root)
	if report.Runs.Total != 2 {
		t.Fatalf("total %d", report.Runs.Total)
	}
	if report.Runs.Done+report.Runs.Stopped+report.Runs.Held+report.Runs.Running != report.Runs.Total {
		t.Fatalf("the outcomes do not add up: %+v", report.Runs)
	}
	if len(report.Recent) != 2 {
		t.Fatalf("recent %d", len(report.Recent))
	}
}

func TestMetricsGroupStagesAndTheAgentAndModelThatRanThem(t *testing.T) {
	opt, agents, _, _, _, _ := baseOpt(t)
	opt.Agents = modelAgents{agents}
	if _, err := Start(context.Background(), opt); err != nil {
		t.Fatal(err)
	}
	report := CollectMetrics(opt.Root)
	var frame *StageStat
	for index := range report.Stages {
		if report.Stages[index].Stage == "frame" {
			frame = &report.Stages[index]
		}
	}
	if frame == nil || frame.Runs != 1 || frame.Advance != 1 {
		t.Fatalf("frame stat %+v", frame)
	}
	var found bool
	for _, row := range report.Agents {
		if row.Agent == "cursor-agent" && row.Model == "model-of-cursor-agent" && row.Stages > 0 && row.Advance > 0 {
			found = true
		}
	}
	if !found {
		t.Fatalf("agents %+v", report.Agents)
	}
}

func TestAStageTakesFromTheLastSealToItsOwnSeal(t *testing.T) {
	start := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	got := stageSeconds(start, []time.Time{start.Add(30 * time.Second), start.Add(90 * time.Second)})
	if len(got) != 2 || got[0] != 30 || got[1] != 60 {
		t.Fatalf("%v", got)
	}
}

func TestPercentilesUseTheNearestRank(t *testing.T) {
	values := []float64{10, 20, 30, 40, 50, 60, 70, 80, 90, 100}
	if got := percentile(values, 50); got != 50 {
		t.Fatalf("median %v", got)
	}
	if got := percentile(values, 90); got != 90 {
		t.Fatalf("p90 %v", got)
	}
	if got := percentile(nil, 50); got != 0 {
		t.Fatalf("empty %v", got)
	}
}

func TestMetricsSkipAFolderThatIsNotARun(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "not-a-run"), 0o755); err != nil {
		t.Fatal(err)
	}
	if report := CollectMetrics(root); report.Runs.Total != 0 {
		t.Fatalf("%+v", report.Runs)
	}
}
