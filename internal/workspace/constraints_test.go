package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lazarok09/ahp-method/internal/cliout"
)

func TestConstraintExcludesFromSynthesis(t *testing.T) {
	dir := t.TempDir()
	ws := Open(dir)
	if err := ws.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	_ = ws.SaveMeta(Meta{Title: "band"})
	_ = ws.UpsertCriterion(Criterion{ID: "value", Name: "Value"})
	_ = ws.UpsertCriterion(Criterion{ID: "fit", Name: "Fit"})
	_ = ws.UpsertAlternative(Alternative{ID: "cheap", Name: "Cheap"})
	_ = ws.UpsertAlternative(Alternative{ID: "mid", Name: "Mid"})
	_ = ws.UpsertAlternative(Alternative{ID: "dear", Name: "Dear"})
	_ = ws.UpsertAttribute(AttributeRow{AlternativeID: "cheap", CriterionID: "value", Value: "150", Unit: "BRL", Source: "shop"})
	_ = ws.UpsertAttribute(AttributeRow{AlternativeID: "mid", CriterionID: "value", Value: "300", Unit: "BRL", Source: "shop"})
	_ = ws.UpsertAttribute(AttributeRow{AlternativeID: "dear", CriterionID: "value", Value: "500", Unit: "BRL", Source: "shop"})

	min, max := 200.0, 400.0
	if err := ws.UpsertConstraint(Constraint{
		CriterionID: "value", Min: &min, Max: &max, Unit: "BRL", Prefer: "lower",
	}); err != nil {
		t.Fatal(err)
	}

	// Full pairwise among all three so matrices would be complete without filter.
	for _, m := range []string{"criteria", "alt:value", "alt:fit"} {
		ids := []string{"cheap", "mid", "dear"}
		if m == "criteria" {
			ids = []string{"value", "fit"}
		}
		for i := 0; i < len(ids); i++ {
			for j := i + 1; j < len(ids); j++ {
				_ = ws.UpsertPairwise(PairwiseRow{
					Matrix: m, Left: ids[i], Right: ids[j], Value: 1, Status: "committed",
				})
			}
		}
	}

	res, err := ws.Compute(false)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.ExcludedAlternatives) == 0 {
		t.Fatal("expected exclusions")
	}
	for _, r := range res.Ranking {
		if r.ID == "cheap" || r.ID == "dear" {
			t.Fatalf("out-of-band %s still ranked", r.ID)
		}
	}
	if len(res.Ranking) != 1 {
		// only mid remains — synthesis with 1 alt is degenerate but should not include excluded
		t.Logf("ranking=%v excluded=%v warnings=%v", res.Ranking, res.ExcludedAlternatives, res.Warnings)
	}
	if res.Complete {
		t.Fatal("complete should be false with <2 eligible alts")
	}

	rep, err := ws.Doctor(DoctorOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !hasFindingCode(rep, "constraint_violation") {
		t.Fatalf("doctor should flag constraint_violation, got %#v", rep.Findings)
	}
	if rep.ExitCode == cliout.ExitOK {
		t.Fatal("doctor should not be OK with constraint violations")
	}
}

func TestPurchaseIntegrityFXSmell(t *testing.T) {
	dir := t.TempDir()
	ws := Open(dir)
	_ = ws.EnsureLayout()
	_ = ws.SaveMeta(Meta{Title: "fx"})
	_ = ws.UpsertCriterion(Criterion{ID: "value", Name: "Value"})
	_ = ws.UpsertAlternative(Alternative{ID: "a", Name: "A"})
	_ = ws.UpsertAlternative(Alternative{ID: "b", Name: "B"})
	_ = ws.UpsertAttribute(AttributeRow{
		AlternativeID: "a", CriterionID: "value", Value: "300", Unit: "BRL",
		Source: "Amazon.com $59 converted", Note: "bad",
	})
	_ = ws.UpsertAttribute(AttributeRow{
		AlternativeID: "b", CriterionID: "value", Value: "320", Unit: "BRL",
		Source: "King Musical boleto",
	})

	rep, err := ws.Doctor(DoctorOptions{Purchase: true})
	if err != nil {
		t.Fatal(err)
	}
	if !hasFindingCode(rep, "fx_or_foreign_source") {
		t.Fatalf("expected fx_or_foreign_source, got %#v", rep.Findings)
	}
}

func TestImportPairwiseCSV(t *testing.T) {
	dir := t.TempDir()
	ws := Open(dir)
	_ = ws.EnsureLayout()
	_ = ws.UpsertCriterion(Criterion{ID: "c", Name: "C"})
	_ = ws.UpsertCriterion(Criterion{ID: "d", Name: "D"})
	path := filepath.Join(dir, "pairs.csv")
	body := "matrix,left,right,value,status,note\ncriteria,c,d,3,,from import\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := ws.ImportPairwiseCSV(path, "proposal")
	if err != nil {
		t.Fatal(err)
	}
	if res.Written != 1 {
		t.Fatalf("written=%d", res.Written)
	}
	pairs, _ := ws.Pairwise()
	if len(pairs) != 1 || pairs[0].Status != "proposal" || pairs[0].Value != 3 {
		t.Fatalf("got %+v", pairs)
	}
}

