package render

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/lazarok09/ahp-method/internal/workspace"
)

func TestSlug(t *testing.T) {
	cases := map[string]string{
		"alt:value":     "alt-value",
		"criteria":      "criteria",
		"Criteria:Cost": "criteria-cost",
		"  Foo Bar  ":   "foo-bar",
	}
	for in, want := range cases {
		if got := slug(in); got != want {
			t.Fatalf("slug(%q)=%q want %q", in, got, want)
		}
	}
}

func TestCellClass(t *testing.T) {
	if cellClass(1, 0, 0) != "diag" {
		t.Fatal("diag")
	}
	if cellClass(1, 0, 1) != "eq" {
		t.Fatal("eq")
	}
	if cellClass(5, 0, 1) != "hi s5" {
		t.Fatal("hi")
	}
	if cellClass(0.2, 0, 1) != "lo s5" {
		t.Fatal("lo")
	}
}

func TestDescribeMatrix(t *testing.T) {
	crit := []workspace.Criterion{
		{ID: "value", Name: "Preço BR (Pix)"},
		{ID: "panel", Name: "Qualidade do painel OLED"},
	}
	m := describeMatrix("alt:value", crit)
	if m.Title != "Preço BR (Pix)" || m.Badge != "Alternatives" || m.Kind != "alternatives" {
		t.Fatalf("alt: %#v", m)
	}
	if strings.Contains(m.Title, "alt:") {
		t.Fatal("title should not contain alt:")
	}
	c := describeMatrix("criteria", crit)
	if c.Title != "Criteria importance" || c.Kind != "criteria" {
		t.Fatalf("criteria: %#v", c)
	}
	s := describeMatrix("criteria:panel", crit)
	if !strings.Contains(s.Title, "Qualidade") || s.Kind != "subcriteria" {
		t.Fatalf("sub: %#v", s)
	}
}

func TestSortedMatrixKeysOrder(t *testing.T) {
	keys := sortedMatrixKeys(map[string]workspace.MatrixPayload{
		"alt:value": {}, "criteria": {}, "alt:panel": {}, "criteria:x": {},
	})
	want := []string{"criteria", "criteria:x", "alt:panel", "alt:value"}
	if len(keys) != len(want) {
		t.Fatalf("%v", keys)
	}
	for i := range want {
		if keys[i] != want[i] {
			t.Fatalf("got %v want %v", keys, want)
		}
	}
}

func TestHTMLTOCFlashAndTitles(t *testing.T) {
	htmlOut := HTML(&workspace.ComputeResult{
		Title: "Sample decision",
		Matrices: map[string]workspace.MatrixPayload{
			"criteria":  {IDs: []string{"a", "b"}, Names: map[string]string{"a": "A", "b": "B"}, Weights: map[string]float64{"a": 0.5, "b": 0.5}},
			"alt:value": {IDs: []string{"x"}, Names: map[string]string{"x": "X"}, Weights: map[string]float64{"x": 1}},
		},
		Criteria: []workspace.Criterion{
			{ID: "value", Name: "Price"},
		},
		Ranking:     []workspace.RankRow{{Rank: 1, ID: "x", Name: "X", Weight: 1}},
		LeafWeights: map[string]float64{"value": 1},
	})

	for _, want := range []string{
		"IBM Plex Sans",
		"tabular-nums",
		"toc-border-blink",
		`nav.toc a[href^="#"]`,
		`setAttribute("data-flash"`,
		`>How this method works</a>`,
		`>Global ranking (Saaty)</a>`,
		`>Contribution breakdown</a>`,
		`>Weights by matrix</a>`,
		`>Consistency repairs</a>`,
		`>Pairwise comparisons</a>`,
		`>Workspace data (CSV)</a>`,
		`href="#matrix-alt-value" class="toc-sub">Price</a>`,
		`id="method"`,
		`id="ranking"`,
		`id="explain"`,
		`id="weights"`,
		`id="repairs"`,
		`id="matrices"`,
		`id="data"`,
		`id="matrix-alt-value"`,
	} {
		if !strings.Contains(htmlOut, want) {
			t.Fatalf("missing %q in report HTML", want)
		}
	}
	if strings.Contains(htmlOut, `>Method</a>`) && !strings.Contains(htmlOut, `>How this method works</a>`) {
		t.Fatal("TOC still uses short Method label instead of section title")
	}
}

