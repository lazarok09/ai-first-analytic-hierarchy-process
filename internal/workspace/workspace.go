package workspace

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/lazarok/ahp-method/internal/engine"
)

type Meta struct {
	Title       string
	Description string
}

type Criterion struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	ParentID    string `json:"parent_id"`
	Description string `json:"description"`
}

type Alternative struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type PairwiseRow struct {
	Matrix string  `json:"matrix"`
	Left   string  `json:"left"`
	Right  string  `json:"right"`
	Value  float64 `json:"value"`
	Status string  `json:"status"`
	Note   string  `json:"note"`
}

type AttributeRow struct {
	AlternativeID string `json:"alternative_id"`
	CriterionID   string `json:"criterion_id"`
	Value         string `json:"value"`
	Unit          string `json:"unit"`
	Source        string `json:"source"`
	Note          string `json:"note"`
}

type MissingPair struct {
	Matrix string `json:"matrix"`
	Left   string `json:"left"`
	Right  string `json:"right"`
}

type RepairRow struct {
	Matrix    string  `json:"matrix"`
	Left      string  `json:"left"`
	Right     string  `json:"right"`
	LeftName  string  `json:"left_name"`
	RightName string  `json:"right_name"`
	Current   float64 `json:"current"`
	Implied   float64 `json:"implied"`
	Suggested float64 `json:"suggested"`
	Error     float64 `json:"error"`
}

type MatrixPayload struct {
	IDs        []string           `json:"ids"`
	Names      map[string]string  `json:"names"`
	Weights    map[string]float64 `json:"weights"`
	LambdaMax  *float64           `json:"lambda_max"`
	CI         *float64           `json:"ci"`
	CR         *float64           `json:"cr"`
	Complete   bool               `json:"complete"`
	Consistent bool               `json:"consistent"`
	Missing    []map[string]string `json:"missing"`
	Matrix     [][]float64        `json:"matrix"`
	Repairs    []RepairRow        `json:"repairs"`
}

type RankRow struct {
	Rank   int     `json:"rank"`
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Weight float64 `json:"weight"`
}

type ComputeResult struct {
	Title            string                    `json:"title"`
	Description      string                    `json:"description"`
	Complete         bool                      `json:"complete"`
	Consistent       bool                      `json:"consistent"`
	IncludeProposals bool                      `json:"include_proposals"`
	Warnings         []string                  `json:"warnings"`
	Criteria         []Criterion               `json:"criteria"`
	Alternatives     []Alternative             `json:"alternatives"`
	Attributes       []AttributeRow            `json:"attributes"`
	Pairwise         []PairwiseRow             `json:"pairwise"`
	Matrices         map[string]MatrixPayload  `json:"matrices"`
	LeafWeights      map[string]float64        `json:"leaf_weights"`
	Ranking          []RankRow                 `json:"ranking"`
}

type StatusSummary struct {
	Workspace         string        `json:"workspace"`
	Title             string        `json:"title"`
	Criteria          int           `json:"criteria"`
	Alternatives      int           `json:"alternatives"`
	Attributes        int           `json:"attributes"`
	PairwiseCommitted int           `json:"pairwise_committed"`
	PairwiseProposals int           `json:"pairwise_proposals"`
	Complete          bool          `json:"complete"`
	Consistent        bool          `json:"consistent"`
	Missing           []MissingPair `json:"missing"`
	Repairs           []RepairRow   `json:"repairs"`
	Ranking           []RankRow     `json:"ranking"`
	Warnings          []string      `json:"warnings"`
	ReportHTML        string        `json:"report_html"`
	// Next is the single recommended follow-up (status v2 / agents).
	Next NextAction `json:"next"`
	// ExitCode is the readiness code for `ahp status --check` (0/2/3/4).
	ExitCode int `json:"exit_code"`
}

type Workspace struct {
	Root string
}

func Open(root string) *Workspace {
	return &Workspace{Root: filepath.Clean(root)}
}

func Find(start string) string {
	cur, err := filepath.Abs(start)
	if err != nil {
		return start
	}
	for {
		if _, err := os.Stat(filepath.Join(cur, "ahp.toml")); err == nil {
			return cur
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return start
		}
		cur = parent
	}
}