func TestMustHaveConstraintExcludesFalsy(t *testing.T) {
	dir := t.TempDir()
	ws := Open(dir)
	_ = ws.EnsureLayout()
	_ = ws.SaveMeta(Meta{Title: "park"})
	_ = ws.UpsertCriterion(Criterion{ID: "parking", Name: "Parking"})
	_ = ws.UpsertCriterion(Criterion{ID: "value", Name: "Value"})
	_ = ws.UpsertAlternative(Alternative{ID: "with", Name: "With park"})
	_ = ws.UpsertAlternative(Alternative{ID: "without", Name: "No park"})
	_ = ws.UpsertAttribute(AttributeRow{AlternativeID: "with", CriterionID: "parking", Value: "1", Unit: "bool", Source: "Booking"})
	_ = ws.UpsertAttribute(AttributeRow{AlternativeID: "without", CriterionID: "parking", Value: "0", Unit: "bool", Source: "Booking"})
	_ = ws.UpsertAttribute(AttributeRow{AlternativeID: "with", CriterionID: "value", Value: "300", Unit: "BRL", Source: "Booking"})
	_ = ws.UpsertAttribute(AttributeRow{AlternativeID: "without", CriterionID: "value", Value: "280", Unit: "BRL", Source: "Booking"})
	if err := ws.UpsertConstraint(Constraint{CriterionID: "parking", MustHave: true}); err != nil {
		t.Fatal(err)
	}
	ok, viol, err := ws.EligibleAlternativeIDs()
	if err != nil {
		t.Fatal(err)
	}
	if len(ok) != 1 || ok[0] != "with" {
		t.Fatalf("eligible=%v viol=%v", ok, viol)
	}
	found := false
	for _, v := range viol {
		if v.AlternativeID == "without" && v.CriterionID == "parking" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected without parking violation, got %#v", viol)
	}
}

func TestRemoveAlternativeCleansPairsAndAttrs(t *testing.T) {
	dir := t.TempDir()
	ws := Open(dir)
	_ = ws.EnsureLayout()
	_ = ws.UpsertCriterion(Criterion{ID: "value", Name: "Value"})
	_ = ws.UpsertAlternative(Alternative{ID: "keep", Name: "Keep"})
	_ = ws.UpsertAlternative(Alternative{ID: "drop", Name: "Drop"})
	_ = ws.UpsertAttribute(AttributeRow{AlternativeID: "keep", CriterionID: "value", Value: "1"})
	_ = ws.UpsertAttribute(AttributeRow{AlternativeID: "drop", CriterionID: "value", Value: "2"})
	_ = ws.UpsertPairwise(PairwiseRow{Matrix: "alt:value", Left: "drop", Right: "keep", Value: 2, Status: "committed"})
	res, err := ws.RemoveAlternative("drop")
	if err != nil {
		t.Fatal(err)
	}
	if res.RemovedAttributes < 1 || res.RemovedPairs < 1 {
		t.Fatalf("cleanup counts: %+v", res)
	}
	alts, _ := ws.Alternatives()
	if len(alts) != 1 || alts[0].ID != "keep" {
		t.Fatalf("alts=%v", alts)
	}
	attrs, _ := ws.Attributes()
	for _, a := range attrs {
		if a.AlternativeID == "drop" {
			t.Fatal("attribute for drop still present")
		}
	}
	pairs, _ := ws.Pairwise()
	for _, p := range pairs {
		if p.Left == "drop" || p.Right == "drop" {
			t.Fatalf("pair still references drop: %+v", p)
		}
	}
}

func TestDoctorAttrsUnrated(t *testing.T) {
	dir := t.TempDir()
	ws := Open(dir)
	_ = ws.EnsureLayout()
	_ = ws.SaveMeta(Meta{Title: "unrated"})
	_ = ws.UpsertCriterion(Criterion{ID: "value", Name: "Value"})
	_ = ws.UpsertAlternative(Alternative{ID: "a", Name: "A"})
	_ = ws.UpsertAlternative(Alternative{ID: "b", Name: "B"})
	_ = ws.UpsertAttribute(AttributeRow{AlternativeID: "a", CriterionID: "value", Value: "100", Unit: "BRL", Source: "shop"})
	_ = ws.UpsertAttribute(AttributeRow{AlternativeID: "b", CriterionID: "value", Value: "200", Unit: "BRL", Source: "shop"})
	// no pairwise on alt:value
	rep, err := ws.Doctor(DoctorOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !hasFindingCode(rep, "attrs_unrated") {
		t.Fatalf("expected attrs_unrated, got %#v", rep.Findings)
	}
}

func TestPickQuotePrefersLowerLocalSource(t *testing.T) {
	dir := t.TempDir()
	ws := Open(dir)
	_ = ws.EnsureLayout()
	_ = ws.UpsertCriterion(Criterion{ID: "value", Name: "Value"})
	_ = ws.UpsertAlternative(Alternative{ID: "swell", Name: "Swell"})
	_ = ws.UpsertQuote(QuoteRow{
		AlternativeID: "swell", CriterionID: "value", Value: "600", Unit: "BRL",
		Source: "Booking.com", URL: "https://www.booking.com/hotel/br/swell-praia.pt-br.html",
	})
	_ = ws.UpsertQuote(QuoteRow{
		AlternativeID: "swell", CriterionID: "value", Value: "353", Unit: "BRL",
		Source: "Decolar Pix", URL: "https://www.decolar.com/hoteis/h-7319753/swell-praia-hotel-natal",
	})
	res, err := ws.PickQuote("value", "lower")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Picked) != 1 || res.Picked[0].Value != "353" || res.Picked[0].Source != "Decolar Pix" {
		t.Fatalf("picked=%+v", res.Picked)
	}
	attrs, _ := ws.Attributes()
	if len(attrs) != 1 || attrs[0].Value != "353" {
		t.Fatalf("attrs=%+v", attrs)
	}
	if !strings.Contains(attrs[0].Note, "Booking.com=600") {
		t.Fatalf("note should mention discarded Booking quote: %s", attrs[0].Note)
	}
}

func hasFindingCode(rep *DoctorReport, code string) bool {
	for _, f := range rep.Findings {
		if f.Code == code {
			return true
		}
	}
	return false
}
