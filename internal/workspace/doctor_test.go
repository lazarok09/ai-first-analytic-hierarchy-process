package workspace_test

import (
	"testing"

	"github.com/lazarok09/ahp-method/internal/cliout"
	"github.com/lazarok09/ahp-method/internal/workspace"
)

func TestDoctorExitCodes(t *testing.T) {
	tests := []struct {
		name         string
		strict       bool
		include      bool
		setup        func(t *testing.T, ws *workspace.Workspace)
		wantCode     int
		wantCodeName string
	}{
		{
			name:         "incomplete_missing_pairs",
			wantCode:     cliout.ExitIncomplete,
			wantCodeName: "incomplete",
			setup: func(t *testing.T, ws *workspace.Workspace) {
				t.Helper()
				must(t, ws.EnsureLayout())
				must(t, ws.UpsertCriterion(workspace.Criterion{ID: "a", Name: "A"}))
				must(t, ws.UpsertCriterion(workspace.Criterion{ID: "b", Name: "B"}))
				must(t, ws.UpsertCriterion(workspace.Criterion{ID: "c", Name: "C"}))
				must(t, ws.UpsertAlternative(workspace.Alternative{ID: "x", Name: "X"}))
				must(t, ws.UpsertAlternative(workspace.Alternative{ID: "y", Name: "Y"}))
				// only one of three criteria pairs
				must(t, ws.UpsertPairwise(workspace.PairwiseRow{
					Matrix: "criteria", Left: "a", Right: "b", Value: 2, Status: "committed",
				}))
			},
		},
		{
			name:         "incomplete_unknown_pairwise_id",
			wantCode:     cliout.ExitIncomplete,
			wantCodeName: "incomplete",
			setup: func(t *testing.T, ws *workspace.Workspace) {
				t.Helper()
				must(t, ws.EnsureLayout())
				must(t, ws.UpsertCriterion(workspace.Criterion{ID: "a", Name: "A"}))
				must(t, ws.UpsertCriterion(workspace.Criterion{ID: "b", Name: "B"}))
				must(t, ws.UpsertAlternative(workspace.Alternative{ID: "x", Name: "X"}))
				must(t, ws.UpsertAlternative(workspace.Alternative{ID: "y", Name: "Y"}))
				must(t, ws.UpsertPairwise(workspace.PairwiseRow{
					Matrix: "criteria", Left: "a", Right: "ghost", Value: 3, Status: "committed",
				}))
			},
		},
		{
			name:         "inconsistent_cr",
			wantCode:     cliout.ExitInconsistent,
			wantCodeName: "inconsistent",
			setup: func(t *testing.T, ws *workspace.Workspace) {
				t.Helper()
				must(t, ws.EnsureLayout())
				must(t, ws.UpsertCriterion(workspace.Criterion{ID: "a", Name: "A"}))
				must(t, ws.UpsertCriterion(workspace.Criterion{ID: "b", Name: "B"}))
				must(t, ws.UpsertCriterion(workspace.Criterion{ID: "c", Name: "C"}))
				must(t, ws.UpsertAlternative(workspace.Alternative{ID: "x", Name: "X"}))
				must(t, ws.UpsertAlternative(workspace.Alternative{ID: "y", Name: "Y"}))
				// Classic inconsistent 3×3: a>b, b>c, but c>>a
				must(t, ws.UpsertPairwise(workspace.PairwiseRow{
					Matrix: "criteria", Left: "a", Right: "b", Value: 9, Status: "committed",
				}))
				must(t, ws.UpsertPairwise(workspace.PairwiseRow{
					Matrix: "criteria", Left: "b", Right: "c", Value: 9, Status: "committed",
				}))
				must(t, ws.UpsertPairwise(workspace.PairwiseRow{
					Matrix: "criteria", Left: "a", Right: "c", Value: 1.0 / 9.0, Status: "committed",
				}))
				must(t, ws.UpsertPairwise(workspace.PairwiseRow{
					Matrix: "alt:a", Left: "x", Right: "y", Value: 1, Status: "committed",
				}))
				must(t, ws.UpsertPairwise(workspace.PairwiseRow{
					Matrix: "alt:b", Left: "x", Right: "y", Value: 1, Status: "committed",
				}))
				must(t, ws.UpsertPairwise(workspace.PairwiseRow{
					Matrix: "alt:c", Left: "x", Right: "y", Value: 1, Status: "committed",
				}))
			},
		},
		{
			name:         "ok_complete_consistent",
			wantCode:     cliout.ExitOK,
			wantCodeName: "ok",
			setup: func(t *testing.T, ws *workspace.Workspace) {
				t.Helper()
				seedReady(t, ws, false)
			},
		},
		{
			name:         "proposals_info_not_strict",
			strict:       false,
			include:      true,
			wantCode:     cliout.ExitOK,
			wantCodeName: "ok",
			setup: func(t *testing.T, ws *workspace.Workspace) {
				t.Helper()
				seedReady(t, ws, true)
			},
		},
		{
			name:         "proposals_strict_exit4",
			strict:       true,
			include:      true,
			wantCode:     cliout.ExitProposals,
			wantCodeName: "proposals",
			setup: func(t *testing.T, ws *workspace.Workspace) {
				t.Helper()
				seedReady(t, ws, true)
			},
		},
		{
			name:         "incomplete_beats_proposals_strict",
			strict:       true,
			wantCode:     cliout.ExitIncomplete,
			wantCodeName: "incomplete",
			setup: func(t *testing.T, ws *workspace.Workspace) {
				t.Helper()
				must(t, ws.EnsureLayout())
				must(t, ws.UpsertCriterion(workspace.Criterion{ID: "a", Name: "A"}))
				must(t, ws.UpsertCriterion(workspace.Criterion{ID: "b", Name: "B"}))
				must(t, ws.UpsertAlternative(workspace.Alternative{ID: "x", Name: "X"}))
				must(t, ws.UpsertAlternative(workspace.Alternative{ID: "y", Name: "Y"}))
				must(t, ws.UpsertPairwise(workspace.PairwiseRow{
					Matrix: "criteria", Left: "a", Right: "b", Value: 2, Status: "proposal",
				}))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			ws := workspace.Open(dir)
			tt.setup(t, ws)

			report, err := ws.Doctor(workspace.DoctorOptions{
				Strict:           tt.strict,
				IncludeProposals: tt.include,
			})
			if err != nil {
				t.Fatal(err)
			}
			if report.ExitCode != tt.wantCode {
				t.Fatalf("exit=%d (%s findings=%v), want %d (%s)",
					report.ExitCode, codesOf(report), summarizeFindings(report), tt.wantCode, tt.wantCodeName)
			}
			if tt.wantCode == cliout.ExitOK && !report.OK {
				t.Fatal("expected OK=true")
			}
			if tt.wantCode != cliout.ExitOK && report.OK {
				t.Fatal("expected OK=false")
			}
		})
	}
}