func (w *Workspace) TomlPath() string  { return filepath.Join(w.Root, "ahp.toml") }
func (w *Workspace) DataDir() string   { return filepath.Join(w.Root, "data") }
func (w *Workspace) OutputDir() string { return filepath.Join(w.Root, "output") }
func (w *Workspace) dataPath(name string) string {
	return filepath.Join(w.DataDir(), name)
}

func (w *Workspace) EnsureLayout() error {
	if err := os.MkdirAll(w.DataDir(), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(w.OutputDir(), 0o755); err != nil {
		return err
	}
	if _, err := os.Stat(w.TomlPath()); os.IsNotExist(err) {
		if err := w.SaveMeta(Meta{Title: "Untitled decision"}); err != nil {
			return err
		}
	}
	headers := map[string][]string{
		"criteria.csv":     {"id", "name", "parent_id", "description"},
		"alternatives.csv": {"id", "name", "description"},
		"pairwise.csv":     {"matrix", "left", "right", "value", "status", "note"},
		"attributes.csv":   {"alternative_id", "criterion_id", "value", "unit", "source", "note"},
	}
	for name, fields := range headers {
		path := w.dataPath(name)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			if err := writeCSV(path, fields, nil); err != nil {
				return err
			}
		}
	}
	return nil
}

func (w *Workspace) LoadMeta() (Meta, error) {
	b, err := os.ReadFile(w.TomlPath())
	if err != nil {
		if os.IsNotExist(err) {
			return Meta{Title: "Untitled decision"}, nil
		}
		return Meta{}, err
	}
	meta := Meta{Title: "Untitled decision"}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "title") {
			meta.Title = unquoteTOML(afterEq(line))
		}
		if strings.HasPrefix(line, "description") {
			meta.Description = unquoteTOML(afterEq(line))
		}
	}
	return meta, nil
}

func (w *Workspace) SaveMeta(m Meta) error {
	content := fmt.Sprintf("[decision]\ntitle = %s\ndescription = %s\n",
		quoteTOML(m.Title), quoteTOML(m.Description))
	return os.WriteFile(w.TomlPath(), []byte(content), 0o644)
}

func (w *Workspace) Criteria() ([]Criterion, error) {
	rows, err := readCSV(w.dataPath("criteria.csv"))
	if err != nil {
		return nil, err
	}
	var out []Criterion
	for _, r := range rows {
		if r["id"] == "" {
			continue
		}
		out = append(out, Criterion{
			ID: r["id"], Name: r["name"], ParentID: r["parent_id"], Description: r["description"],
		})
	}
	return out, nil
}

func (w *Workspace) Alternatives() ([]Alternative, error) {
	rows, err := readCSV(w.dataPath("alternatives.csv"))
	if err != nil {
		return nil, err
	}
	var out []Alternative
	for _, r := range rows {
		if r["id"] == "" {
			continue
		}
		out = append(out, Alternative{ID: r["id"], Name: r["name"], Description: r["description"]})
	}
	return out, nil
}

func (w *Workspace) Pairwise() ([]PairwiseRow, error) {
	rows, err := readCSV(w.dataPath("pairwise.csv"))
	if err != nil {
		return nil, err
	}
	var out []PairwiseRow
	for _, r := range rows {
		if r["left"] == "" || r["right"] == "" {
			continue
		}
		v, err := engine.ParseSaaty(r["value"])
		if err != nil {
			return nil, fmt.Errorf("pairwise value %q: %w", r["value"], err)
		}
		status := r["status"]
		if status == "" {
			status = "committed"
		}
		matrix := r["matrix"]
		if matrix == "" {
			matrix = "criteria"
		}
		out = append(out, PairwiseRow{
			Matrix: matrix, Left: r["left"], Right: r["right"],
			Value: v, Status: status, Note: r["note"],
		})
	}
	return out, nil
}

func (w *Workspace) Attributes() ([]AttributeRow, error) {
	rows, err := readCSV(w.dataPath("attributes.csv"))
	if err != nil {
		return nil, err
	}
	var out []AttributeRow
	for _, r := range rows {
		if r["alternative_id"] == "" || r["criterion_id"] == "" {
			continue
		}
		out = append(out, AttributeRow{
			AlternativeID: r["alternative_id"], CriterionID: r["criterion_id"],
			Value: r["value"], Unit: r["unit"], Source: r["source"], Note: r["note"],
		})
	}
	return out, nil
}

