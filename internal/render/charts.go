package render

import (
	"fmt"
	"html"
	"math"
	"strings"
	"unicode/utf8"

	"github.com/lazarok09/ahp-method/internal/engine"
	"github.com/lazarok09/ahp-method/internal/workspace"
)

// Palette for alternative polygons on the profile radar.
var radarPalette = []string{
	"#1f6b55", "#8a4b12", "#3d5a80", "#b9483a", "#5c4d7a", "#2a9d8f",
}

// writeChoiceProfileRadar draws an SVG radar of each alternative's local
// weights across leaf criteria. Needs ≥3 leaf criteria with alt:* matrices.
func writeChoiceProfileRadar(b *strings.Builder, result *workspace.ComputeResult) {
	if result == nil || len(result.Ranking) == 0 {
		return
	}
	leaf, local, _, altNames, critNames := explainInputs(result)

	type axis struct {
		id, name string
	}
	var axes []axis
	for _, c := range result.Criteria {
		if _, ok := leaf[c.ID]; !ok {
			continue
		}
		if len(local[c.ID]) == 0 {
			continue
		}
		name := critNames[c.ID]
		if name == "" {
			name = c.ID
		}
		axes = append(axes, axis{id: c.ID, name: name})
	}
	if len(axes) < 3 {
		return
	}

	type series struct {
		id, name, color string
		values          []float64
	}
	var seriesList []series
	for i, r := range result.Ranking {
		vals := make([]float64, len(axes))
		okAny := false
		for j, ax := range axes {
			v := local[ax.id][r.ID]
			vals[j] = v
			if v > 0 {
				okAny = true
			}
		}
		if !okAny {
			continue
		}
		name := r.Name
		if name == "" {
			name = altNames[r.ID]
		}
		if name == "" {
			name = r.ID
		}
		seriesList = append(seriesList, series{
			id: r.ID, name: name, color: radarPalette[i%len(radarPalette)], values: vals,
		})
	}
	if len(seriesList) == 0 {
		return
	}

	// Wide canvas + room for wrapped axis labels outside the plot.
	const (
		vbW      = 720.0
		vbH      = 620.0
		cx       = 360.0
		cy       = 310.0
		radius   = 195.0
		labelGap = 52.0
		maxChars = 16 // wrap width per line (runes)
	)
	n := len(axes)

	b.WriteString(`<div class="chart-card" id="ranking-radar">
<p class="chart-kicker">Choice profile</p>
<p class="muted chart-lede">Radar of local weights per criterion (from each <code>alt:*</code> matrix). Farther from center = stronger on that criterion — not the global rank alone.</p>
`)
	fmt.Fprintf(b, `<svg class="radar" viewBox="0 0 %.0f %.0f" role="img" aria-label="Radar chart of alternatives across criteria">`, vbW, vbH)

	// Grid rings + spokes
	for ring := 1; ring <= 4; ring++ {
		rr := radius * float64(ring) / 4
		var pts []string
		for i := 0; i < n; i++ {
			x, y := polar(cx, cy, rr, angleFor(i, n))
			pts = append(pts, fmt.Sprintf("%.1f,%.1f", x, y))
		}
		fmt.Fprintf(b, `<polygon class="radar-grid" points="%s"/>`, strings.Join(pts, " "))
	}
	for i := 0; i < n; i++ {
		x, y := polar(cx, cy, radius, angleFor(i, n))
		fmt.Fprintf(b, `<line class="radar-spoke" x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f"/>`, cx, cy, x, y)
	}

	// Series polygons (draw weaker first so top ranks sit on top — reverse draw order)
	for i := len(seriesList) - 1; i >= 0; i-- {
		s := seriesList[i]
		var pts []string
		for j, v := range s.values {
			if v < 0 {
				v = 0
			}
			if v > 1 {
				v = 1
			}
			x, y := polar(cx, cy, radius*v, angleFor(j, n))
			pts = append(pts, fmt.Sprintf("%.1f,%.1f", x, y))
		}
		fmt.Fprintf(b,
			`<polygon class="radar-series" fill="%s" stroke="%s" points="%s"><title>%s</title></polygon>`,
			html.EscapeString(s.color)+"33", html.EscapeString(s.color), strings.Join(pts, " "), html.EscapeString(s.name),
		)
	}

	// Axis labels — wrap long criterion names instead of truncating.
	for i, ax := range axes {
		x, y := polar(cx, cy, radius+labelGap, angleFor(i, n))
		anchor := "middle"
		if x < cx-12 {
			anchor = "end"
		} else if x > cx+12 {
			anchor = "start"
		}
		lines := wrapLabel(ax.name, maxChars)
		fmt.Fprintf(b, `<text class="radar-label" x="%.1f" y="%.1f" text-anchor="%s">`, x, y, anchor)
		for li, line := range lines {
			dy := "0"
			if li > 0 {
				dy = "1.15em"
			}
			fmt.Fprintf(b, `<tspan x="%.1f" dy="%s">%s</tspan>`, x, dy, html.EscapeString(line))
		}
		b.WriteString(`</text>`)
	}

	b.WriteString(`</svg><ul class="radar-legend">`)
	for _, s := range seriesList {
		fmt.Fprintf(b, `<li><span class="swatch" style="background:%s"></span>%s</li>`,
			html.EscapeString(s.color), html.EscapeString(s.name))
	}
	b.WriteString(`</ul></div>`)
}

func angleFor(i, n int) float64 {
	return -math.Pi/2 + 2*math.Pi*float64(i)/float64(n)
}

