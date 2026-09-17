package render

import (
	"fmt"
	"html"
	"math"
	"strings"

	"github.com/lazarok09/ahp-method/internal/engine"
	"github.com/lazarok09/ahp-method/internal/workspace"
)

func HTML(result *workspace.ComputeResult) string {
	var b strings.Builder
	b.WriteString(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8"/>
<meta name="viewport" content="width=device-width, initial-scale=1"/>
<title>`)
	b.WriteString(html.EscapeString(result.Title))
	b.WriteString(` — AHP report</title>
<style>
:root{--bg:#f6f4ef;--ink:#1c1916;--muted:#5c564c;--line:#d9d2c5;--card:#fffdf8;--accent:#245c4a;--warn:#8a4b12;--bad:#8f1d1d;--ok:#1f6b45}
*{box-sizing:border-box}body{margin:0;font:16px/1.5 "Iowan Old Style","Palatino Linotype",Palatino,Georgia,serif;color:var(--ink);background:var(--bg)}
header,main{max-width:960px;margin:0 auto;padding:2rem 1.25rem}header{padding-bottom:0}
h1{font-size:2rem;margin:0 0 .4rem}h2{font-size:1.25rem;margin:2.2rem 0 .7rem;border-bottom:1px solid var(--line);padding-bottom:.3rem}
h3{font-size:1.05rem;margin:1.4rem 0 .5rem}.muted{color:var(--muted)}.lede{font-size:1.05rem;max-width:42em}
nav.toc a{color:var(--accent);margin-right:1rem;text-decoration:none}
.status{display:flex;gap:.6rem;flex-wrap:wrap;margin:1rem 0 0}
.pill{display:inline-block;padding:.15rem .55rem;border:1px solid var(--line);border-radius:999px;font-size:.85rem;background:var(--card)}
.pill.ok{color:var(--ok)}.pill.bad{color:var(--bad)}.pill.warn{color:var(--warn)}
.card{background:var(--card);border:1px solid var(--line);padding:1rem 1.1rem;margin:.8rem 0}
table{width:100%;border-collapse:collapse;font-variant-numeric:tabular-nums;background:var(--card)}
th,td{border:1px solid var(--line);padding:.4rem .55rem;text-align:left}th{font-size:.85rem}
td.num,th.num{text-align:right}.bar{height:10px;background:#ece7dc}.bar>span{display:block;height:100%;background:var(--accent)}
.warn-list{color:var(--warn)}footer{max-width:960px;margin:0 auto;padding:0 1.25rem 3rem;color:var(--muted);font-size:.9rem}
</style></head><body><header>
<p class="muted">Analytic Hierarchy Process — local workspace report</p>
<h1>`)
	b.WriteString(html.EscapeString(result.Title))
	b.WriteString(`</h1>`)
	if result.Description != "" {
		b.WriteString(`<p class="lede">` + html.EscapeString(result.Description) + `</p>`)
	}
	compClass, compLabel := "warn", "Incomplete pairwise"
	if result.Complete {
		compClass, compLabel = "ok", "Matrices complete"
	}
	consClass, consLabel := "bad", "Inconsistent or incomplete"
	if result.Consistent {
		consClass, consLabel = "ok", "CR ≤ 0.10"
	}
	fmt.Fprintf(&b, `<div class="status"><span class="pill %s">%s</span><span class="pill %s">%s</span>`, compClass, compLabel, consClass, consLabel)
	if result.IncludeProposals {
		b.WriteString(`<span class="pill warn">Includes AI proposals</span>`)
	}
	b.WriteString(`</div>
<nav class="toc" style="margin-top:1rem">
<a href="#method">Method</a><a href="#ranking">Ranking</a><a href="#weights">Weights</a>
<a href="#repairs">CR repairs</a><a href="#matrices">Matrices</a><a href="#data">CSV data</a>
</nav></header><main>`)

	if len(result.Warnings) > 0 {
		b.WriteString(`<div class="card warn-list"><strong>Warnings</strong><ul>`)
		for _, w := range result.Warnings {
			b.WriteString(`<li>` + html.EscapeString(w) + `</li>`)
		}
		b.WriteString(`</ul></div>`)
	}

	b.WriteString(`<section id="method"><h2>How this method works</h2>
<p>AHP turns a decision into a hierarchy (goal → criteria → alternatives), then asks for pairwise Saaty judgments (1 = equal, 9 = extreme preference). Each complete reciprocal matrix has a principal eigenvector: those entries are the local weights. Consistency ratio (CR) flags whether the judgments contradict each other; CR ≤ 0.10 is the usual acceptance cut.</p>
<p>Global alternative scores are the weighted sum of local scores across leaf criteria. Objective facts live in <code>data/attributes.csv</code> so they can inform human (or proposed) judgments without becoming the ranking themselves.</p>
<p class="muted">Edit CSVs, then re-run <code>ahp compute</code>. Proposal rows are ignored until status is <code>committed</code>.</p>
</section><section id="ranking"><h2>Global ranking</h2>`)

	if len(result.Ranking) == 0 {
		b.WriteString(`<p class="muted">No alternatives yet.</p>`)
	} else {
		b.WriteString(`<table><thead><tr><th>Rank</th><th>Alternative</th><th class="num">Weight</th><th>Share</th></tr></thead><tbody>`)
		for _, r := range result.Ranking {
			fmt.Fprintf(&b, `<tr><td>%d</td><td>%s</td><td class="num">%.4f</td><td><div class="bar"><span style="width:%.1f%%"></span></div></td></tr>`,
				r.Rank, html.EscapeString(r.Name), r.Weight, r.Weight*100)
		}
		b.WriteString(`</tbody></table>`)
	}

	b.WriteString(`</section><section id="weights"><h2>Weights by matrix</h2>`)
	for key, m := range result.Matrices {
		fmt.Fprintf(&b, `<h3>%s`, html.EscapeString(key))
		if m.CR != nil {
			fmt.Fprintf(&b, ` — CR %.3f`, *m.CR)
		}
		if !m.Complete {
			b.WriteString(` (incomplete)`)
		}
		b.WriteString(`</h3><table><thead><tr><th>Item</th><th class="num">Local weight</th><th>Share</th></tr></thead><tbody>`)
		for _, id := range m.IDs {
			wt := m.Weights[id]
			fmt.Fprintf(&b, `<tr><td>%s</td><td class="num">%.4f</td><td><div class="bar"><span style="width:%.1f%%"></span></div></td></tr>`,
				html.EscapeString(m.Names[id]), wt, wt*100)
		}
		b.WriteString(`</tbody></table>`)
	}

	b.WriteString(`</section><section id="repairs"><h2>Consistency repairs</h2>`)
	hasRepairs := false
	for key, m := range result.Matrices {
		if len(m.Repairs) == 0 {
			continue
		}
		hasRepairs = true
		cr := 0.0
		if m.CR != nil {
			cr = *m.CR
		}
		fmt.Fprintf(&b, `<h3>%s — CR %.3f</h3>`, html.EscapeString(key), cr)
		b.WriteString(`<p class="muted">Pairs that disagree most with derived weights. Suggested values snap to Saaty nearest to w<sub>i</sub>/w<sub>j</sub>.</p>
<table><thead><tr><th>Left</th><th>Right</th><th class="num">Current</th><th class="num">Implied</th><th class="num">Suggested</th></tr></thead><tbody>`)
		for _, h := range m.Repairs {
			fmt.Fprintf(&b, `<tr><td>%s</td><td>%s</td><td class="num">%s</td><td class="num">%.3f</td><td class="num">%s</td></tr>`,
				html.EscapeString(h.LeftName), html.EscapeString(h.RightName),
				html.EscapeString(engine.FormatSaaty(h.Current)), h.Implied, html.EscapeString(engine.FormatSaaty(h.Suggested)))
		}
		b.WriteString(`</tbody></table>`)
	}
	if !hasRepairs {
		b.WriteString(`<p class="muted">No repair hints — matrices are consistent or incomplete.</p>`)
	}

	b.WriteString(`</section><section id="matrices"><h2>Pairwise matrices</h2>`)
	for key, m := range result.Matrices {
		fmt.Fprintf(&b, `<h3>%s</h3>`, html.EscapeString(key))
		if len(m.IDs) == 0 {
			continue
		}
		b.WriteString(`<table><thead><tr><th></th>`)
		for _, id := range m.IDs {
			b.WriteString(`<th>` + html.EscapeString(m.Names[id]) + `</th>`)
		}
		b.WriteString(`</tr></thead><tbody>`)
		for i, rid := range m.IDs {
			b.WriteString(`<tr><th>` + html.EscapeString(m.Names[rid]) + `</th>`)
			for j := range m.IDs {
				cell := 0.0
				if i < len(m.Matrix) && j < len(m.Matrix[i]) {
					cell = m.Matrix[i][j]
				}
				b.WriteString(`<td class="num">` + html.EscapeString(fmtCell(cell)) + `</td>`)
			}
			b.WriteString(`</tr>`)
		}
		b.WriteString(`</tbody></table>`)
	}

	b.WriteString(`</section><section id="data"><h2>Workspace data (CSV)</h2><h3>Criteria</h3>`)
	writeSimpleTable(&b, []string{"id", "name", "parent_id", "description"}, criteriaRows(result.Criteria))
	b.WriteString(`<h3>Alternatives</h3>`)
	writeSimpleTable(&b, []string{"id", "name", "description"}, altRows(result.Alternatives))
	b.WriteString(`<h3>Objective attributes</h3>`)
	writeSimpleTable(&b, []string{"alternative_id", "criterion_id", "value", "unit", "source", "note"}, attrRows(result.Attributes))
	b.WriteString(`<h3>Pairwise judgments</h3>`)
	writePairwiseTable(&b, result.Pairwise)
	b.WriteString(`</section></main>
<footer>Generated by ahp-method (Go). Source: ahp.toml, data/*.csv. Derived: output/weights.csv, output/ranking.csv.</footer>
</body></html>`)
	return b.String()
}

func fmtCell(v float64) string {
	if math.Abs(v-math.Round(v)) < 1e-9 && v >= 1 {
		return fmt.Sprintf("%d", int(math.Round(v)))
	}
	inv := 1 / v
	if v < 1 && math.Abs(inv-math.Round(inv)) < 1e-9 {
		return fmt.Sprintf("1/%d", int(math.Round(inv)))
	}
	return fmt.Sprintf("%.3f", v)
}

func writeSimpleTable(b *strings.Builder, cols []string, rows []map[string]string) {
	if len(rows) == 0 {
		b.WriteString(`<p class="muted">Empty.</p>`)
		return
	}
	b.WriteString(`<table><thead><tr>`)
	for _, c := range cols {
		b.WriteString(`<th>` + html.EscapeString(c) + `</th>`)
	}
	b.WriteString(`</tr></thead><tbody>`)
	for _, row := range rows {
		b.WriteString(`<tr>`)
		for _, c := range cols {
			b.WriteString(`<td>` + html.EscapeString(row[c]) + `</td>`)
		}
		b.WriteString(`</tr>`)
	}
	b.WriteString(`</tbody></table>`)
}

func writePairwiseTable(b *strings.Builder, rows []workspace.PairwiseRow) {
	if len(rows) == 0 {
		b.WriteString(`<p class="muted">Empty.</p>`)
		return
	}
	b.WriteString(`<table><thead><tr><th>matrix</th><th>left</th><th>right</th><th>value</th><th>status</th><th>note</th></tr></thead><tbody>`)
	for _, r := range rows {
		fmt.Fprintf(b, `<tr><td>%s</td><td>%s</td><td>%s</td><td class="num">%s</td><td>%s</td><td>%s</td></tr>`,
			html.EscapeString(r.Matrix), html.EscapeString(r.Left), html.EscapeString(r.Right),
			html.EscapeString(engine.FormatSaaty(r.Value)), html.EscapeString(r.Status), html.EscapeString(r.Note))
	}
	b.WriteString(`</tbody></table>`)
}

func criteriaRows(items []workspace.Criterion) []map[string]string {
	out := make([]map[string]string, 0, len(items))
	for _, c := range items {
		out = append(out, map[string]string{"id": c.ID, "name": c.Name, "parent_id": c.ParentID, "description": c.Description})
	}
	return out
}

func altRows(items []workspace.Alternative) []map[string]string {
	out := make([]map[string]string, 0, len(items))
	for _, a := range items {
		out = append(out, map[string]string{"id": a.ID, "name": a.Name, "description": a.Description})
	}
	return out
}

func attrRows(items []workspace.AttributeRow) []map[string]string {
	out := make([]map[string]string, 0, len(items))
	for _, a := range items {
		out = append(out, map[string]string{
			"alternative_id": a.AlternativeID, "criterion_id": a.CriterionID,
			"value": a.Value, "unit": a.Unit, "source": a.Source, "note": a.Note,
		})
	}
	return out
}