func (w *Workspace) WriteCriteria(items []Criterion) error {
	rows := make([]map[string]string, 0, len(items))
	for _, c := range items {
		rows = append(rows, map[string]string{
			"id": c.ID, "name": c.Name, "parent_id": c.ParentID, "description": c.Description,
		})
	}
	return writeCSV(w.dataPath("criteria.csv"), []string{"id", "name", "parent_id", "description"}, rows)
}

func (w *Workspace) WriteAlternatives(items []Alternative) error {
	rows := make([]map[string]string, 0, len(items))
	for _, a := range items {
		rows = append(rows, map[string]string{"id": a.ID, "name": a.Name, "description": a.Description})
	}
	return writeCSV(w.dataPath("alternatives.csv"), []string{"id", "name", "description"}, rows)
}

func (w *Workspace) WritePairwise(items []PairwiseRow) error {
	rows := make([]map[string]string, 0, len(items))
	for _, p := range items {
		rows = append(rows, map[string]string{
			"matrix": p.Matrix, "left": p.Left, "right": p.Right,
			"value": engine.FormatSaaty(p.Value), "status": p.Status, "note": p.Note,
		})
	}
	return writeCSV(w.dataPath("pairwise.csv"), []string{"matrix", "left", "right", "value", "status", "note"}, rows)
}

func (w *Workspace) WriteAttributes(items []AttributeRow) error {
	rows := make([]map[string]string, 0, len(items))
	for _, a := range items {
		rows = append(rows, map[string]string{
			"alternative_id": a.AlternativeID, "criterion_id": a.CriterionID,
			"value": a.Value, "unit": a.Unit, "source": a.Source, "note": a.Note,
		})
	}
	return writeCSV(w.dataPath("attributes.csv"),
		[]string{"alternative_id", "criterion_id", "value", "unit", "source", "note"}, rows)
}