func polar(cx, cy, r, ang float64) (float64, float64) {
	return cx + r*math.Cos(ang), cy + r*math.Sin(ang)
}

// wrapLabel breaks s into lines of at most maxChars runes, preferring spaces.
// Overlong tokens are hard-split so nothing is ellipsized away.
func wrapLabel(s string, maxChars int) []string {
	s = strings.TrimSpace(s)
	if s == "" || maxChars < 1 {
		return []string{s}
	}
	if utf8.RuneCountInString(s) <= maxChars {
		return []string{s}
	}
	words := strings.Fields(s)
	if len(words) == 0 {
		return []string{s}
	}
	var lines []string
	var cur strings.Builder
	curLen := 0
	flush := func() {
		if cur.Len() == 0 {
			return
		}
		lines = append(lines, cur.String())
		cur.Reset()
		curLen = 0
	}
	for _, w := range words {
		wLen := utf8.RuneCountInString(w)
		for wLen > maxChars {
			flush()
			runes := []rune(w)
			lines = append(lines, string(runes[:maxChars]))
			w = string(runes[maxChars:])
			wLen = utf8.RuneCountInString(w)
		}
		if wLen == 0 {
			continue
		}
		need := wLen
		if curLen > 0 {
			need++ // space
		}
		if curLen > 0 && curLen+need > maxChars {
			flush()
		}
		if curLen > 0 {
			cur.WriteByte(' ')
			curLen++
		}
		cur.WriteString(w)
		curLen += wLen
	}
	flush()
	if len(lines) == 0 {
		return []string{s}
	}
	return lines
}

// Criterion colors for stacked contribution bars (distinct from alternative palette).
var contribPalette = []string{
	"#1f6b55", "#3d5a80", "#8a4b12", "#5c4d7a", "#b9483a", "#2a9d8f",
	"#6b705c", "#bc6c25", "#457b9d", "#9b2226",
}

// writeContributionBars replaces a bare score curve with stacked bars:
// each row is a product; each segment is criterion_weight × local_weight.
// Bar length is absolute (vs top score) so #1 is longest and you can see why.
func writeContributionBars(b *strings.Builder, result *workspace.ComputeResult) {
	if result == nil || len(result.Ranking) == 0 || len(result.LeafWeights) == 0 {
		return
	}
	leaf, local, altIDs, altNames, critNames := explainInputs(result)
	if len(leaf) == 0 {
		return
	}
	exp := engine.Explain(leaf, local, altIDs, altNames, critNames)
	if len(exp.Alternatives) == 0 {
		return
	}

	type critMeta struct {
		id, name, color string
	}
	var crits []critMeta
	seen := map[string]bool{}
	for _, c := range result.Criteria {
		if _, ok := leaf[c.ID]; !ok || seen[c.ID] {
			continue
		}
		seen[c.ID] = true
		name := critNames[c.ID]
		if name == "" {
			name = c.Name
		}
		if name == "" {
			name = c.ID
		}
		crits = append(crits, critMeta{
			id: c.ID, name: name, color: contribPalette[len(crits)%len(contribPalette)],
		})
	}
	if len(crits) == 0 {
		return
	}

	maxW := 0.0
	for _, a := range exp.Alternatives {
		if a.GlobalWeight > maxW {
			maxW = a.GlobalWeight
		}
	}
	if maxW <= 0 {
		maxW = 1
	}

	// contrib lookup: alt → criterion → contribution
	byAlt := map[string]map[string]float64{}
	for _, a := range exp.Alternatives {
		m := map[string]float64{}
		for _, c := range a.Contributions {
			m[c.CriterionID] = c.Contribution
		}
		byAlt[a.ID] = m
	}

	b.WriteString(`<div class="chart-card" id="ranking-contrib">
<p class="chart-kicker">Why they rank</p>
<p class="muted chart-lede">Stacked contribution = criterion weight × local score. Longer bar = higher global weight. Hover a segment for the exact piece.</p>
<div class="contrib-legend">`)
	mid := (len(crits) + 1) / 2
	writeContribLegendCol := func(items []critMeta) {
		b.WriteString(`<ul class="contrib-legend-col">`)
		for _, c := range items {
			fmt.Fprintf(b, `<li><span class="swatch" style="background:%s"></span>%s</li>`,
				html.EscapeString(c.color), html.EscapeString(c.name))
		}
		b.WriteString(`</ul>`)
	}
	writeContribLegendCol(crits[:mid])
	if mid < len(crits) {
		writeContribLegendCol(crits[mid:])
	}
	b.WriteString(`</div><div class="contrib-chart" role="img" aria-label="Stacked contribution bars by alternative">`)

	for _, a := range exp.Alternatives {
		name := a.Name
		if name == "" {
			name = altNames[a.ID]
		}
		if name == "" {
			name = a.ID
		}
		fmt.Fprintf(b, `<div class="contrib-row">
<div class="contrib-meta"><span class="contrib-rank">#%d</span><span class="contrib-name">%s</span><span class="contrib-total">%.3f</span></div>
<div class="contrib-track">`, a.Rank, html.EscapeString(name), a.GlobalWeight)

		parts := byAlt[a.ID]
		for _, c := range crits {
			v := parts[c.id]
			if v <= 0 {
				continue
			}
			pct := 100 * v / maxW
			fmt.Fprintf(b,
				`<span class="contrib-seg" style="width:%.2f%%;background:%s" title="%s: %.4f (crit wt × local)"></span>`,
				pct, html.EscapeString(c.color), html.EscapeString(c.name), v,
			)
		}
		b.WriteString(`</div></div>`)
	}
	b.WriteString(`</div></div>`)
}
