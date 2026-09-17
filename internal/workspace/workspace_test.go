package workspace_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lazarok09/ahp-method/internal/render"
	"github.com/lazarok09/ahp-method/internal/workspace"
)

func TestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	ws := workspace.Open(dir)
	if err := ws.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	_ = ws.UpsertCriterion(workspace.Criterion{ID: "cost", Name: "Cost"})
	_ = ws.UpsertCriterion(workspace.Criterion{ID: "quality", Name: "Quality"})
	_ = ws.UpsertAlternative(workspace.Alternative{ID: "acme", Name: "Acme"})
	_ = ws.UpsertAlternative(workspace.Alternative{ID: "globex", Name: "Globex"})
	_ = ws.UpsertPairwise(workspace.PairwiseRow{
		Matrix: "criteria", Left: "cost", Right: "quality", Value: 3, Status: "proposal",
	})
	result, err := ws.Compute(false)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Warnings) == 0 {
		t.Fatal("expected proposal warning")
	}
	_ = ws.UpsertPairwise(workspace.PairwiseRow{
		Matrix: "criteria", Left: "cost", Right: "quality", Value: 3, Status: "committed",
	})
	_ = ws.UpsertPairwise(workspace.PairwiseRow{
		Matrix: "alt:cost", Left: "acme", Right: "globex", Value: 0.2, Status: "committed",
	})
	_ = ws.UpsertPairwise(workspace.PairwiseRow{
		Matrix: "alt:quality", Left: "acme", Right: "globex", Value: 3, Status: "committed",
	})
	result, err = ws.Compute(false)
	if err != nil {
		t.Fatal(err)
	}
	paths, err := ws.WriteOutputs(result, render.HTML(result))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(paths["report_html"]); err != nil {
		t.Fatal(err)
	}
	html, _ := os.ReadFile(paths["report_html"])
	if !contains(string(html), "How this method works") {
		t.Fatal("missing method section")
	}
	if !result.Complete {
		t.Fatal("expected complete")
	}
}

func TestCommitProposals(t *testing.T) {
	dir := t.TempDir()
	ws := workspace.Open(dir)
	_ = ws.EnsureLayout()
	_ = ws.UpsertCriterion(workspace.Criterion{ID: "cost", Name: "Cost"})
	_ = ws.UpsertCriterion(workspace.Criterion{ID: "quality", Name: "Quality"})
	_ = ws.UpsertPairwise(workspace.PairwiseRow{
		Matrix: "criteria", Left: "cost", Right: "quality", Value: 2, Status: "proposal",
	})
	updated, err := ws.CommitProposals("", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(updated) != 1 || updated[0].Status != "committed" {
		t.Fatalf("%+v", updated)
	}
}

func TestStatusMissing(t *testing.T) {
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
	if s.Complete {
		t.Fatal("expected incomplete")
	}
	if len(s.Missing) == 0 {
		t.Fatal("expected missing")
	}
	_ = filepath.Join(dir, "x")
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		(func() bool {
			for i := 0; i+len(sub) <= len(s); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		})())
}
