package workspace

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func seedCompleteWorkspace(t *testing.T, ws *Workspace) {
	t.Helper()
	if err := ws.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	_ = ws.SaveMeta(Meta{Title: "Journal fixture"})
	_ = ws.UpsertCriterion(Criterion{ID: "cost", Name: "Cost"})
	_ = ws.UpsertCriterion(Criterion{ID: "quality", Name: "Quality"})
	_ = ws.UpsertAlternative(Alternative{ID: "acme", Name: "Acme"})
	_ = ws.UpsertAlternative(Alternative{ID: "globex", Name: "Globex"})
	_ = ws.UpsertPairwise(PairwiseRow{
		Matrix: "criteria", Left: "cost", Right: "quality", Value: 3, Status: "committed",
	})
	_ = ws.UpsertPairwise(PairwiseRow{
		Matrix: "alt:cost", Left: "acme", Right: "globex", Value: 0.2, Status: "committed",
	})
	_ = ws.UpsertPairwise(PairwiseRow{
		Matrix: "alt:quality", Left: "acme", Right: "globex", Value: 3, Status: "committed",
	})
}

func TestJournalWriteListPrune(t *testing.T) {
	t.Setenv("AHP_JOURNAL", "")
	dir := t.TempDir()
	ws := Open(dir)
	seedCompleteWorkspace(t, ws)

	result, err := ws.Compute(false)
	if err != nil {
		t.Fatal(err)
	}
	on := true
	opts := WriteOutputsOpts{Journal: &on, Command: "compute", AHPVersion: "test"}
	paths, err := ws.WriteOutputs(result, "<html>ok</html>", opts)
	if err != nil {
		t.Fatal(err)
	}
	if paths["journal_id"] == "" || paths["journal_path"] == "" {
		t.Fatalf("expected journal paths, got %#v", paths)
	}
	if _, err := os.Stat(filepath.Join(paths["journal_path"], "results", "compute.json")); err != nil {
		t.Fatal(err)
	}
	outCompute, _ := os.ReadFile(paths["compute_json"])
	jCompute, _ := os.ReadFile(filepath.Join(paths["journal_path"], "results", "compute.json"))
	if !bytes.Equal(outCompute, jCompute) {
		t.Fatal("journal compute.json != output/compute.json")
	}
	// HTML off by default
	if _, err := os.Stat(filepath.Join(paths["journal_path"], "results", "report.html")); !os.IsNotExist(err) {
		t.Fatalf("report.html should be absent by default: %v", err)
	}

	// Second compute after a judgment change → second entry
	time.Sleep(10 * time.Millisecond)
	_ = ws.UpsertPairwise(PairwiseRow{
		Matrix: "alt:quality", Left: "acme", Right: "globex", Value: 5, Status: "committed",
	})
	result2, err := ws.Compute(false)
	if err != nil {
		t.Fatal(err)
	}
	paths2, err := ws.WriteOutputs(result2, "<html>ok2</html>", opts)
	if err != nil {
		t.Fatal(err)
	}
	if paths2["journal_id"] == paths["journal_id"] {
		t.Fatal("expected distinct journal ids")
	}
	list, err := ws.JournalList(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("list=%d want 2", len(list))
	}
	show, err := ws.JournalShow("latest")
	if err != nil {
		t.Fatal(err)
	}
	if show.Meta.ID != paths2["journal_id"] {
		t.Fatalf("latest=%s want %s", show.Meta.ID, paths2["journal_id"])
	}
	jp, err := ws.JournalPath(paths["journal_id"])
	if err != nil {
		t.Fatal(err)
	}
	if jp != paths["journal_path"] {
		t.Fatalf("path=%s want %s", jp, paths["journal_path"])
	}

	removed, err := ws.JournalPrune(1)
	if err != nil {
		t.Fatal(err)
	}
	if removed != 1 {
		t.Fatalf("removed=%d want 1", removed)
	}
	list, err = ws.JournalList(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ID != paths2["journal_id"] {
		t.Fatalf("after prune: %+v", list)
	}
}

func TestJournalDisabledByDefault(t *testing.T) {
	t.Setenv("AHP_JOURNAL", "")
	dir := t.TempDir()
	ws := Open(dir)
	seedCompleteWorkspace(t, ws)
	result, err := ws.Compute(false)
	if err != nil {
		t.Fatal(err)
	}
	paths, err := ws.WriteOutputs(result, "<html/>")
	if err != nil {
		t.Fatal(err)
	}
	if paths["journal_id"] != "" {
		t.Fatalf("journal should be off by default: %#v", paths)
	}
	if _, err := os.Stat(filepath.Join(dir, "tmp", "journal")); !os.IsNotExist(err) {
		t.Fatalf("tmp/journal should not exist: %v", err)
	}
}

func TestJournalEnvKillSwitch(t *testing.T) {
	t.Setenv("AHP_JOURNAL", "0")
	dir := t.TempDir()
	ws := Open(dir)
	seedCompleteWorkspace(t, ws)
	_ = os.WriteFile(ws.TomlPath(), []byte(`[decision]
title = "x"
description = ""

[journal]
enabled = true
`), 0o644)
	result, err := ws.Compute(false)
	if err != nil {
		t.Fatal(err)
	}
	on := true
	paths, err := ws.WriteOutputs(result, "<html/>", WriteOutputsOpts{Journal: &on})
	if err != nil {
		t.Fatal(err)
	}
	if paths["journal_id"] != "" {
		t.Fatal("AHP_JOURNAL=0 should kill journal even with --journal")
	}
}

func TestSaveMetaPreservesJournalSection(t *testing.T) {
	dir := t.TempDir()
	ws := Open(dir)
	_ = ws.EnsureLayout()
	body := `[decision]
title = "Old"
description = ""

[journal]
enabled = true
keep = 10
`
	if err := os.WriteFile(ws.TomlPath(), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ws.SaveMeta(Meta{Title: "New", Description: "d"}); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(ws.TomlPath())
	if !bytes.Contains(b, []byte("[journal]")) || !bytes.Contains(b, []byte("enabled = true")) {
		t.Fatalf("lost journal section: %s", b)
	}
	if !bytes.Contains(b, []byte(`title = "New"`)) {
		t.Fatalf("title not updated: %s", b)
	}
}