func TestChoiceProfileRadar(t *testing.T) {
	result := &workspace.ComputeResult{
		Title: "Vendor pick",
		Criteria: []workspace.Criterion{
			{ID: "cost", Name: "Cost"},
			{ID: "quality", Name: "Quality"},
			{ID: "risk", Name: "Risk"},
		},
		Alternatives: []workspace.Alternative{
			{ID: "acme", Name: "Acme"},
			{ID: "globex", Name: "Globex"},
		},
		LeafWeights: map[string]float64{"cost": 0.5, "quality": 0.3, "risk": 0.2},
		Matrices: map[string]workspace.MatrixPayload{
			"alt:cost":    {Weights: map[string]float64{"acme": 0.7, "globex": 0.3}},
			"alt:quality": {Weights: map[string]float64{"acme": 0.2, "globex": 0.8}},
			"alt:risk":    {Weights: map[string]float64{"acme": 0.4, "globex": 0.6}},
		},
		Ranking: []workspace.RankRow{
			{Rank: 1, ID: "acme", Name: "Acme", Weight: 0.55},
			{Rank: 2, ID: "globex", Name: "Globex", Weight: 0.45},
		},
	}
	htmlOut := HTML(result)
	for _, want := range []string{
		`id="ranking-radar"`,
		`Choice profile`,
		`<svg class="radar"`,
		`radar-series`,
		`>Acme</li>`,
		`>Globex</li>`,
		`>Cost</tspan>`,
		`>Quality</tspan>`,
		`>Risk</tspan>`,
	} {
		if !strings.Contains(htmlOut, want) {
			t.Fatalf("missing %q in radar HTML", want)
		}
	}

	// Too few axes → no radar
	thin := HTML(&workspace.ComputeResult{
		Criteria:    []workspace.Criterion{{ID: "cost", Name: "Cost"}, {ID: "quality", Name: "Quality"}},
		LeafWeights: map[string]float64{"cost": 0.6, "quality": 0.4},
		Matrices: map[string]workspace.MatrixPayload{
			"alt:cost":    {Weights: map[string]float64{"acme": 1}},
			"alt:quality": {Weights: map[string]float64{"acme": 1}},
		},
		Ranking: []workspace.RankRow{{Rank: 1, ID: "acme", Name: "Acme", Weight: 1}},
	})
	if strings.Contains(thin, `id="ranking-radar"`) {
		t.Fatal("radar should not render with fewer than 3 criteria")
	}
}

func TestContributionBars(t *testing.T) {
	htmlOut := HTML(&workspace.ComputeResult{
		Criteria: []workspace.Criterion{
			{ID: "cost", Name: "Cost"},
			{ID: "quality", Name: "Quality"},
			{ID: "risk", Name: "Risk"},
		},
		Alternatives: []workspace.Alternative{
			{ID: "acme", Name: "Acme Parse"},
			{ID: "globex", Name: "Globex Cloud"},
		},
		LeafWeights: map[string]float64{"cost": 0.5, "quality": 0.3, "risk": 0.2},
		Matrices: map[string]workspace.MatrixPayload{
			"alt:cost":    {Weights: map[string]float64{"acme": 0.7, "globex": 0.3}},
			"alt:quality": {Weights: map[string]float64{"acme": 0.2, "globex": 0.8}},
			"alt:risk":    {Weights: map[string]float64{"acme": 0.4, "globex": 0.6}},
		},
		Ranking: []workspace.RankRow{
			{Rank: 1, ID: "acme", Name: "Acme Parse", Weight: 0.55},
			{Rank: 2, ID: "globex", Name: "Globex Cloud", Weight: 0.45},
		},
	})
	for _, want := range []string{
		`id="ranking-contrib"`,
		`Why they rank`,
		`contrib-seg`,
		`contrib-legend-col`,
		`Acme Parse`,
		`Globex Cloud`,
		`Cost`,
		`Quality`,
		`Risk`,
	} {
		if !strings.Contains(htmlOut, want) {
			t.Fatalf("missing %q in contribution bars HTML", want)
		}
	}
	if strings.Contains(htmlOut, `id="ranking-curve"`) {
		t.Fatal("score curve should be gone")
	}
}

func TestMatrixDialog(t *testing.T) {
	htmlOut := HTML(&workspace.ComputeResult{
		Title: "Dialog check",
		Criteria: []workspace.Criterion{
			{ID: "cost", Name: "Cost"},
			{ID: "quality", Name: "Quality"},
		},
		Matrices: map[string]workspace.MatrixPayload{
			"criteria": {
				IDs:     []string{"cost", "quality"},
				Names:   map[string]string{"cost": "Cost", "quality": "Quality"},
				Weights: map[string]float64{"cost": 0.6, "quality": 0.4},
				Matrix:  [][]float64{{1, 2}, {0.5, 1}},
				Complete: true,
			},
		},
	})
	for _, want := range []string{
		`command="show-modal"`,
		`commandfor="matrix-criteria-dialog"`,
		`<dialog id="matrix-criteria-dialog" class="matrix-dialog"`,
		`method="dialog"`,
		`class="matrix-expand"`,
		`>Expand</button>`,
		`>Close</button>`,
		`width:80vw`,
		`max-width:80%`,
	} {
		if !strings.Contains(htmlOut, want) {
			t.Fatalf("missing %q in matrix dialog HTML", want)
		}
	}
}

func TestWrapLabel(t *testing.T) {
	got := wrapLabel("Qualidade do painel OLED", 16)
	if len(got) < 2 {
		t.Fatalf("expected wrap, got %#v", got)
	}
	joined := strings.Join(got, " ")
	if !strings.Contains(joined, "Qualidade") || !strings.Contains(joined, "OLED") {
		t.Fatalf("lost words: %#v", got)
	}
	for _, line := range got {
		if utf8.RuneCountInString(line) > 16 {
			t.Fatalf("line too long %q", line)
		}
	}
	if one := wrapLabel("Preço BR (Pix)", 16); len(one) != 1 {
		t.Fatalf("short label should stay one line: %#v", one)
	}
}