func (w *Workspace) UpsertCriterion(item Criterion) error {
	items, err := w.Criteria()
	if err != nil {
		return err
	}
	by := map[string]Criterion{}
	for _, c := range items {
		by[c.ID] = c
	}
	by[item.ID] = item
	out := make([]Criterion, 0, len(by))
	for _, c := range by {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return w.WriteCriteria(out)
}

func (w *Workspace) UpsertAlternative(item Alternative) error {
	items, err := w.Alternatives()
	if err != nil {
		return err
	}
	by := map[string]Alternative{}
	for _, a := range items {
		by[a.ID] = a
	}
	by[item.ID] = item
	out := make([]Alternative, 0, len(by))
	for _, a := range by {
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return w.WriteAlternatives(out)
}

func (w *Workspace) UpsertAttribute(item AttributeRow) error {
	items, err := w.Attributes()
	if err != nil {
		return err
	}
	kept := items[:0]
	for _, a := range items {
		if a.AlternativeID == item.AlternativeID && a.CriterionID == item.CriterionID {
			continue
		}
		kept = append(kept, a)
	}
	kept = append(kept, item)
	return w.WriteAttributes(kept)
}

func (w *Workspace) UpsertPairwise(row PairwiseRow) error {
	if row.Left == row.Right {
		return fmt.Errorf("left and right must differ")
	}
	if !engine.IsSaatyValue(row.Value) {
		return fmt.Errorf("value must be Saaty 1-9 or reciprocal")
	}
	if row.Status != "proposal" && row.Status != "committed" {
		return fmt.Errorf("status must be proposal or committed")
	}
	items, err := w.Pairwise()
	if err != nil {
		return err
	}
	key := pairSet(row.Matrix, row.Left, row.Right)
	kept := items[:0]
	for _, existing := range items {
		if pairSet(existing.Matrix, existing.Left, existing.Right) != key {
			kept = append(kept, existing)
		}
	}
	kept = append(kept, row)
	return w.WritePairwise(kept)
}

func (w *Workspace) CommitProposals(matrix, left, right string) ([]PairwiseRow, error) {
	items, err := w.Pairwise()
	if err != nil {
		return nil, err
	}
	var updated []PairwiseRow
	for i := range items {
		if items[i].Status != "proposal" {
			continue
		}
		if matrix != "" && items[i].Matrix != matrix {
			continue
		}
		if left != "" && right != "" && !unorderedEqual(items[i].Left, items[i].Right, left, right) {
			continue
		}
		items[i].Status = "committed"
		updated = append(updated, items[i])
	}
	if len(updated) == 0 {
		return updated, nil
	}
	return updated, w.WritePairwise(items)
}

func (w *Workspace) Compute(includeProposals bool) (*ComputeResult, error) {
	meta, err := w.LoadMeta()
	if err != nil {
		return nil, err
	}
	criteria, err := w.Criteria()
	if err != nil {
		return nil, err
	}
	alternatives, err := w.Alternatives()
	if err != nil {
		return nil, err
	}
	pairwise, err := w.Pairwise()
	if err != nil {
		return nil, err
	}
	attributes, err := w.Attributes()
	if err != nil {
		return nil, err
	}

	var used []PairwiseRow
	var warnings []string
	var skipped int
	for _, p := range pairwise {
		if includeProposals || p.Status == "committed" {
			used = append(used, p)
		} else if p.Status == "proposal" {
			skipped++
		}
	}
	if includeProposals {
		warnings = append(warnings, "Including proposal judgments in compute (not human-committed).")
	}
	if skipped > 0 {
		warnings = append(warnings, fmt.Sprintf("%d proposal pair(s) ignored. Commit them in data/pairwise.csv or pass --include-proposals.", skipped))
	}

	var root []Criterion
	childByParent := map[string][]Criterion{}
	critNames := map[string]string{}
	for _, c := range criteria {
		critNames[c.ID] = c.Name
		if c.ParentID == "" {
			root = append(root, c)
		} else {
			childByParent[c.ParentID] = append(childByParent[c.ParentID], c)
		}
	}

	pairMap := groupPairs(used)
	matrices := map[string]MatrixPayload{}

	rootIDs := idsOf(root)
	critResult := engine.SolvePairwise(rootIDs, pairMap["criteria"])
	matrices["criteria"] = matrixPayload(critResult, critNames)
	if !critResult.Complete {
		warnings = append(warnings, "Criteria pairwise matrix is incomplete.")
	} else if critResult.CR > engine.CRAccept {
		warnings = append(warnings, fmt.Sprintf("Criteria CR=%.3f exceeds 0.10.", critResult.CR))
		for _, h := range matrices["criteria"].Repairs {
			if len(warnings) > 20 {
				break
			}
			warnings = append(warnings, fmt.Sprintf("  revisit criteria %s vs %s: %s → %s",
				h.Left, h.Right, engine.FormatSaaty(h.Current), engine.FormatSaaty(h.Suggested)))
		}
	}

	leafWeights := map[string]float64{}
	localWeights := map[string]map[string]float64{}

	if len(childByParent) > 0 {
		for _, parent := range root {
			children := childByParent[parent.ID]
			if len(children) == 0 {
				leafWeights[parent.ID] = critResult.Weights[parent.ID]
				continue
			}
			key := "criteria:" + parent.ID
			local := engine.SolvePairwise(idsOf(children), pairMap[key])
			matrices[key] = matrixPayload(local, critNames)
			pw := critResult.Weights[parent.ID]
			for cid, wt := range local.Weights {
				leafWeights[cid] = pw * wt
			}
			if !local.Complete {
				warnings = append(warnings, fmt.Sprintf("Sub-criteria under %s incomplete.", parent.ID))
			} else if local.CR > engine.CRAccept {
				warnings = append(warnings, fmt.Sprintf("Sub-criteria %s CR=%.3f exceeds 0.10.", parent.ID, local.CR))
			}
		}
		for _, parent := range root {
			if _, ok := childByParent[parent.ID]; !ok {
				leafWeights[parent.ID] = critResult.Weights[parent.ID]
			}
		}
	} else {
		leafWeights = critResult.Weights
	}

	altIDs := make([]string, 0, len(alternatives))
	altNames := map[string]string{}
	for _, a := range alternatives {
		altIDs = append(altIDs, a.ID)
		altNames[a.ID] = a.Name
	}
	for cid := range leafWeights {
		key := "alt:" + cid
		local := engine.SolvePairwise(altIDs, pairMap[key])
		matrices[key] = matrixPayload(local, altNames)
		localWeights[cid] = local.Weights
		if !local.Complete {
			warnings = append(warnings, fmt.Sprintf("Alternative pairwise under %s is incomplete.", cid))
		} else if local.CR > engine.CRAccept {
			warnings = append(warnings, fmt.Sprintf("Alternatives under %s CR=%.3f exceeds 0.10.", cid, local.CR))
		}
	}

	rankingWeights := engine.Synthesize(leafWeights, localWeights, altIDs)
	type kv struct {
		id string
		w  float64
	}
	var ranked []kv
	for id, wt := range rankingWeights {
		ranked = append(ranked, kv{id, wt})
	}
	sort.Slice(ranked, func(i, j int) bool { return ranked[i].w > ranked[j].w })
	ranking := make([]RankRow, 0, len(ranked))
	for i, r := range ranked {
		ranking = append(ranking, RankRow{Rank: i + 1, ID: r.id, Name: altNames[r.id], Weight: r.w})
	}

	complete := len(matrices) > 0
	consistent := complete
	for _, m := range matrices {
		if !m.Complete {
			complete = false
			consistent = false
		}
		if m.Complete && m.CR != nil && *m.CR > engine.CRAccept {
			consistent = false
		}
	}

	return &ComputeResult{
		Title: meta.Title, Description: meta.Description,
		Complete: complete, Consistent: consistent, IncludeProposals: includeProposals,
		Warnings: warnings, Criteria: criteria, Alternatives: alternatives,
		Attributes: attributes, Pairwise: pairwise, Matrices: matrices,
		LeafWeights: leafWeights, Ranking: ranking,
	}, nil
}

func (w *Workspace) WriteOutputs(result *ComputeResult, html string) (map[string]string, error) {
	if err := os.MkdirAll(w.OutputDir(), 0o755); err != nil {
		return nil, err
	}
	var weightRows []map[string]string
	for scope, m := range result.Matrices {
		for itemID, wt := range m.Weights {
			weightRows = append(weightRows, map[string]string{
				"scope": scope, "item_id": itemID, "name": m.Names[itemID],
				"weight": fmt.Sprintf("%.6f", wt),
				"lambda_max": fmtOpt(m.LambdaMax), "ci": fmtOpt(m.CI), "cr": fmtOpt(m.CR),
				"complete": strconv.FormatBool(m.Complete),
			})
		}
	}
	weightsPath := filepath.Join(w.OutputDir(), "weights.csv")
	rankingPath := filepath.Join(w.OutputDir(), "ranking.csv")
	jsonPath := filepath.Join(w.OutputDir(), "compute.json")
	htmlPath := filepath.Join(w.OutputDir(), "report.html")

	if err := writeCSV(weightsPath,
		[]string{"scope", "item_id", "name", "weight", "lambda_max", "ci", "cr", "complete"}, weightRows); err != nil {
		return nil, err
	}
	rankRows := make([]map[string]string, 0, len(result.Ranking))
	for _, r := range result.Ranking {
		rankRows = append(rankRows, map[string]string{
			"rank": strconv.Itoa(r.Rank), "alternative_id": r.ID, "name": r.Name,
			"global_weight": fmt.Sprintf("%.6f", r.Weight),
		})
	}
	if err := writeCSV(rankingPath, []string{"rank", "alternative_id", "name", "global_weight"}, rankRows); err != nil {
		return nil, err
	}
	jb, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(jsonPath, jb, 0o644); err != nil {
		return nil, err
	}
	if err := os.WriteFile(htmlPath, []byte(html), 0o644); err != nil {
		return nil, err
	}
	return map[string]string{
		"weights_csv": weightsPath, "ranking_csv": rankingPath,
		"compute_json": jsonPath, "report_html": htmlPath,
	}, nil
}

func (w *Workspace) Status(includeProposals bool) (*StatusSummary, error) {
	result, err := w.Compute(includeProposals)
	if err != nil {
		return nil, err
	}
	pairwise, err := w.Pairwise()
	if err != nil {
		return nil, err
	}
	committed, proposals := 0, 0
	for _, p := range pairwise {
		if p.Status == "proposal" {
			proposals++
		} else {
			committed++
		}
	}
	var missing []MissingPair
	var repairs []RepairRow
	for key, m := range result.Matrices {
		for _, p := range m.Missing {
			missing = append(missing, MissingPair{Matrix: key, Left: p["left"], Right: p["right"]})
		}
		for _, h := range m.Repairs {
			h.Matrix = key
			repairs = append(repairs, h)
		}
	}
	s := &StatusSummary{
		Workspace: w.Root, Title: result.Title,
		Criteria: len(result.Criteria), Alternatives: len(result.Alternatives),
		Attributes: len(result.Attributes),
		PairwiseCommitted: committed, PairwiseProposals: proposals,
		Complete: result.Complete, Consistent: result.Consistent,
		Missing: missing, Repairs: repairs, Ranking: result.Ranking,
		Warnings: result.Warnings, ReportHTML: filepath.Join(w.OutputDir(), "report.html"),
	}
	s.Next = NextActionFromStatus(s)
	s.ExitCode = ReadinessExit(s)
	return s, nil
}

func matrixPayload(r engine.MatrixResult, names map[string]string) MatrixPayload {
	nameMap := map[string]string{}
	for _, id := range r.IDs {
		if n, ok := names[id]; ok {
			nameMap[id] = n
		} else {
			nameMap[id] = id
		}
	}
	missing := make([]map[string]string, 0, len(r.Missing))
	for _, p := range r.Missing {
		missing = append(missing, map[string]string{"left": p.Left, "right": p.Right})
	}
	repairs := make([]RepairRow, 0, len(r.Repairs))
	for _, h := range r.Repairs {
		repairs = append(repairs, RepairRow{
			Left: h.Left, Right: h.Right,
			LeftName: nameMap[h.Left], RightName: nameMap[h.Right],
			Current: h.Current, Implied: h.Implied, Suggested: h.Suggested, Error: h.Error,
		})
	}
	var lam, ci, cr *float64
	if !math.IsNaN(r.LambdaMax) {
		v := r.LambdaMax
		lam = &v
	}
	if !math.IsNaN(r.CI) {
		v := r.CI
		ci = &v
	}
	if !math.IsNaN(r.CR) {
		v := r.CR
		cr = &v
	}
	return MatrixPayload{
		IDs: r.IDs, Names: nameMap, Weights: r.Weights,
		LambdaMax: lam, CI: ci, CR: cr,
		Complete: r.Complete, Consistent: r.Consistent(),
		Missing: missing, Matrix: r.Matrix, Repairs: repairs,
	}
}

func groupPairs(rows []PairwiseRow) map[string]map[engine.PairKey]float64 {
	out := map[string]map[engine.PairKey]float64{}
	for _, r := range rows {
		if out[r.Matrix] == nil {
			out[r.Matrix] = map[engine.PairKey]float64{}
		}
		out[r.Matrix][engine.PairKey{Left: r.Left, Right: r.Right}] = r.Value
	}
	return out
}

func idsOf(items []Criterion) []string {
	ids := make([]string, len(items))
	for i, c := range items {
		ids[i] = c.ID
	}
	return ids
}

func pairSet(matrix, left, right string) string {
	a, b := left, right
	if a > b {
		a, b = b, a
	}
	return matrix + "|" + a + "|" + b
}

func unorderedEqual(a1, a2, b1, b2 string) bool {
	return (a1 == b1 && a2 == b2) || (a1 == b2 && a2 == b1)
}

func fmtOpt(v *float64) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%.6f", *v)
}

func readCSV(path string) ([]map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.FieldsPerRecord = -1
	records, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, nil
	}
	header := records[0]
	var out []map[string]string
	for _, rec := range records[1:] {
		row := map[string]string{}
		for i, h := range header {
			val := ""
			if i < len(rec) {
				val = strings.TrimSpace(rec[i])
			}
			row[h] = val
		}
		out = append(out, row)
	}
	return out, nil
}

func writeCSV(path string, fields []string, rows []map[string]string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	if err := w.Write(fields); err != nil {
		return err
	}
	for _, row := range rows {
		rec := make([]string, len(fields))
		for i, f := range fields {
			rec[i] = row[f]
		}
		if err := w.Write(rec); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}

func afterEq(line string) string {
	i := strings.IndexByte(line, '=')
	if i < 0 {
		return ""
	}
	return strings.TrimSpace(line[i+1:])
}

func quoteTOML(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func unquoteTOML(s string) string {
	s = strings.TrimSpace(s)
	if u, err := strconv.Unquote(s); err == nil {
		return u
	}
	return strings.Trim(s, `"'`)
}
