package workspace_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lazarok09/ahp-method/internal/workspace"
)

func TestTreeVendorSelectionSmoke(t *testing.T) {
	root := filepath.Join("..", "..", "examples", "vendor-selection")
	if _, err := os.Stat(filepath.Join(root, "ahp.toml")); err != nil {
		t.Skip("vendor-selection example not present")
	}
	ws := workspace.Open(root)
	tree, err := ws.Tree()
	if err != nil {
		t.Fatal(err)
	}
	out := workspace.FormatTree(tree)
	if strings.TrimSpace(out) == "" {
		t.Fatal("empty tree")
	}
	if !strings.Contains(out, "cost") || !strings.Contains(out, "alt:cost") {
		t.Fatalf("expected criteria and alt matrices:\n%s", out)
	}
	if !strings.Contains(out, "acme") {
		t.Fatalf("expected alternatives:\n%s", out)
	}
	if len(tree.Criteria) == 0 || len(tree.Alternatives) == 0 {
		t.Fatalf("tree json empty: %+v", tree)
	}
}

func TestRecommendNextPriorityOrder(t *testing.T) {
	// Explicit priority chain: structure → incomplete → inconsistent → proposals → ready.
	cases := []struct {
		name string
		s    workspace.StatusSummary
		kind string
		cmd  string
	}{
		{"structure", workspace.StatusSummary{Criteria: 0}, "structure", "ahp add-criterion"},
		{"incomplete", workspace.StatusSummary{
			Criteria: 2, Alternatives: 2, Complete: false,
			Missing: []workspace.MissingPair{{Matrix: "criteria", Left: "a", Right: "b"}},
			Consistent: false, PairwiseProposals: 9,
		}, "incomplete", "ahp set-pairwise"},
		{"inconsistent", workspace.StatusSummary{
			Criteria: 2, Alternatives: 2, Complete: true, Consistent: false, PairwiseProposals: 9,
		}, "inconsistent", "ahp validate"},
		{"proposals", workspace.StatusSummary{
			Criteria: 2, Alternatives: 2, Complete: true, Consistent: true, PairwiseProposals: 2,
		}, "proposals", "ahp commit-proposals"},
		{"ready", workspace.StatusSummary{
			Criteria: 2, Alternatives: 2, Complete: true, Consistent: true,
			ReportHTML: filepath.Join(t.TempDir(), "gone.html"),
		}, "ready", "ahp compute"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := workspace.RecommendNext(&tt.s)
			if got.Kind != tt.kind || got.Command != tt.cmd {
				t.Fatalf("got kind=%q cmd=%q want kind=%q cmd=%q", got.Kind, got.Command, tt.kind, tt.cmd)
			}
		})
	}
}
