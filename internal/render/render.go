package render

import (
	"fmt"
	"html"
	"math"
	"sort"
	"strings"

	"github.com/lazarok09/ahp-method/internal/engine"
	"github.com/lazarok09/ahp-method/internal/workspace"
)

func HTML(result *workspace.ComputeResult) string {
	var b strings.Builder
	matrixKeys := sortedMatrixKeys(result.Matrices)
	hasComparative := result.Gaussian != nil || result.Absolute != nil
	hasExplain := len(result.Ranking) > 0 && len(result.LeafWeights) > 0

	b.WriteString(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8"/>
<meta name="viewport" content="width=device-width, initial-scale=1"/>
<title>`)
	b.WriteString(html.EscapeString(result.Title))
	b.WriteString(` — AHP report</title>
<link rel="preconnect" href="https://fonts.googleapis.com"/>
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin/>
<link href="https://fonts.googleapis.com/css2?family=IBM+Plex+Mono:wght@400;500;600&family=IBM+Plex+Sans:wght@400;500;600;700&display=swap" rel="stylesheet"/>
<style>
`)
	b.WriteString(reportCSS)
	b.WriteString(`</style></head><body>
<div class="shell">
`)
	writeTOC(&b, tocOpts{
		HasComparative: hasComparative,
		HasExplain:     hasExplain,
		MatrixKeys:     matrixKeys,
		Criteria:       result.Criteria,
	})

	b.WriteString(`<div class="content"><header>
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
	b.WriteString(`</div></header><main>`)

	if len(result.Warnings) > 0 {
		b.WriteString(`<div class="card warn-list"><strong>Warnings</strong><ul>`)
		for _, w := range result.Warnings {
			b.WriteString(`<li>` + html.EscapeString(w) + `</li>`)
		}
		b.WriteString(`</ul></div>`)
	}

	b.WriteString(`<section id="method"><h2 id="method-h">How this method works</h2>
<p>AHP turns a decision into a hierarchy (goal → criteria → alternatives), then asks for pairwise Saaty judgments (1 = equal, 9 = extreme preference). Each complete reciprocal matrix has a principal eigenvector: those entries are the local weights. Consistency ratio (CR) flags whether the judgments contradict each other; CR ≤ 0.10 is the usual acceptance cut.</p>
<p>Global alternative scores are the weighted sum of local scores across leaf criteria. Objective facts live in <code>data/attributes.csv</code> so they can inform human (or proposed) judgments without becoming the ranking themselves.</p>
<p class="muted">Edit CSVs, then re-run <code>ahp compute</code>. Proposal rows are ignored until status is <code>committed</code>. Opt-in comparative views: <code>ahp absolute</code> / <code>ahp gaussian</code> (Santos absolute measurement + σ/μ reweight) — see below. Those scores have <strong>no Saaty CR</strong>.</p>
</section>

<section id="ranking"><h2 id="ranking-h">Global ranking (Saaty)</h2>`)

	if len(result.Ranking) == 0 {
		b.WriteString(`<p class="muted">No alternatives yet.</p>`)
	} else {
		b.WriteString(`<div class="table-wrap"><table><thead><tr><th>Rank</th><th>Alternative</th><th class="num">Weight</th><th>Share</th></tr></thead><tbody>`)
		for _, r := range result.Ranking {
			fmt.Fprintf(&b, `<tr><td>%d</td><td>%s</td><td class="num">%.4f</td><td><div class="bar"><span style="width:%.1f%%"></span></div></td></tr>`,
				r.Rank, html.EscapeString(r.Name), r.Weight, r.Weight*100)
		}
		b.WriteString(`</tbody></table></div>`)
		writeContributionBars(&b, result)
		writeChoiceProfileRadar(&b, result)
	}
	b.WriteString(`</section>`)

	writeComparativeSection(&b, result)

	if hasExplain {
		leaf, local, altIDs, altNames, critNames := explainInputs(result)
		exp := engine.Explain(leaf, local, altIDs, altNames, critNames)
		b.WriteString(`<section id="explain"><h2 id="explain-h">Contribution breakdown</h2>
<p class="muted">Each global weight is the sum of criterion_weight × local_weight. Use <code>ahp explain</code> / <code>ahp sensitivity</code> for terminal analysis.</p>`)
		for _, a := range exp.Alternatives {
			aid := "explain-" + slug(a.ID)
			fmt.Fprintf(&b, `<h3 id="%s">#%d %s — %.4f</h3>`, aid, a.Rank, html.EscapeString(a.Name), a.GlobalWeight)
			b.WriteString(`<div class="table-wrap"><table><thead><tr><th>Criterion</th><th class="num">Crit wt</th><th class="num">Local</th><th class="num">Contribution</th><th>Share</th></tr></thead><tbody>`)
			for _, c := range a.Contributions {
				fmt.Fprintf(&b, `<tr><td>%s</td><td class="num">%.3f</td><td class="num">%.3f</td><td class="num">%.4f</td><td><div class="bar"><span style="width:%.1f%%"></span></div></td></tr>`,
					html.EscapeString(c.CriterionName), c.CriterionWeight, c.LocalWeight, c.Contribution, c.Share*100)
			}
			b.WriteString(`</tbody></table></div>`)
		}
		b.WriteString(`</section>`)
	}

	b.WriteString(`<section id="weights"><h2 id="weights-h">Weights by matrix</h2>`)
	for _, key := range matrixKeys {
		m := result.Matrices[key]
		meta := describeMatrix(key, result.Criteria)
		fmt.Fprintf(&b, `<h3 id="weight-%s">%s`, slug(key), html.EscapeString(meta.Title))
		if m.CR != nil {
			fmt.Fprintf(&b, ` — CR %.3f`, *m.CR)
		}
		if !m.Complete {
			b.WriteString(` (incomplete)`)
		}
		b.WriteString(`</h3>`)
		if meta.Subtitle != "" {
			fmt.Fprintf(&b, `<p class="muted matrix-sub">%s</p>`, html.EscapeString(meta.Subtitle))
		}
		b.WriteString(`<div class="table-wrap"><table><thead><tr><th>Item</th><th class="num">Local weight</th><th>Share</th></tr></thead><tbody>`)
		for _, id := range m.IDs {
			wt := m.Weights[id]
			fmt.Fprintf(&b, `<tr><td>%s</td><td class="num">%.4f</td><td><div class="bar"><span style="width:%.1f%%"></span></div></td></tr>`,
				html.EscapeString(m.Names[id]), wt, wt*100)
		}
		b.WriteString(`</tbody></table></div>`)
	}
	b.WriteString(`</section>`)

	b.WriteString(`<section id="repairs"><h2 id="repairs-h">Consistency repairs</h2>`)
	hasRepairs := false
	for _, key := range matrixKeys {
		m := result.Matrices[key]
		if len(m.Repairs) == 0 {
			continue
		}
		hasRepairs = true
		cr := 0.0
		if m.CR != nil {
			cr = *m.CR
		}
		meta := describeMatrix(key, result.Criteria)
		fmt.Fprintf(&b, `<h3 id="repair-%s">%s — CR %.3f</h3>`, slug(key), html.EscapeString(meta.Title), cr)
		b.WriteString(`<p class="muted">Pairs that disagree most with derived weights. Suggested values snap to Saaty nearest to w<sub>i</sub>/w<sub>j</sub>.</p>
<div class="table-wrap"><table><thead><tr><th>Left</th><th>Right</th><th class="num">Current</th><th class="num">Implied</th><th class="num">Suggested</th></tr></thead><tbody>`)
		for _, h := range m.Repairs {
			fmt.Fprintf(&b, `<tr><td>%s</td><td>%s</td><td class="num">%s</td><td class="num">%.3f</td><td class="num">%s</td></tr>`,
				html.EscapeString(h.LeftName), html.EscapeString(h.RightName),
				html.EscapeString(engine.FormatSaaty(h.Current)), h.Implied, html.EscapeString(engine.FormatSaaty(h.Suggested)))
		}
		b.WriteString(`</tbody></table></div>`)
	}
	if !hasRepairs {
		b.WriteString(`<p class="muted">No repair hints — matrices are consistent or incomplete.</p>`)
	}
	b.WriteString(`</section>`)

	writeMatricesSection(&b, result, matrixKeys)

	b.WriteString(`<section id="data"><h2 id="data-h">Workspace data (CSV)</h2><h3 id="data-criteria">Criteria</h3>`)
	writeSimpleTable(&b, []string{"id", "name", "parent_id", "description"}, criteriaRows(result.Criteria))
	b.WriteString(`<h3 id="data-alternatives">Alternatives</h3>`)
	writeSimpleTable(&b, []string{"id", "name", "description"}, altRows(result.Alternatives))
	b.WriteString(`<h3 id="data-attributes">Objective attributes</h3>`)
	writeSimpleTable(&b, []string{"alternative_id", "criterion_id", "value", "unit", "source", "note"}, attrRows(result.Attributes))
	b.WriteString(`<h3 id="data-pairwise">Pairwise judgments</h3>`)
	writePairwiseTable(&b, result.Pairwise)
	b.WriteString(`</section></main>
<footer>Generated by ahp-method (Go). Source: ahp.toml, data/*.csv. Derived: output/weights.csv, output/ranking.csv.</footer>
</div></div>
</body></html>`)
	return b.String()
}

const reportCSS = `
:root{
  --bg:#f3f1eb;--ink:#1a1814;--muted:#5c564c;--line:#d6cec0;--card:#fffcf6;
  --accent:#1f6b55;--accent-soft:#d8eee6;--warn:#8a4b12;--bad:#8f1d1d;--ok:#1f6b45;
  --hi:#0f7a5f;--lo:#b9483a;--diag:#ebe4d6;--glow-hi:rgba(31,107,85,.35);--glow-lo:rgba(185,72,58,.32);
  --gutter:.875rem;--toc-w:14rem;--shell-max:100%;--prose:46rem;
  --font-sans:"IBM Plex Sans",ui-sans-serif,system-ui,-apple-system,"Segoe UI",Roboto,sans-serif;
  --font-mono:"IBM Plex Mono",ui-monospace,SFMono-Regular,Menlo,Consolas,monospace;
}
*{box-sizing:border-box}
html{scroll-behavior:smooth}
body{
  margin:0;color:var(--ink);
  font-family:var(--font-sans);
  font-size:clamp(17px,1.05vw + 14px,19px);
  line-height:1.55;
  font-variant-numeric:tabular-nums lining-nums;
  background:
  radial-gradient(900px 480px at 8% -8%,#e8f3ee 0%,transparent 55%),
  radial-gradient(700px 420px at 100% 0%,#f7ebe4 0%,transparent 50%),
  var(--bg)
}

/* Mobile-first shell */
.shell{
  display:grid;grid-template-columns:minmax(0,1fr);gap:.85rem;
  width:min(100%,var(--shell-max));margin:0 auto;padding:var(--gutter) var(--gutter) 2.5rem;
}
.content{min-width:0}
header{padding:.35rem 0 0}
main{padding:.25rem 0 0}
footer{padding:1.5rem 0 0;color:var(--muted);font-size:.92rem}
h1{font-size:clamp(1.65rem,4.5vw,2.35rem);margin:0 0 .45rem;letter-spacing:-.02em;line-height:1.2;font-weight:700}
h2{font-size:clamp(1.25rem,2.8vw,1.5rem);margin:0 0 .75rem;padding-bottom:.35rem;border-bottom:1px solid var(--line);font-weight:600}
h3{font-size:1.12rem;margin:1.2rem 0 .5rem;font-weight:600}
.muted{color:var(--muted);font-size:1.02rem;line-height:1.55}
.lede{font-size:clamp(1.08rem,1.2vw + .85rem,1.22rem);line-height:1.6;max-width:var(--prose);margin:.35rem 0 .85rem}
p{margin:.55rem 0;max-width:var(--prose)}
section{margin:1.75rem 0}
section[id],h2[id],h3[id],article[id],.matrix-card[id]{
  scroll-margin-block-start:28vh;scroll-margin-block-end:28vh;
}
.status{display:flex;gap:.45rem;flex-wrap:wrap;margin:.85rem 0 0}
.pill{display:inline-block;padding:.22rem .65rem;border:1px solid var(--line);border-radius:999px;font-size:.88rem;background:var(--card);font-weight:500}
.pill.ok{color:var(--ok);border-color:#b9d7c6;background:#eef7f1}
.pill.bad{color:var(--bad);border-color:#e0b7b7;background:#faf0f0}
.pill.warn{color:var(--warn);border-color:#e2c8a4;background:#fbf4ea}
.card{background:var(--card);border:1px solid var(--line);padding:.85rem .95rem;margin:.7rem 0;border-radius:12px}
.table-wrap{overflow-x:auto;-webkit-overflow-scrolling:touch;margin:.4rem 0 1rem;border-radius:10px;border:1px solid var(--line)}
table{width:100%;border-collapse:collapse;font-variant-numeric:tabular-nums lining-nums;background:var(--card);font-size:1rem}
th,td{border:1px solid var(--line);padding:.5rem .65rem;text-align:left}
th{font-size:.88rem;font-weight:600}
td.num,th.num{text-align:right;font-family:var(--font-mono);font-size:.95em}
.bar{height:10px;background:#ece7dc;border-radius:999px;overflow:hidden}
.bar>span{display:block;height:100%;background:linear-gradient(90deg,#2a8f72,var(--accent));border-radius:999px}
.crit-weights{display:flex;flex-direction:column;gap:.85rem;margin:.2rem 0 .1rem}
.crit-weight-row{display:grid;grid-template-columns:2.4rem minmax(0,1fr) auto;column-gap:.65rem;row-gap:.35rem;align-items:baseline}
.crit-weight-rank{
  font-family:var(--font-sans);font-weight:700;font-size:.85rem;letter-spacing:.04em;
  color:var(--accent);line-height:1
}
.crit-weight-name{font-weight:600;font-size:1.02rem;min-width:0;line-height:1.3;color:var(--ink)}
.crit-weight-pct{
  font-family:var(--font-sans);font-weight:700;font-variant-numeric:tabular-nums;
  font-size:1.35rem;letter-spacing:-.02em;line-height:1;color:var(--ink);text-align:right
}
.crit-weight-bar{grid-column:1 / -1;height:14px;background:#ece7dc;border-radius:999px;overflow:hidden}
.crit-weight-bar>span{display:block;height:100%;background:linear-gradient(90deg,#2a8f72,var(--accent));border-radius:999px}
.crit-weight-row.is-top .crit-weight-pct{color:var(--accent)}
.crit-weight-row.is-top .crit-weight-bar>span{background:linear-gradient(90deg,#1a7a62,var(--accent))}
.warn-list{color:var(--warn)}
code{font-family:var(--font-mono);font-size:.9em;line-height:1.35;background:#efeae1;padding:.05em .3em;border-radius:4px}

/* TOC: chip row on small screens */
nav.toc{
  position:sticky;top:0;z-index:50;
  display:flex;flex-direction:row;flex-wrap:nowrap;align-items:center;gap:.3rem;
  padding:.55rem .6rem;border:1px solid var(--line);border-radius:12px;
  background:color-mix(in srgb, var(--card) 94%, transparent);
  backdrop-filter:blur(10px);-webkit-backdrop-filter:blur(10px);
  box-shadow:0 6px 18px rgba(28,25,22,.06);
  overflow-x:auto;-webkit-overflow-scrolling:touch;scrollbar-width:thin;
  font-family:var(--font-sans);font-size:.88rem;line-height:1.35;
}
nav.toc .toc-title{
  flex:0 0 auto;margin:0 .15rem 0 0;padding-right:.45rem;border-right:1px solid var(--line);
  font-weight:700;letter-spacing:.05em;text-transform:uppercase;color:var(--muted);font-size:.72rem;white-space:nowrap
}
nav.toc a{
  flex:0 0 auto;color:var(--accent);text-decoration:none;padding:.32rem .6rem;border-radius:999px;
  border:1px solid transparent;white-space:nowrap;background:transparent;font-weight:500
}
nav.toc a:hover{background:var(--accent-soft);border-color:#c5ddd4}
nav.toc a.toc-sub{display:none}

/* Pairwise */
.matrix-legend{margin:.15rem 0 .5rem;font-size:1.02rem}
.legend-swatches{display:flex;align-items:center;gap:.3rem;margin:0 0 .9rem;flex-wrap:wrap}
.legend-swatches .sw{width:1rem;height:1rem;border-radius:4px;border:1px solid var(--line)}
.legend-swatches .legend-label{margin-left:.25rem;font-size:.9rem;color:var(--muted)}
.legend-swatches .sw.eq{background:#f7f3eb}
.legend-swatches .sw.hi.s5{background:color-mix(in srgb,var(--accent) 32%,var(--card))}
.legend-swatches .sw.hi.s9{background:color-mix(in srgb,var(--accent) 66%,var(--card))}
.legend-swatches .sw.lo.s5{background:color-mix(in srgb,var(--warn) 32%,var(--card))}
.legend-swatches .sw.lo.s9{background:color-mix(in srgb,var(--warn) 66%,var(--card))}
.matrices-grid{display:grid;grid-template-columns:1fr;gap:.9rem}
.matrices-group{margin:1.25rem 0 1.6rem}
.matrices-group > h3{margin:0 0 .35rem;font-size:1.15rem;border-bottom:0}
.matrices-group > .matrix-sub{margin:0 0 .85rem}
.matrix-card{
  margin:0;padding:.85rem .85rem 1rem;border:1px solid var(--line);border-radius:14px;
  background:linear-gradient(180deg,#fffef9 0%,var(--card) 100%);
  box-shadow:0 6px 18px rgba(28,25,22,.05);min-width:0
}
.matrix-head{display:flex;align-items:flex-start;justify-content:space-between;gap:.55rem;flex-wrap:wrap;margin-bottom:.55rem}
.matrix-head .matrix-titles{flex:1 1 12rem;min-width:0}
.matrix-head h3{margin:.25rem 0 0;font-size:1.12rem}
.matrix-head .matrix-pills{display:flex;gap:.35rem;flex-wrap:wrap;align-items:center}
.matrix-sub{margin:.25rem 0 0;font-size:1.02rem;line-height:1.5;max-width:42rem;color:var(--muted)}
.pill.kind{color:var(--accent);border-color:#b9d7c6;background:#eef7f1;font-weight:600;letter-spacing:.02em}
.matrix-wrap{overflow-x:auto;-webkit-overflow-scrolling:touch;border-radius:10px;border:1px solid var(--line)}
.matrix-expand{
  appearance:none;cursor:pointer;font-family:var(--font-sans);font-weight:600;font-size:.88rem;
  padding:.22rem .7rem;border-radius:999px;border:1px solid #92400e;
  background:#b45309;color:#fff
}
.matrix-expand:hover{background:#92400e;border-color:#78350f}
.matrix-expand:focus-visible{outline:2px solid #b45309;outline-offset:2px}

dialog.matrix-dialog{
  width:80vw;max-width:80%;max-height:85vh;
  margin:auto;border:1px solid var(--line);border-radius:16px;padding:0;
  background:var(--card);color:var(--ink);
  box-shadow:0 28px 80px rgba(26,24,20,.28)
}
dialog.matrix-dialog::backdrop{
  background:rgba(26,24,20,.48);
  backdrop-filter:blur(3px);-webkit-backdrop-filter:blur(3px)
}
.matrix-dialog-shell{display:flex;flex-direction:column;max-height:85vh;min-height:0}
.matrix-dialog-head{
  display:flex;align-items:flex-start;justify-content:space-between;gap:1rem;
  padding:1.05rem 1.25rem;border-bottom:1px solid var(--line);flex:0 0 auto;background:var(--card)
}
.matrix-dialog-titles{min-width:0;flex:1 1 auto}
.matrix-dialog-titles h3{margin:.3rem 0 0;font-size:1.2rem}
.matrix-dialog-close{
  appearance:none;cursor:pointer;flex:0 0 auto;font-family:var(--font-sans);font-weight:600;font-size:.92rem;
  padding:.4rem .85rem;border-radius:10px;border:1px solid var(--line);background:#f3efe6;color:var(--ink)
}
.matrix-dialog-close:hover{background:#ebe6da}
.matrix-dialog-body{flex:1 1 auto;min-height:0;overflow:auto;padding:1rem 1.25rem 1.35rem}
.matrix-wrap-dialog{border-radius:12px}
.matrix-wrap-dialog table.matrix{font-size:1rem}
.matrix-wrap-dialog table.matrix thead th,
.matrix-wrap-dialog table.matrix tbody th{font-size:.88rem;max-width:12rem;white-space:normal;overflow:visible;text-overflow:unset}
table.matrix{
  width:100%;border-collapse:separate;border-spacing:2px;font-size:.95rem;
  background:var(--line);font-family:var(--font-mono)
}
table.matrix th,table.matrix td{
  border:1px solid color-mix(in srgb,var(--line) 70%,#8a8276);
  padding:.45rem .4rem;min-width:2.5rem;text-align:center
}
table.matrix td{background:var(--card)}
table.matrix thead th,table.matrix tbody th{
  background:#f0ebe3;font-size:.78rem;font-weight:600;color:#3d3830;font-family:var(--font-sans);
  position:sticky;left:0;z-index:1;text-align:left;white-space:nowrap;max-width:7.5rem;overflow:hidden;text-overflow:ellipsis;
  border-color:color-mix(in srgb,var(--line) 55%,#9a9184);
  box-shadow:1px 0 0 var(--line)
}
table.matrix thead th{
  text-align:center;position:sticky;top:0;z-index:2;left:auto;max-width:none;
  box-shadow:0 1px 0 var(--line)
}
table.matrix th.corner{left:0;z-index:3;background:#e8e2d8;box-shadow:1px 1px 0 var(--line)}
table.matrix td span{display:inline-block;min-width:1.35rem;font-variant-numeric:tabular-nums;font-weight:600}
table.matrix td.diag{background:#efeae1;color:var(--muted)}
table.matrix td.eq{background:#f7f3eb;color:#4a453c}
/* Preference fills — muted ink-on-paper, same accent/warn family as the report */
table.matrix td.hi.s2{background:color-mix(in srgb,var(--accent) 8%,var(--card))}
table.matrix td.hi.s3{background:color-mix(in srgb,var(--accent) 14%,var(--card))}
table.matrix td.hi.s4{background:color-mix(in srgb,var(--accent) 22%,var(--card))}
table.matrix td.hi.s5{background:color-mix(in srgb,var(--accent) 32%,var(--card))}
table.matrix td.hi.s6{background:color-mix(in srgb,var(--accent) 42%,var(--card));color:#163f34}
table.matrix td.hi.s7{background:color-mix(in srgb,var(--accent) 54%,var(--card));color:#12352c}
table.matrix td.hi.s8,table.matrix td.hi.s9{background:color-mix(in srgb,var(--accent) 66%,var(--card));color:#0e2c24}
table.matrix td.lo.s2{background:color-mix(in srgb,var(--warn) 8%,var(--card))}
table.matrix td.lo.s3{background:color-mix(in srgb,var(--warn) 14%,var(--card))}
table.matrix td.lo.s4{background:color-mix(in srgb,var(--warn) 22%,var(--card))}
table.matrix td.lo.s5{background:color-mix(in srgb,var(--warn) 32%,var(--card))}
table.matrix td.lo.s6{background:color-mix(in srgb,var(--warn) 42%,var(--card));color:#5a3210}
table.matrix td.lo.s7{background:color-mix(in srgb,var(--warn) 54%,var(--card));color:#4a2a0e}
table.matrix td.lo.s8,table.matrix td.lo.s9{background:color-mix(in srgb,var(--warn) 66%,var(--card));color:#3a210c}

/* Tablet */
@media (min-width:720px){
  :root{--gutter:1.15rem;--prose:46rem}
  section{margin:2.1rem 0}
  table{font-size:.95rem}
  nav.toc{flex-wrap:wrap;overflow-x:visible}
  nav.toc a.toc-sub{display:inline-flex;padding:.22rem .5rem;font-size:.7rem;color:var(--muted);border-radius:8px}
}

/* Laptop / small desktop: sticky side TOC, grow shell */
@media (min-width:1100px){
  :root{--gutter:1.35rem;--toc-w:15rem;--shell-max:1080px;--prose:48rem}
  .shell{
    grid-template-columns:minmax(0,1fr) var(--toc-w);
    align-items:start;gap:1.25rem 1.75rem;padding-top:1.25rem
  }
  nav.toc{
    grid-column:2;grid-row:1;position:sticky;top:1rem;
    flex-direction:column;flex-wrap:nowrap;align-items:stretch;
    width:100%;max-height:calc(100vh - 2rem);overflow:auto;
    padding:.85rem .9rem;border-radius:14px;gap:.18rem
  }
  nav.toc .toc-title{border-right:0;border-bottom:1px solid var(--line);padding:0 0 .45rem;margin:0 0 .25rem;width:100%}
  nav.toc a{border-radius:8px;padding:.32rem .45rem}
  nav.toc a.toc-sub{display:block;padding-left:.85rem;font-size:.72rem}
  .content{grid-column:1;grid-row:1}
  section[id],h2[id],h3[id],article[id],.matrix-card[id]{
    scroll-margin-block-start:42vh;scroll-margin-block-end:42vh
  }
}

/* Full HD */
@media (min-width:1600px){
  :root{--gutter:1.6rem;--toc-w:16rem;--shell-max:1480px;--prose:52rem}
  .matrices-grid{grid-template-columns:repeat(2,minmax(0,1fr));gap:1.1rem}
  table.matrix{font-size:.9rem}
  table.matrix th,table.matrix td{min-width:2.55rem;padding:.4rem .35rem}
}

/* Ultrawide */
@media (min-width:2200px){
  :root{--gutter:2rem;--toc-w:17rem;--shell-max:2040px;--prose:56rem}
  .matrices-grid{grid-template-columns:repeat(3,minmax(0,1fr));gap:1.25rem}
  body{font-size:19px}
  .lede{max-width:var(--prose)}
}

@media (min-width:3000px){
  :root{--shell-max:2600px;--toc-w:18rem}
  .matrices-grid{grid-template-columns:repeat(4,minmax(0,1fr))}
}

.chart-card{
  margin:1.25rem 0 0;padding:1.25rem 1.35rem 1.45rem;background:var(--card);
  border:1px solid var(--line);border-radius:14px;max-width:min(100%,var(--prose,52rem));
  width:100%
}
.chart-kicker{
  margin:0 0 .4rem;font-family:var(--font-sans);font-weight:600;font-size:.82rem;line-height:1.25;
  letter-spacing:.14em;text-transform:uppercase;color:var(--accent)
}
.chart-lede{margin:0 0 1rem;font-size:1.05rem;line-height:1.55;max-width:44rem;color:var(--muted)}
svg.radar{display:block;width:100%;max-width:44rem;height:auto;margin-inline:auto}
.radar-grid{fill:none;stroke:var(--line);stroke-width:1.25}
.radar-spoke{stroke:var(--line);stroke-width:1.25}
.radar-series{stroke-width:2.25;stroke-linejoin:round}
.radar-label{font-family:var(--font-sans);font-weight:600;font-size:.9rem;line-height:1.3;fill:var(--muted)}
.radar-legend{
  list-style:none;margin:.95rem 0 0;padding:0;display:flex;flex-wrap:wrap;gap:.55rem 1.1rem;
  font-family:var(--font-sans);font-weight:600;font-size:.95rem;line-height:1.35;color:var(--ink);justify-content:center
}
.radar-legend .swatch{
  display:inline-block;width:.8rem;height:.8rem;border-radius:2px;margin-right:.4rem;
  vertical-align:-.05rem
}
.contrib-legend{
  display:flex;justify-content:space-between;align-items:flex-start;gap:2.5rem 3.5rem;
  margin:0 0 1.15rem;padding:0
}
.contrib-legend-col{
  list-style:none;margin:0;padding:0;display:flex;flex-direction:column;gap:.45rem;
  flex:1 1 0;min-width:0;max-width:calc(50% - 1rem);
  font-family:var(--font-sans);font-weight:600;font-size:.95rem;line-height:1.4;color:var(--ink)
}
.contrib-legend-col .swatch{
  display:inline-block;width:.75rem;height:.75rem;border-radius:2px;margin-right:.45rem;vertical-align:-.08rem
}
@media (max-width:640px){
  .contrib-legend{flex-direction:column;gap:.85rem}
  .contrib-legend-col{max-width:none}
}
.contrib-chart{display:flex;flex-direction:column;gap:1rem}
.contrib-row{display:flex;flex-direction:column;gap:.4rem;align-items:stretch;min-width:0}
.contrib-meta{display:flex;flex-wrap:wrap;align-items:baseline;gap:.4rem .55rem;min-width:0;width:100%}
.contrib-rank{font-family:var(--font-sans);font-weight:700;font-size:.85rem;line-height:1;color:var(--accent);letter-spacing:.04em}
.contrib-name{font-family:var(--font-sans);font-weight:600;font-size:1.02rem;line-height:1.3;color:var(--ink);overflow-wrap:anywhere}
.contrib-total{font-family:var(--font-mono);font-weight:600;font-size:.92rem;line-height:1;color:var(--muted);margin-left:auto}
.contrib-track{
  display:flex;width:100%;height:1.55rem;border-radius:8px;overflow:hidden;background:color-mix(in srgb,var(--line) 55%,transparent);
  min-width:0
}
.contrib-seg{display:block;height:100%;min-width:0;transition:filter 120ms ease}
.contrib-seg:hover{filter:brightness(1.08)}
`

type tocOpts struct {
	HasComparative bool
	HasExplain     bool
	MatrixKeys     []string
	Criteria       []workspace.Criterion
}

func writeTOC(b *strings.Builder, opts tocOpts) {
	b.WriteString(`<nav class="toc" aria-label="Table of contents"><p class="toc-title">Contents</p>`)
	// Labels match the visible section <h2> titles so Contents stays correct as sections change.
	links := []struct{ href, label, class string }{
		{"#method", "How this method works", ""},
		{"#ranking", "Global ranking (Saaty)", ""},
	}
	if opts.HasComparative {
		links = append(links, struct{ href, label, class string }{"#comparative", "Absolute / Gaussian (comparative)", ""})
	}
	if opts.HasExplain {
		links = append(links, struct{ href, label, class string }{"#explain", "Contribution breakdown", ""})
	}
	links = append(links,
		struct{ href, label, class string }{"#weights", "Weights by matrix", ""},
		struct{ href, label, class string }{"#repairs", "Consistency repairs", ""},
		struct{ href, label, class string }{"#matrices", "Pairwise comparisons", ""},
	)
	for _, key := range opts.MatrixKeys {
		meta := describeMatrix(key, opts.Criteria)
		links = append(links, struct{ href, label, class string }{
			"#matrix-" + slug(key), meta.Title, "toc-sub",
		})
	}
	links = append(links, struct{ href, label, class string }{"#data", "Workspace data (CSV)", ""})
	for _, l := range links {
		cls := ""
		if l.class != "" {
			cls = ` class="` + l.class + `"`
		}
		fmt.Fprintf(b, `<a href="%s"%s>%s</a>`, l.href, cls, html.EscapeString(l.label))
	}
	b.WriteString(`</nav>`)
}

type matrixMeta struct {
	Key      string
	Title    string
	Short    string
	Subtitle string
	Badge    string
	Kind     string // criteria | subcriteria | alternatives | other
}

func describeMatrix(key string, criteria []workspace.Criterion) matrixMeta {
	names := criterionNames(criteria)
	switch {
	case key == "criteria":
		return matrixMeta{
			Key: key, Kind: "criteria", Badge: "Criteria",
			Title:    "Criteria importance",
			Short:    "Criteria importance",
			Subtitle: "Local weights from the criteria pairwise (eigenvector). Expand for Saaty judgments.",
		}
	case strings.HasPrefix(key, "criteria:"):
		pid := strings.TrimPrefix(key, "criteria:")
		name := names[pid]
		if name == "" {
			name = pid
		}
		return matrixMeta{
			Key: key, Kind: "subcriteria", Badge: "Sub-criteria",
			Title:    "Under “" + name + "”",
			Short:    name + " (sub)",
			Subtitle: "Local weights among children under this parent. Expand for Saaty judgments.",
		}
	case strings.HasPrefix(key, "alt:"):
		cid := strings.TrimPrefix(key, "alt:")
		name := names[cid]
		if name == "" {
			name = cid
		}
		return matrixMeta{
			Key: key, Kind: "alternatives", Badge: "Alternatives",
			Title:    name,
			Short:    name,
			Subtitle: "Which option is better on this criterion? (alternative vs alternative)",
		}
	default:
		return matrixMeta{
			Key: key, Kind: "other", Badge: "Matrix",
			Title: key, Short: key,
		}
	}
}

func criterionNames(criteria []workspace.Criterion) map[string]string {
	out := make(map[string]string, len(criteria))
	for _, c := range criteria {
		n := strings.TrimSpace(c.Name)
		if n == "" {
			n = c.ID
		}
		out[c.ID] = n
	}
	return out
}

func writeMatricesSection(b *strings.Builder, result *workspace.ComputeResult, matrixKeys []string) {
	var critKeys, altKeys, otherKeys []string
	for _, key := range matrixKeys {
		meta := describeMatrix(key, result.Criteria)
		switch meta.Kind {
		case "criteria", "subcriteria":
			critKeys = append(critKeys, key)
		case "alternatives":
			altKeys = append(altKeys, key)
		default:
			otherKeys = append(otherKeys, key)
		}
	}

	b.WriteString(`<section id="matrices"><h2 id="matrices-h">Pairwise comparisons</h2>`)

	if len(critKeys) > 0 {
		b.WriteString(`<div class="matrices-group" id="matrices-criteria">
<h3>Criteria importance</h3>
<p class="muted matrix-sub">Bar length is the local weight (how much each criterion matters). Expand opens the Saaty pairwise judgments that produced these weights.</p>
<div class="matrices-grid">`)
		for _, key := range critKeys {
			writeCriteriaWeightCard(b, key, result)
		}
		b.WriteString(`</div></div>`)
	}

	if len(altKeys) > 0 {
		b.WriteString(`<div class="matrices-group" id="matrices-alternatives">
<h3>Alternatives by criterion</h3>
<p class="muted matrix-sub">For each criterion below, options are compared against each other — not criterion vs criterion.</p>
<p class="muted matrix-legend">Each cell is a Saaty judgment: teal = row preferred over column, coral = column preferred, neutral = equal. Diagonal is always 1.</p>
<div class="legend-swatches" aria-hidden="true">
<span class="sw lo s9"></span><span class="sw lo s5"></span><span class="sw eq"></span><span class="sw hi s5"></span><span class="sw hi s9"></span>
<span class="legend-label">column wins ← equal → row wins</span>
</div>
<div class="matrices-grid">`)
		for _, key := range altKeys {
			writeMatrixCard(b, key, result)
		}
		b.WriteString(`</div></div>`)
	}

	if len(otherKeys) > 0 {
		b.WriteString(`<div class="matrices-group" id="matrices-other"><div class="matrices-grid">`)
		for _, key := range otherKeys {
			writeMatrixCard(b, key, result)
		}
		b.WriteString(`</div></div>`)
	}

	b.WriteString(`</section>`)
}

func writeCriteriaWeightCard(b *strings.Builder, key string, result *workspace.ComputeResult) {
	m := result.Matrices[key]
	meta := describeMatrix(key, result.Criteria)
	mid := "matrix-" + slug(key)
	dialogID := mid + "-dialog"

	writeMatrixCardHeader(b, mid, dialogID, meta, m)
	if len(m.IDs) == 0 {
		b.WriteString(`<p class="muted">Empty matrix.</p></article>`)
		return
	}
	writeCriteriaWeightBars(b, m)
	writeMatrixJudgmentDialog(b, mid, dialogID, meta, m)
	b.WriteString(`</article>`)
}

func writeMatrixCard(b *strings.Builder, key string, result *workspace.ComputeResult) {
	m := result.Matrices[key]
	meta := describeMatrix(key, result.Criteria)
	mid := "matrix-" + slug(key)
	dialogID := mid + "-dialog"

	writeMatrixCardHeader(b, mid, dialogID, meta, m)
	if len(m.IDs) == 0 {
		b.WriteString(`<p class="muted">Empty matrix.</p></article>`)
		return
	}
	b.WriteString(`<div class="matrix-wrap">`)
	writeMatrixTable(b, m, false)
	b.WriteString(`</div>`)
	writeMatrixJudgmentDialog(b, mid, dialogID, meta, m)
	b.WriteString(`</article>`)
}

func writeMatrixCardHeader(b *strings.Builder, mid, dialogID string, meta matrixMeta, m workspace.MatrixPayload) {
	fmt.Fprintf(b, `<article class="matrix-card" id="%s"><header class="matrix-head"><div class="matrix-titles">`, mid)
	fmt.Fprintf(b, `<span class="pill kind">%s</span>`, html.EscapeString(meta.Badge))
	fmt.Fprintf(b, `<h3 id="%s-h">%s</h3>`, mid, html.EscapeString(meta.Title))
	if meta.Subtitle != "" {
		fmt.Fprintf(b, `<p class="muted matrix-sub">%s</p>`, html.EscapeString(meta.Subtitle))
	}
	b.WriteString(`</div><div class="matrix-pills">`)
	if m.CR != nil {
		crClass := "ok"
		if *m.CR > 0.10 {
			crClass = "bad"
		}
		fmt.Fprintf(b, `<span class="pill %s">CR %.3f</span>`, crClass, *m.CR)
	}
	if !m.Complete {
		b.WriteString(`<span class="pill warn">incomplete</span>`)
	}
	if len(m.IDs) > 0 {
		fmt.Fprintf(b,
			`<button type="button" class="matrix-expand" commandfor="%s" command="show-modal">Expand</button>`,
			html.EscapeString(dialogID),
		)
	}
	b.WriteString(`</div></header>`)
}

func writeMatrixJudgmentDialog(b *strings.Builder, mid, dialogID string, meta matrixMeta, m workspace.MatrixPayload) {
	fmt.Fprintf(b, `<dialog id="%s" class="matrix-dialog" closedby="any" aria-labelledby="%s-dialog-title">`, html.EscapeString(dialogID), mid)
	b.WriteString(`<div class="matrix-dialog-shell">`)
	b.WriteString(`<header class="matrix-dialog-head">`)
	b.WriteString(`<div class="matrix-dialog-titles">`)
	fmt.Fprintf(b, `<span class="pill kind">%s</span>`, html.EscapeString(meta.Badge))
	fmt.Fprintf(b, `<h3 id="%s-dialog-title">%s — judgments</h3>`, mid, html.EscapeString(meta.Title))
	if meta.Subtitle != "" {
		fmt.Fprintf(b, `<p class="muted matrix-sub">Saaty pairwise matrix that produced the local weights.</p>`)
	}
	b.WriteString(`</div>`)
	b.WriteString(`<form method="dialog"><button class="matrix-dialog-close" value="close" aria-label="Close matrix">Close</button></form>`)
	b.WriteString(`</header>`)
	b.WriteString(`<div class="matrix-dialog-body"><div class="matrix-wrap matrix-wrap-dialog">`)
	writeMatrixTable(b, m, true)
	b.WriteString(`</div></div></div></dialog>`)
}

func writeCriteriaWeightBars(b *strings.Builder, m workspace.MatrixPayload) {
	ids := append([]string(nil), m.IDs...)
	sort.SliceStable(ids, func(i, j int) bool {
		wi, wj := m.Weights[ids[i]], m.Weights[ids[j]]
		if wi != wj {
			return wi > wj
		}
		return ids[i] < ids[j]
	})

	b.WriteString(`<div class="crit-weights" role="img" aria-label="Criteria importance ranked by local weight">`)
	for i, id := range ids {
		wt := m.Weights[id]
		name := m.Names[id]
		if name == "" {
			name = id
		}
		pct := wt * 100
		if pct < 0 {
			pct = 0
		}
		rank := i + 1
		topClass := ""
		if rank == 1 {
			topClass = " is-top"
		}
		fmt.Fprintf(b, `<div class="crit-weight-row%s">
<span class="crit-weight-rank" aria-hidden="true">#%d</span>
<span class="crit-weight-name">%s</span>
<span class="crit-weight-pct">%.1f%%</span>
<div class="crit-weight-bar" title="local weight %.4f"><span style="width:%.1f%%"></span></div>
</div>`,
			topClass, rank, html.EscapeString(name), pct, wt, pct)
	}
	b.WriteString(`</div>`)
}

// writeMatrixTable emits one Saaty pairwise table. fullLabels uses longer header text (modal).
func writeMatrixTable(b *strings.Builder, m workspace.MatrixPayload, fullLabels bool) {
	b.WriteString(`<table class="matrix"><thead><tr><th class="corner"></th>`)
	for _, id := range m.IDs {
		label := m.Names[id]
		if !fullLabels {
			label = shortLabel(label)
		}
		b.WriteString(`<th scope="col" title="` + html.EscapeString(id) + `">` + html.EscapeString(label) + `</th>`)
	}
	b.WriteString(`</tr></thead><tbody>`)
	for i, rid := range m.IDs {
		rowLabel := m.Names[rid]
		if !fullLabels {
			rowLabel = shortLabel(rowLabel)
		}
		b.WriteString(`<tr><th scope="row" title="` + html.EscapeString(rid) + `">` + html.EscapeString(rowLabel) + `</th>`)
		for j := range m.IDs {
			cell := 0.0
			if i < len(m.Matrix) && j < len(m.Matrix[i]) {
				cell = m.Matrix[i][j]
			}
			cls := cellClass(cell, i, j)
			left := m.Names[rid]
			right := m.Names[m.IDs[j]]
			hint := fmtCell(cell)
			if i != j && cell > 0 {
				if cell > 1 {
					hint = fmt.Sprintf("%s prefers %s over %s (%s)", left, left, right, fmtCell(cell))
				} else if cell < 1 {
					hint = fmt.Sprintf("%s prefers %s over %s (%s)", left, right, left, fmtCell(cell))
				} else {
					hint = fmt.Sprintf("%s equal to %s", left, right)
				}
			}
			fmt.Fprintf(b, `<td class="num %s" title="%s"><span>%s</span></td>`,
				cls, html.EscapeString(hint), html.EscapeString(fmtCell(cell)))
		}
		b.WriteString(`</tr>`)
	}
	b.WriteString(`</tbody></table>`)
}

func sortedMatrixKeys(m map[string]workspace.MatrixPayload) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.SliceStable(keys, func(i, j int) bool {
		oi, oj := matrixSortOrder(keys[i]), matrixSortOrder(keys[j])
		if oi != oj {
			return oi < oj
		}
		return keys[i] < keys[j]
	})
	return keys
}

