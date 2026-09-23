package workspace

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestGaussianFromAttributesWorkspace(t *testing.T) {
	dir := t.TempDir()
	ws := Open(dir)
	if err := ws.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	if err := ws.SaveMeta(Meta{Title: "Generic pick"}); err != nil {
		t.Fatal(err)
	}
	if err := ws.WriteCriteria([]Criterion{
		{ID: "quality", Name: "Quality"},
		{ID: "cost", Name: "Cost"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := ws.WriteAlternatives([]Alternative{
		{ID: "a", Name: "A"},
		{ID: "b", Name: "B"},
		{ID: "c", Name: "C"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := ws.WriteAttributes([]AttributeRow{
		{AlternativeID: "a", CriterionID: "quality", Value: "8"},
		{AlternativeID: "b", CriterionID: "quality", Value: "9"},
		{AlternativeID: "c", CriterionID: "quality", Value: "7"},
		{AlternativeID: "a", CriterionID: "cost", Value: "100", Unit: "BRL"},
		{AlternativeID: "b", CriterionID: "cost", Value: "120", Unit: "BRL"},
		{AlternativeID: "c", CriterionID: "cost", Value: "90", Unit: "BRL"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := ws.UpsertConstraint(Constraint{CriterionID: "quality", Prefer: "higher"}); err != nil {
		t.Fatal(err)
	}
	if err := ws.UpsertConstraint(Constraint{CriterionID: "cost", Prefer: "lower"}); err != nil {
		t.Fatal(err)
	}

	gres, err := ws.Gaussian()
	if err != nil {
		t.Fatal(err)
	}
	if len(gres.Ranking) != 3 {
		t.Fatalf("ranking=%v", gres.Ranking)
	}
	path, err := ws.WriteGaussianOutputs(gres)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(path) != "gaussian.json" {
		t.Fatalf("path=%s", path)
	}

	mf, err := ws.MethodFindings()
	if err != nil {
		t.Fatal(err)
	}
	foundHint := false
	for _, f := range mf {
		if f.Code == "method_ready_hint" {
			foundHint = true
		}
	}
	if !foundHint {
		t.Fatalf("expected method_ready_hint in %v", mf)
	}
}

func TestPreferOnlyConstraint(t *testing.T) {
	dir := t.TempDir()
	ws := Open(dir)
	if err := ws.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	if err := ws.UpsertConstraint(Constraint{CriterionID: "quality", Prefer: "higher"}); err != nil {
		t.Fatal(err)
	}
	items, err := ws.Constraints()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Prefer != "higher" {
		t.Fatalf("%v", items)
	}
}

func TestAbsoluteHybridSkippedWithoutCriteriaPairwise(t *testing.T) {
	dir := t.TempDir()
	ws := Open(dir)
	if err := ws.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	if err := ws.SaveMeta(Meta{Title: "Hybrid gate"}); err != nil {
		t.Fatal(err)
	}
	if err := ws.WriteCriteria([]Criterion{
		{ID: "quality", Name: "Quality"},
		{ID: "cost", Name: "Cost"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := ws.WriteAlternatives([]Alternative{
		{ID: "a", Name: "A"},
		{ID: "b", Name: "B"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := ws.WriteAttributes([]AttributeRow{
		{AlternativeID: "a", CriterionID: "quality", Value: "8"},
		{AlternativeID: "b", CriterionID: "quality", Value: "9"},
		{AlternativeID: "a", CriterionID: "cost", Value: "100"},
		{AlternativeID: "b", CriterionID: "cost", Value: "120"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := ws.UpsertConstraint(Constraint{CriterionID: "quality", Prefer: "higher"}); err != nil {
		t.Fatal(err)
	}
	if err := ws.UpsertConstraint(Constraint{CriterionID: "cost", Prefer: "lower"}); err != nil {
		t.Fatal(err)
	}

	ares, err := ws.Absolute(false)
	if err != nil {
		t.Fatal(err)
	}
	if ares.Method != "absolute" {
		t.Fatalf("expected matrix-only absolute, got %q ranking=%v warnings=%v", ares.Method, ares.Ranking, ares.Warnings)
	}
	if len(ares.Ranking) != 0 {
		t.Fatalf("expected no hybrid ranking, got %v", ares.Ranking)
	}

	if err := ws.AttachComparative(&ComputeResult{}, false, "hybrid"); err == nil {
		t.Fatal("expected hybrid attach to fail without criteria pairwise")
	}
}

func TestAbsoluteHybridWithCriteriaPairwise(t *testing.T) {
	dir := t.TempDir()
	ws := Open(dir)
	if err := ws.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	if err := ws.SaveMeta(Meta{Title: "Hybrid ok"}); err != nil {
		t.Fatal(err)
	}
	if err := ws.WriteCriteria([]Criterion{
		{ID: "quality", Name: "Quality"},
		{ID: "cost", Name: "Cost"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := ws.WriteAlternatives([]Alternative{
		{ID: "a", Name: "A"},
		{ID: "b", Name: "B"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := ws.WriteAttributes([]AttributeRow{
		{AlternativeID: "a", CriterionID: "quality", Value: "8"},
		{AlternativeID: "b", CriterionID: "quality", Value: "9"},
		{AlternativeID: "a", CriterionID: "cost", Value: "100"},
		{AlternativeID: "b", CriterionID: "cost", Value: "120"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := ws.UpsertConstraint(Constraint{CriterionID: "quality", Prefer: "higher"}); err != nil {
		t.Fatal(err)
	}
	if err := ws.UpsertConstraint(Constraint{CriterionID: "cost", Prefer: "lower"}); err != nil {
		t.Fatal(err)
	}
	if err := ws.WritePairwise([]PairwiseRow{
		{Matrix: "criteria", Left: "quality", Right: "cost", Value: 3, Status: "committed"},
	}); err != nil {
		t.Fatal(err)
	}

	ares, err := ws.Absolute(false)
	if err != nil {
		t.Fatal(err)
	}
	if ares.Method != "hybrid" {
		t.Fatalf("expected hybrid, got %q warnings=%v", ares.Method, ares.Warnings)
	}
	if len(ares.Ranking) != 2 {
		t.Fatalf("ranking=%v", ares.Ranking)
	}

	saaty, err := ws.Compute(false)
	if err != nil {
		t.Fatal(err)
	}
	// Incomplete alt pairwise must not block hybrid attach.
	if err := ws.AttachComparative(saaty, false, "hybrid"); err != nil {
		t.Fatal(err)
	}
	if saaty.Absolute == nil || saaty.Absolute.Method != "hybrid" {
		t.Fatalf("attach hybrid: %+v", saaty.Absolute)
	}
}

func TestAbsoluteHybridSkippedOnHotCriteriaCR(t *testing.T) {
	dir := t.TempDir()
	ws := Open(dir)
	if err := ws.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	if err := ws.SaveMeta(Meta{Title: "Hot CR"}); err != nil {
		t.Fatal(err)
	}
	if err := ws.WriteCriteria([]Criterion{
		{ID: "c1", Name: "C1"},
		{ID: "c2", Name: "C2"},
		{ID: "c3", Name: "C3"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := ws.WriteAlternatives([]Alternative{
		{ID: "a", Name: "A"},
		{ID: "b", Name: "B"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := ws.WriteAttributes([]AttributeRow{
		{AlternativeID: "a", CriterionID: "c1", Value: "1"},
		{AlternativeID: "b", CriterionID: "c1", Value: "2"},
		{AlternativeID: "a", CriterionID: "c2", Value: "3"},
		{AlternativeID: "b", CriterionID: "c2", Value: "4"},
		{AlternativeID: "a", CriterionID: "c3", Value: "5"},
		{AlternativeID: "b", CriterionID: "c3", Value: "6"},
	}); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"c1", "c2", "c3"} {
		if err := ws.UpsertConstraint(Constraint{CriterionID: id, Prefer: "higher"}); err != nil {
			t.Fatal(err)
		}
	}
	// Inconsistent triad: c1≫c2, c2≫c3, but c1≈c3 (should be ≫≫).
	if err := ws.WritePairwise([]PairwiseRow{
		{Matrix: "criteria", Left: "c1", Right: "c2", Value: 9, Status: "committed"},
		{Matrix: "criteria", Left: "c2", Right: "c3", Value: 9, Status: "committed"},
		{Matrix: "criteria", Left: "c1", Right: "c3", Value: 1, Status: "committed"},
	}); err != nil {
		t.Fatal(err)
	}

	cw, err := ws.CriteriaLeafWeights(false)
	if err != nil {
		t.Fatal(err)
	}
	ready, reason := criteriaMatricesReady(cw.Matrices)
	if ready {
		t.Fatalf("expected hot CR, matrices=%+v", cw.Matrices["criteria"])
	}
	if !strings.Contains(reason, "CR=") {
		t.Fatalf("reason=%q", reason)
	}

	ares, err := ws.Absolute(false)
	if err != nil {
		t.Fatal(err)
	}
	if ares.Method != "absolute" {
		t.Fatalf("expected matrix-only on hot CR, got %q warnings=%v", ares.Method, ares.Warnings)
	}
}

func TestAttachComparativeCompareSoftAbsolute(t *testing.T) {
	dir := t.TempDir()
	ws := Open(dir)
	if err := ws.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	if err := ws.SaveMeta(Meta{Title: "Compare soft"}); err != nil {
		t.Fatal(err)
	}
	if err := ws.WriteCriteria([]Criterion{{ID: "quality", Name: "Quality"}}); err != nil {
		t.Fatal(err)
	}
	if err := ws.WriteAlternatives([]Alternative{
		{ID: "a", Name: "A"},
		{ID: "b", Name: "B"},
	}); err != nil {
		t.Fatal(err)
	}
	// No attributes / prefer → absolute fails; gaussian also fails.
	result := &ComputeResult{Warnings: nil}
	err := ws.AttachComparative(result, false, "compare")
	if err == nil {
		t.Fatal("expected compare to fail when gaussian not ready")
	}
	if len(result.Warnings) == 0 || !strings.Contains(result.Warnings[0], "absolute:") {
		t.Fatalf("expected absolute soft-fail warning, got %v", result.Warnings)
	}
}