func TestDoctorReportsFixCommands(t *testing.T) {
	dir := t.TempDir()
	ws := workspace.Open(dir)
	must(t, ws.EnsureLayout())
	must(t, ws.UpsertCriterion(workspace.Criterion{ID: "a", Name: "A"}))
	must(t, ws.UpsertCriterion(workspace.Criterion{ID: "b", Name: "B"}))
	must(t, ws.UpsertAlternative(workspace.Alternative{ID: "x", Name: "X"}))
	must(t, ws.UpsertAlternative(workspace.Alternative{ID: "y", Name: "Y"}))

	report, err := ws.ValidateGraph(false)
	if err != nil {
		t.Fatal(err)
	}
	if report.ExitCode != cliout.ExitIncomplete {
		t.Fatalf("exit=%d", report.ExitCode)
	}
	foundFix := false
	for _, f := range report.Findings {
		if f.Code == "empty_matrix" || f.Code == "missing_pair" {
			if f.Fix == "" || !containsStr(f.Fix, "pair set") {
				t.Fatalf("expected pair set fix, got %+v", f)
			}
			foundFix = true
		}
	}
	if !foundFix {
		t.Fatalf("expected missing/empty finding, got %+v", report.Findings)
	}
}

func seedReady(t *testing.T, ws *workspace.Workspace, proposal bool) {
	t.Helper()
	must(t, ws.EnsureLayout())
	must(t, ws.UpsertCriterion(workspace.Criterion{ID: "cost", Name: "Cost"}))
	must(t, ws.UpsertCriterion(workspace.Criterion{ID: "quality", Name: "Quality"}))
	must(t, ws.UpsertAlternative(workspace.Alternative{ID: "acme", Name: "Acme"}))
	must(t, ws.UpsertAlternative(workspace.Alternative{ID: "globex", Name: "Globex"}))
	status := "committed"
	if proposal {
		status = "proposal"
	}
	must(t, ws.UpsertPairwise(workspace.PairwiseRow{
		Matrix: "criteria", Left: "cost", Right: "quality", Value: 3, Status: status,
	}))
	must(t, ws.UpsertPairwise(workspace.PairwiseRow{
		Matrix: "alt:cost", Left: "acme", Right: "globex", Value: 2, Status: "committed",
	}))
	must(t, ws.UpsertPairwise(workspace.PairwiseRow{
		Matrix: "alt:quality", Left: "acme", Right: "globex", Value: 1.0 / 3.0, Status: "committed",
	}))
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func codesOf(r *workspace.DoctorReport) []string {
	out := make([]string, 0, len(r.Findings))
	for _, f := range r.Findings {
		out = append(out, f.Code)
	}
	return out
}

func summarizeFindings(r *workspace.DoctorReport) string {
	if len(r.Findings) == 0 {
		return "none"
	}
	s := r.Findings[0].Message
	if len(r.Findings) > 1 {
		s += "…"
	}
	return s
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		func() bool {
			for i := 0; i+len(sub) <= len(s); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		}())
}