func matrixSortOrder(key string) int {
	switch {
	case key == "criteria":
		return 0
	case strings.HasPrefix(key, "criteria:"):
		return 1
	case strings.HasPrefix(key, "alt:"):
		return 2
	default:
		return 3
	}
}

func slug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	lastDash := false
	for _, r := range s {
		ok := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if ok {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func shortLabel(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return name
	}
	if len([]rune(name)) <= 22 {
		return name
	}
	r := []rune(name)
	return string(r[:20]) + "…"
}

func cellClass(v float64, i, j int) string {
	if i == j {
		return "diag"
	}
	if math.Abs(v-1) < 1e-9 || v <= 0 {
		return "eq"
	}
	strength := func(x float64) int {
		s := int(math.Round(x))
		if s < 2 {
			s = 2
		}
		if s > 9 {
			s = 9
		}
		return s
	}
	if v > 1 {
		return fmt.Sprintf("hi s%d", strength(v))
	}
	return fmt.Sprintf("lo s%d", strength(1/v))
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
	b.WriteString(`<div class="table-wrap"><table><thead><tr>`)
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
	b.WriteString(`</tbody></table></div>`)
}

func writePairwiseTable(b *strings.Builder, rows []workspace.PairwiseRow) {
	if len(rows) == 0 {
		b.WriteString(`<p class="muted">Empty.</p>`)
		return
	}
	b.WriteString(`<div class="table-wrap"><table><thead><tr><th>matrix</th><th>left</th><th>right</th><th>value</th><th>status</th><th>note</th></tr></thead><tbody>`)
	for _, r := range rows {
		fmt.Fprintf(b, `<tr><td>%s</td><td>%s</td><td>%s</td><td class="num">%s</td><td>%s</td><td>%s</td></tr>`,
			html.EscapeString(r.Matrix), html.EscapeString(r.Left), html.EscapeString(r.Right),
			html.EscapeString(engine.FormatSaaty(r.Value)), html.EscapeString(r.Status), html.EscapeString(r.Note))
	}
	b.WriteString(`</tbody></table></div>`)
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

func explainInputs(result *workspace.ComputeResult) (
	leaf map[string]float64,
	local map[string]map[string]float64,
	altIDs []string,
	altNames, critNames map[string]string,
) {
	leaf = result.LeafWeights
	if leaf == nil {
		leaf = map[string]float64{}
	}
	local = map[string]map[string]float64{}
	for cid := range leaf {
		key := "alt:" + cid
		if m, ok := result.Matrices[key]; ok {
			local[cid] = m.Weights
		} else {
			local[cid] = map[string]float64{}
		}
	}
	altNames = map[string]string{}
	altIDs = make([]string, 0, len(result.Alternatives))
	for _, a := range result.Alternatives {
		altIDs = append(altIDs, a.ID)
		altNames[a.ID] = a.Name
	}
	critNames = map[string]string{}
	for _, c := range result.Criteria {
		critNames[c.ID] = c.Name
	}
	return leaf, local, altIDs, altNames, critNames
}

func writeComparativeSection(b *strings.Builder, result *workspace.ComputeResult) {
	if result.Gaussian == nil && result.Absolute == nil {
		return
	}
	b.WriteString(`<section id="comparative"><h2 id="comparative-h">Absolute / Gaussian (comparative)</h2>
<p class="muted">Dispersion (σ/μ) is not decision-maker importance. These scores have <strong>no Saaty CR</strong>. Saaty ranking above remains canonical unless you explicitly choose a comparative mode.</p>`)
	if result.Absolute != nil {
		fmt.Fprintf(b, `<h3 id="absolute">Absolute / hybrid (%s)</h3>`, html.EscapeString(result.Absolute.Method))
		if len(result.Absolute.Ranking) > 0 {
			b.WriteString(`<div class="table-wrap"><table><thead><tr><th>Rank</th><th>Alternative</th><th class="num">Score</th></tr></thead><tbody>`)
			for _, r := range result.Absolute.Ranking {
				fmt.Fprintf(b, `<tr><td>%d</td><td>%s</td><td class="num">%.4f</td></tr>`,
					r.Rank, html.EscapeString(r.Name), r.Weight)
			}
			b.WriteString(`</tbody></table></div>`)
		} else if result.Absolute.Matrix != nil {
			fmt.Fprintf(b, `<p class="muted">Decision matrix ready (%d criteria × %d alternatives). Hybrid ranking unavailable — complete criteria pairwise with CR ≤ 0.10, or use <code>ahp gaussian</code>.</p>`,
				len(result.Absolute.Matrix.Columns), len(result.Absolute.Matrix.AlternativeIDs))
		}
	}
	if result.Gaussian != nil && len(result.Gaussian.Ranking) > 0 {
		b.WriteString(`<h3 id="gaussian">AHP-Gaussian</h3>`)
		if result.Gaussian.Result != nil && len(result.Gaussian.Result.Factors) > 0 {
			b.WriteString(`<div class="table-wrap"><table><thead><tr><th>Criterion</th><th class="num">μ</th><th class="num">σ</th><th class="num">f=σ/μ</th><th class="num">w</th></tr></thead><tbody>`)
			for _, f := range result.Gaussian.Result.Factors {
				fmt.Fprintf(b, `<tr><td>%s</td><td class="num">%.4f</td><td class="num">%.4f</td><td class="num">%.4f</td><td class="num">%.4f</td></tr>`,
					html.EscapeString(f.CriterionID), f.Mean, f.SD, f.Factor, f.Weight)
			}
			b.WriteString(`</tbody></table></div>`)
		}
		b.WriteString(`<div class="table-wrap"><table><thead><tr><th>Rank</th><th>Alternative</th><th class="num">Score</th></tr></thead><tbody>`)
		for _, r := range result.Gaussian.Ranking {
			fmt.Fprintf(b, `<tr><td>%d</td><td>%s</td><td class="num">%.4f</td></tr>`,
				r.Rank, html.EscapeString(r.Name), r.Weight)
		}
		b.WriteString(`</tbody></table></div>`)
	}
	b.WriteString(`</section>`)
}
