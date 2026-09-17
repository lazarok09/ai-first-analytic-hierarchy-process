package workspace_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lazarok09/ahp-method/internal/cliout"
	"github.com/lazarok09/ahp-method/internal/workspace"
)

func TestRecommendNextIncomplete(t *testing.T) {
	s := &workspace.StatusSummary{
		Criteria:     3,
		Alternatives: 2,
		Complete:     false,
		Missing: []workspace.MissingPair{
			{Matrix: "criteria", Left: "a", Right: "b"},
			{Matrix: "criteria", Left: "a", Right: "c"},
		},
	}
	next := workspace.RecommendNext(s)
	if next.Kind != "incomplete" || next.Command == "" {
		t.Fatalf("got %+v", next)
	}
	if workspace.ReadinessExit(s) != cliout.ExitIncomplete {
		t.Fatalf("exit=%d", workspace.ReadinessExit(s))
	}
}

func TestRecommendNextProposals(t *testing.T) {
	s := &workspace.StatusSummary{
		Criteria:          2,
		Alternatives:      2,
		Complete:          true,
		Consistent:        true,
		PairwiseProposals: 2,
	}
	next := workspace.RecommendNext(s)
	if next.Kind != "proposals" || next.Command != "ahp plan" {
		t.Fatalf("got %+v", next)
	}
	if workspace.ReadinessExit(s) != cliout.ExitProposals {
		t.Fatalf("exit=%d", workspace.ReadinessExit(s))
	}
}

func TestRecommendNextInconsistent(t *testing.T) {
	s := &workspace.StatusSummary{
		Criteria:     3,
		Alternatives: 2,
		Complete:     true,
		Consistent:   false,
		Repairs: []workspace.RepairRow{
			{Matrix: "criteria", Left: "a", Right: "b", Current: 9, Suggested: 3},
		},
	}
	next := workspace.RecommendNext(s)
	if next.Kind != "inconsistent" || next.Command == "" {
		t.Fatalf("got %+v", next)
	}
	if workspace.ReadinessExit(s) != cliout.ExitInconsistent {
		t.Fatalf("exit=%d", workspace.ReadinessExit(s))
	}
}

func TestRecommendNextReadyCompute(t *testing.T) {
	s := &workspace.StatusSummary{
		Criteria:     2,
		Alternatives: 2,
		Complete:     true,
		Consistent:   true,
		ReportHTML:   filepath.Join(t.TempDir(), "missing-report.html"),
	}
	next := workspace.RecommendNext(s)
	if next.Kind != "ready" || next.Command != "ahp compute" {
		t.Fatalf("got %+v", next)
	}
	if workspace.ReadinessExit(s) != cliout.ExitOK {
		t.Fatalf("exit=%d", workspace.ReadinessExit(s))
	}
}

func TestRecommendNextReadyOpen(t *testing.T) {
	dir := t.TempDir()
	report := filepath.Join(dir, "report.html")
	if err := os.WriteFile(report, []byte("<html/>"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := &workspace.StatusSummary{
		Criteria:     2,
		Alternatives: 2,
		Complete:     true,
		Consistent:   true,
		ReportHTML:   report,
	}
	next := workspace.RecommendNext(s)
	if next.Kind != "ready" || next.Command != "ahp open" {
		t.Fatalf("got %+v", next)
	}
}

func TestStatusAttachesNext(t *testing.T) {
	dir := t.TempDir()
	ws := workspace.Open(dir)
	_ = ws.EnsureLayout()
	_ = ws.UpsertCriterion(workspace.Criterion{ID: "a", Name: "A"})
	_ = ws.UpsertCriterion(workspace.Criterion{ID: "b", Name: "B"})
	_ = ws.UpsertCriterion(workspace.Criterion{ID: "c", Name: "C"})
	s, err := ws.Status(false)
	if err != nil {
		t.Fatal(err)
	}
	if s.Next.Command == "" || s.Next.Kind == "" {
		t.Fatalf("next=%+v", s.Next)
	}
	if s.ExitCode != cliout.ExitIncomplete {
		t.Fatalf("exit_code=%d", s.ExitCode)
	}
}
